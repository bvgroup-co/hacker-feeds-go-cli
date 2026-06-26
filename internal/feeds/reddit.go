package feeds

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRedditOAuthBase = "https://oauth.reddit.com"
	defaultRedditTokenURL  = "https://www.reddit.com/api/v1/access_token"
	redditInstalledGrant   = "https://oauth.reddit.com/grants/installed_client"
	redditTokenRefreshSkew = time.Minute
)

var errMissingRedditOAuthConfig = errors.New("Reddit OAuth is required. Set HFEEDS_REDDIT_CLIENT_ID, HFEEDS_REDDIT_DEVICE_ID, and HFEEDS_REDDIT_USER_AGENT.")

type redditToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

type redditTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type redditListingResponse struct {
	Data struct {
		Children []redditThing `json:"children"`
	} `json:"data"`
}

type redditListingArray []redditListingResponse

type redditThing struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type redditPostData struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Title       string  `json:"title"`
	SelfText    string  `json:"selftext"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Subreddit   string  `json:"subreddit"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	Ups         int     `json:"ups"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	IsSelf      bool    `json:"is_self"`
	Domain      string  `json:"domain"`
}

type redditCommentData struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Author     string          `json:"author"`
	Body       string          `json:"body"`
	Score      int             `json:"score"`
	CreatedUTC float64         `json:"created_utc"`
	Permalink  string          `json:"permalink"`
	Replies    json.RawMessage `json:"replies"`
	Count      int             `json:"count"`
}

func ValidRedditSort(sort string) bool {
	return sort == "hot" || sort == "new" || sort == "top" || sort == "best"
}

func ValidRedditCommentSort(sort string) bool {
	switch sort {
	case "confidence", "top", "new", "controversial", "old", "qa":
		return true
	default:
		return false
	}
}

func (client *Client) FetchReddit(topic string, sort string, limit int) ([]RedditPost, error) {
	if !ValidRedditSort(sort) {
		return nil, fmt.Errorf("sort must be hot, new, top, or best")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}
	if err := client.validateRedditOAuthConfig(); err != nil {
		return nil, err
	}
	body, err := client.redditAPI(http.MethodGet, strings.Join([]string{"r", topic, sort}, "/"), url.Values{
		"limit":    []string{strconv.Itoa(limit)},
		"raw_json": []string{"1"},
	}, nil)
	if err != nil {
		return nil, err
	}
	var decoded redditListingResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	posts := make([]RedditPost, 0, len(decoded.Data.Children))
	for _, child := range decoded.Data.Children {
		post, err := client.decodeRedditPost(child)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (client *Client) FetchRedditComments(topic string, postID string, limit int, depth int, sort string) (RedditDiscussion, error) {
	if strings.TrimSpace(postID) == "" {
		return RedditDiscussion{}, fmt.Errorf("--post is required")
	}
	if limit <= 0 {
		return RedditDiscussion{}, fmt.Errorf("limit must be greater than 0")
	}
	if depth <= 0 {
		return RedditDiscussion{}, fmt.Errorf("depth must be greater than 0")
	}
	if !ValidRedditCommentSort(sort) {
		return RedditDiscussion{}, fmt.Errorf("comment sort must be confidence, top, new, controversial, old, or qa")
	}
	if err := client.validateRedditOAuthConfig(); err != nil {
		return RedditDiscussion{}, err
	}
	body, err := client.redditAPI(http.MethodGet, strings.Join([]string{"r", topic, "comments", postID}, "/"), url.Values{
		"limit":    []string{strconv.Itoa(limit)},
		"depth":    []string{strconv.Itoa(depth)},
		"sort":     []string{sort},
		"raw_json": []string{"1"},
	}, nil)
	if err != nil {
		return RedditDiscussion{}, err
	}
	var decoded redditListingArray
	if err := json.Unmarshal(body, &decoded); err != nil {
		return RedditDiscussion{}, err
	}
	if len(decoded) != 2 {
		return RedditDiscussion{}, fmt.Errorf("reddit comments response must contain post and comment listings")
	}
	discussion := RedditDiscussion{}
	if len(decoded[0].Data.Children) > 0 {
		post, err := client.decodeRedditPost(decoded[0].Data.Children[0])
		if err != nil {
			return RedditDiscussion{}, err
		}
		discussion.Post = post
	}
	comments, err := client.decodeRedditComments(decoded[1].Data.Children)
	if err != nil {
		return RedditDiscussion{}, err
	}
	discussion.Comments = comments
	return discussion, nil
}

func (client *Client) validateRedditOAuthConfig() error {
	if strings.TrimSpace(client.RedditClientID) == "" || strings.TrimSpace(client.RedditDeviceID) == "" || strings.TrimSpace(client.RedditUserAgent) == "" {
		return errMissingRedditOAuthConfig
	}
	return nil
}

func (client *Client) redditAPI(method string, path string, query url.Values, body io.Reader) ([]byte, error) {
	token, err := client.redditAccessToken()
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(client.RedditOAuthBase)
	if err != nil {
		return nil, err
	}
	base.Path = joinURLPath(base.Path, path)
	base.RawQuery = query.Encode()
	req, err := http.NewRequest(method, base.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("User-Agent", client.RedditUserAgent)
	req.Header.Set("Accept", "application/json")
	response, err := client.do(req)
	if err != nil {
		return nil, redditActionableError("api", err)
	}
	return response, nil
}

func (client *Client) redditAccessToken() (redditToken, error) {
	now := time.Now()
	if client.Now != nil {
		now = client.Now()
	}
	if client.redditToken.AccessToken != "" && now.Add(redditTokenRefreshSkew).Before(client.redditToken.ExpiresAt) {
		return client.redditToken, nil
	}
	form := url.Values{}
	form.Set("grant_type", redditInstalledGrant)
	form.Set("device_id", client.RedditDeviceID)
	req, err := http.NewRequest(http.MethodPost, client.RedditTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return redditToken{}, err
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(client.RedditClientID+":")))
	req.Header.Set("User-Agent", client.RedditUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	body, err := client.do(req)
	if err != nil {
		return redditToken{}, redditActionableError("token", err)
	}
	var decoded redditTokenResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return redditToken{}, err
	}
	if decoded.AccessToken == "" {
		return redditToken{}, fmt.Errorf("reddit token response did not include access_token")
	}
	if !strings.EqualFold(decoded.TokenType, "bearer") {
		return redditToken{}, fmt.Errorf("reddit token response used unsupported token_type %q", decoded.TokenType)
	}
	client.redditToken = redditToken{
		AccessToken: decoded.AccessToken,
		TokenType:   decoded.TokenType,
		ExpiresAt:   now.Add(time.Duration(decoded.ExpiresIn) * time.Second),
	}
	return client.redditToken, nil
}

func (client *Client) decodeRedditPost(thing redditThing) (RedditPost, error) {
	if thing.Kind != "t3" {
		return RedditPost{}, fmt.Errorf("reddit listing included unsupported kind %q", thing.Kind)
	}
	var data redditPostData
	if err := json.Unmarshal(thing.Data, &data); err != nil {
		return RedditPost{}, err
	}
	permalink := absoluteRedditPermalink(client.RedditOAuthBase, data.Permalink)
	return RedditPost{
		ID:          data.ID,
		Name:        data.Name,
		Title:       data.Title,
		Content:     data.SelfText,
		URL:         data.URL,
		Permalink:   permalink,
		Subreddit:   data.Subreddit,
		Author:      data.Author,
		Score:       data.Score,
		Ups:         data.Ups,
		NumComments: data.NumComments,
		CreatedUTC:  int64(data.CreatedUTC),
		IsSelf:      data.IsSelf,
		Domain:      data.Domain,
		Comment:     data.NumComments,
		Link:        permalink,
		Votes:       data.Ups,
		Topic:       data.Subreddit,
	}, nil
}

func (client *Client) decodeRedditComments(things []redditThing) ([]RedditComment, error) {
	comments := make([]RedditComment, 0, len(things))
	for _, thing := range things {
		comment, err := client.decodeRedditComment(thing)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (client *Client) decodeRedditComment(thing redditThing) (RedditComment, error) {
	var data redditCommentData
	if err := json.Unmarshal(thing.Data, &data); err != nil {
		return RedditComment{}, err
	}
	switch thing.Kind {
	case "t1":
		replies, err := client.decodeRedditReplies(data.Replies)
		if err != nil {
			return RedditComment{}, err
		}
		return RedditComment{
			ID:         data.ID,
			Name:       data.Name,
			Author:     data.Author,
			Body:       data.Body,
			Score:      data.Score,
			CreatedUTC: int64(data.CreatedUTC),
			Permalink:  absoluteRedditPermalink(client.RedditOAuthBase, data.Permalink),
			Replies:    replies,
		}, nil
	case "more":
		return RedditComment{More: true, ID: data.ID, Name: data.Name, Count: data.Count}, nil
	default:
		return RedditComment{}, fmt.Errorf("reddit comments included unsupported kind %q", thing.Kind)
	}
}

func (client *Client) decodeRedditReplies(raw json.RawMessage) ([]RedditComment, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte(`""`)) {
		return nil, nil
	}
	var listing redditListingResponse
	if err := json.Unmarshal(raw, &listing); err != nil {
		return nil, err
	}
	return client.decodeRedditComments(listing.Data.Children)
}

func redditActionableError(context string, err error) error {
	var reqErr requestError
	if !errors.As(err, &reqErr) {
		return err
	}
	prefix := "reddit API request failed"
	if context == "token" {
		prefix = "reddit OAuth token request failed"
	}
	message := redditStatusMessage(reqErr.StatusCode)
	if reqErr.RetryAfter != "" {
		message += "; retry after " + reqErr.RetryAfter
	}
	if reqErr.Body != "" {
		message += "; response: " + reqErr.Body
	}
	return fmt.Errorf("%s: %s", prefix, message)
}

func redditStatusMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "400 invalid grant/device ID/request"
	case http.StatusUnauthorized:
		return "401 invalid Reddit client ID/auth header/token"
	case http.StatusForbidden:
		return "403 Reddit API forbidden; check app setup/scopes/user-agent"
	case http.StatusNotFound:
		return "404 subreddit/post not found"
	case http.StatusTooManyRequests:
		return "429 rate limited"
	default:
		if statusCode >= 500 {
			return fmt.Sprintf("%d Reddit server error", statusCode)
		}
		return fmt.Sprintf("request failed with status %d", statusCode)
	}
}

func absoluteRedditPermalink(base string, permalink string) string {
	if permalink == "" {
		return ""
	}
	if strings.HasPrefix(permalink, "http://") || strings.HasPrefix(permalink, "https://") {
		return permalink
	}
	parsed, err := url.Parse(base)
	if err != nil {
		panic("invalid reddit oauth base: " + err.Error())
	}
	return parsed.Scheme + "://www.reddit.com" + permalink
}
