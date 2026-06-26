package feeds

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRedditBase        = "https://www.reddit.com"
	defaultArcticShiftBase   = "https://arctic-shift.photon-reddit.com"
	defaultRedditUserAgent   = "hacker-feeds-go-cli/dev (+https://github.com/bvgroup-co/hacker-feeds-go-cli)"
	redditRSSSource          = "reddit-rss"
	redditShredditSource     = "reddit-shreddit"
	redditArcticShiftSource  = "arctic-shift"
	redditUnavailableMessage = "Reddit source unavailable without OAuth. Tried reddit-rss/reddit-shreddit and arctic-shift fallback."
)

var (
	redditHTMLTagPattern       = regexp.MustCompile(`<[^>]*>`)
	redditThingIDPattern       = regexp.MustCompile(`\bt3_[A-Za-z0-9]+\b`)
	redditShredditCommentRegex = regexp.MustCompile(`(?is)<shreddit-comment\b([^>]*)>(.*?)</shreddit-comment>`)
	redditSlotCommentRegex     = regexp.MustCompile(`(?is)<[^>]*\bslot=["']comment["'][^>]*>(.*?)</[^>]+>`)
	redditAttrRegex            = regexp.MustCompile(`(?is)([A-Za-z_:][-A-Za-z0-9_:]*)\s*=\s*("([^"]*)"|'([^']*)')`)
)

type redditAtomFeed struct {
	XMLName  xml.Name          `xml:"feed"`
	Title    string            `xml:"title"`
	Updated  string            `xml:"updated"`
	Entries  []redditAtomEntry `xml:"entry"`
	Category struct {
		Term string `xml:"term,attr"`
	} `xml:"category"`
}

type redditAtomEntry struct {
	ID        string `xml:"id"`
	Title     string `xml:"title"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Author    struct {
		Name string `xml:"name"`
		URI  string `xml:"uri"`
	} `xml:"author"`
	Link struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Content struct {
		Body string `xml:",innerxml"`
	} `xml:"content"`
	Category struct {
		Term string `xml:"term,attr"`
	} `xml:"category"`
}

type arcticShiftPostsResponse struct {
	Data []arcticShiftPost `json:"data"`
}

type arcticShiftPost struct {
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

type arcticShiftCommentsResponse struct {
	Data []arcticShiftComment `json:"data"`
}

type arcticShiftComment struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	LinkID     string  `json:"link_id"`
	ParentID   string  `json:"parent_id"`
	Author     string  `json:"author"`
	Body       string  `json:"body"`
	Score      int     `json:"score"`
	Permalink  string  `json:"permalink"`
	CreatedUTC float64 `json:"created_utc"`
}

func ValidRedditSort(sort string) bool {
	return sort == "hot" || sort == "new" || sort == "top" || sort == "best"
}

func ValidRedditCommentSort(sort string) bool {
	return sort == "confidence" || sort == "top" || sort == "new" || sort == "controversial" || sort == "old" || sort == "qa"
}

func (client *Client) FetchReddit(topic string, sortValue string, limit int) ([]RedditPost, error) {
	if !ValidRedditSort(sortValue) {
		return nil, fmt.Errorf("sort must be hot, new, top, or best")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}
	posts, rssErr := client.fetchRedditRSS(topic, limit)
	if rssErr == nil {
		return posts, nil
	}
	posts, arcticErr := client.fetchArcticShiftPosts(topic, limit)
	if arcticErr == nil {
		return posts, nil
	}
	return nil, redditCombinedError(rssErr, arcticErr)
}

func (client *Client) FetchRedditComments(topic string, postID string, limit int, depth int, sortValue string) (RedditDiscussion, error) {
	if strings.TrimSpace(postID) == "" {
		return RedditDiscussion{}, fmt.Errorf("--post is required")
	}
	if limit <= 0 {
		return RedditDiscussion{}, fmt.Errorf("limit must be greater than 0")
	}
	if depth <= 0 {
		return RedditDiscussion{}, fmt.Errorf("depth must be greater than 0")
	}
	if !ValidRedditCommentSort(sortValue) {
		return RedditDiscussion{}, fmt.Errorf("comment sort must be confidence, top, new, controversial, old, or qa")
	}
	discussion, shredErr := client.fetchShredditComments(topic, postID, limit, depth)
	if shredErr == nil {
		return discussion, nil
	}
	discussion, arcticErr := client.fetchArcticShiftComments(topic, postID, limit)
	if arcticErr == nil {
		return discussion, nil
	}
	return RedditDiscussion{}, redditCombinedError(shredErr, arcticErr)
}

func (client *Client) fetchRedditRSS(topic string, limit int) ([]RedditPost, error) {
	base, err := url.Parse(client.RedditBase)
	if err != nil {
		return nil, err
	}
	base.Path = joinURLPath(base.Path, "r", topic, ".rss")
	base.RawQuery = url.Values{"limit": []string{strconv.Itoa(limit)}}.Encode()
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", client.redditUserAgent())
	req.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/xml, text/xml")
	body, err := client.do(req)
	if err != nil {
		return nil, redditSourceError{Source: redditRSSSource, Err: err}
	}
	posts, err := parseRedditAtom(body, topic)
	if err != nil {
		return nil, redditSourceError{Source: redditRSSSource, Err: err}
	}
	return posts, nil
}

func (client *Client) fetchArcticShiftPosts(topic string, limit int) ([]RedditPost, error) {
	base, err := url.Parse(client.ArcticShiftBase)
	if err != nil {
		return nil, err
	}
	base.Path = joinURLPath(base.Path, "api", "posts", "search")
	base.RawQuery = url.Values{
		"subreddit": []string{topic},
		"limit":     []string{strconv.Itoa(limit)},
		"sort":      []string{"desc"},
		"sort_type": []string{"created_utc"},
	}.Encode()
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", client.redditUserAgent())
	req.Header.Set("Accept", "application/json")
	body, err := client.do(req)
	if err != nil {
		return nil, redditSourceError{Source: redditArcticShiftSource, Err: err}
	}
	posts, err := parseArcticShiftPosts(body)
	if err != nil {
		return nil, redditSourceError{Source: redditArcticShiftSource, Err: err}
	}
	return posts, nil
}

func (client *Client) fetchShredditComments(topic string, postID string, limit int, depth int) (RedditDiscussion, error) {
	base, err := url.Parse(client.RedditBase)
	if err != nil {
		return RedditDiscussion{}, err
	}
	base.Path = joinURLPath(base.Path, "svc", "shreddit", "comments", "r", topic, fullPostID(postID))
	base.RawQuery = url.Values{"limit": []string{strconv.Itoa(limit)}, "depth": []string{strconv.Itoa(depth)}}.Encode()
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return RedditDiscussion{}, err
	}
	req.Header.Set("User-Agent", client.redditUserAgent())
	req.Header.Set("Accept", "text/html, */*;q=0.5")
	body, err := client.do(req)
	if err != nil {
		return RedditDiscussion{}, redditSourceError{Source: redditShredditSource, Err: err}
	}
	comments, err := parseShredditComments(body)
	if err != nil {
		return RedditDiscussion{}, redditSourceError{Source: redditShredditSource, Err: err}
	}
	return RedditDiscussion{Post: RedditPost{ID: trimThingPrefix(postID), Name: fullPostID(postID), Subreddit: topic, Source: redditShredditSource}, Comments: comments}, nil
}

func (client *Client) fetchArcticShiftComments(topic string, postID string, limit int) (RedditDiscussion, error) {
	base, err := url.Parse(client.ArcticShiftBase)
	if err != nil {
		return RedditDiscussion{}, err
	}
	base.Path = joinURLPath(base.Path, "api", "comments", "search")
	base.RawQuery = url.Values{"link_id": []string{fullPostID(postID)}, "limit": []string{strconv.Itoa(limit)}}.Encode()
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return RedditDiscussion{}, err
	}
	req.Header.Set("User-Agent", client.redditUserAgent())
	req.Header.Set("Accept", "application/json")
	body, err := client.do(req)
	if err != nil {
		return RedditDiscussion{}, redditSourceError{Source: redditArcticShiftSource, Err: err}
	}
	comments, err := parseArcticShiftComments(body)
	if err != nil {
		return RedditDiscussion{}, redditSourceError{Source: redditArcticShiftSource, Err: err}
	}
	return RedditDiscussion{Post: RedditPost{ID: trimThingPrefix(postID), Name: fullPostID(postID), Subreddit: topic, Source: redditArcticShiftSource}, Comments: comments}, nil
}

func parseRedditAtom(body []byte, fallbackTopic string) ([]RedditPost, error) {
	var feed redditAtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	if len(feed.Entries) == 0 {
		return []RedditPost{}, nil
	}
	feedTopic := redditFeedTopic(feed, fallbackTopic)
	posts := make([]RedditPost, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		postID := redditPostID(entry.ID)
		if postID == "" {
			postID = redditPostID(entry.Link.Href)
		}
		posts = append(posts, RedditPost{
			ID:         postID,
			Name:       thingName("t3", postID),
			Title:      strings.TrimSpace(html.UnescapeString(entry.Title)),
			Content:    plainRedditHTML(entry.Content.Body),
			Permalink:  entry.Link.Href,
			Subreddit:  redditEntryTopic(entry, feedTopic),
			Author:     strings.TrimSpace(entry.Author.Name),
			AuthorURI:  strings.TrimSpace(entry.Author.URI),
			CreatedUTC: parseRedditTime(entry.Published),
			UpdatedUTC: parseRedditTime(entry.Updated),
			Source:     redditRSSSource,
		})
	}
	return posts, nil
}

func parseArcticShiftPosts(body []byte) ([]RedditPost, error) {
	var decoded arcticShiftPostsResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	posts := make([]RedditPost, 0, len(decoded.Data))
	for _, item := range decoded.Data {
		postID := trimThingPrefix(valueOrDefault(item.ID, item.Name))
		posts = append(posts, RedditPost{
			ID:          postID,
			Name:        thingName("t3", postID),
			Title:       item.Title,
			Content:     item.SelfText,
			URL:         item.URL,
			Permalink:   absoluteRedditPermalink(item.Permalink),
			Subreddit:   item.Subreddit,
			Author:      item.Author,
			Score:       item.Score,
			Ups:         item.Ups,
			NumComments: item.NumComments,
			CreatedUTC:  int64(item.CreatedUTC),
			IsSelf:      item.IsSelf,
			Domain:      item.Domain,
			Source:      redditArcticShiftSource,
		})
	}
	return posts, nil
}

func parseShredditComments(body []byte) ([]RedditComment, error) {
	matches := redditShredditCommentRegex.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("reddit-shreddit response did not contain comments")
	}
	comments := make([]RedditComment, 0, len(matches))
	for _, match := range matches {
		attrs := parseHTMLAttrs(string(match[1]))
		bodyText := shredCommentBody(match[2])
		comments = append(comments, RedditComment{
			ID:         trimThingPrefix(firstAttr(attrs, "thingid", "thing-id")),
			Name:       firstAttr(attrs, "thingid", "thing-id"),
			ParentID:   firstAttr(attrs, "parentid", "parent-id"),
			PostID:     firstAttr(attrs, "postid", "post-id"),
			Author:     firstAttr(attrs, "author"),
			Body:       bodyText,
			Score:      parseLeadingInt(firstAttr(attrs, "score")),
			CreatedUTC: parseRedditTime(firstAttr(attrs, "created")),
			Permalink:  firstAttr(attrs, "permalink"),
			Depth:      parseLeadingInt(firstAttr(attrs, "depth")),
			Source:     redditShredditSource,
		})
	}
	return buildRedditCommentTree(comments), nil
}

func parseArcticShiftComments(body []byte) ([]RedditComment, error) {
	var decoded arcticShiftCommentsResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	comments := make([]RedditComment, 0, len(decoded.Data))
	for _, item := range decoded.Data {
		commentID := trimThingPrefix(valueOrDefault(item.ID, item.Name))
		comments = append(comments, RedditComment{
			ID:         commentID,
			Name:       thingName("t1", commentID),
			ParentID:   item.ParentID,
			PostID:     item.LinkID,
			Author:     item.Author,
			Body:       item.Body,
			Score:      item.Score,
			CreatedUTC: int64(item.CreatedUTC),
			Permalink:  absoluteRedditPermalink(item.Permalink),
			Source:     redditArcticShiftSource,
		})
	}
	return buildRedditCommentTree(comments), nil
}

func buildRedditCommentTree(flat []RedditComment) []RedditComment {
	comments := make([]RedditComment, len(flat))
	copy(comments, flat)
	byName := make(map[string]*RedditComment, len(comments))
	for index := range comments {
		if comments[index].Name != "" {
			byName[comments[index].Name] = &comments[index]
		}
		if comments[index].ID != "" {
			byName[thingName("t1", comments[index].ID)] = &comments[index]
		}
	}
	childIndexes := make(map[int]bool, len(comments))
	for index := range comments {
		parentID := comments[index].ParentID
		if !strings.HasPrefix(parentID, "t1_") {
			continue
		}
		parent := byName[parentID]
		if parent == nil {
			continue
		}
		parent.Replies = append(parent.Replies, comments[index])
		childIndexes[index] = true
	}
	roots := make([]RedditComment, 0, len(comments))
	for index := range comments {
		if !childIndexes[index] {
			roots = append(roots, comments[index])
		}
	}
	sortRedditComments(roots)
	return roots
}

func sortRedditComments(comments []RedditComment) {
	sort.SliceStable(comments, func(left int, right int) bool {
		return comments[left].CreatedUTC < comments[right].CreatedUTC
	})
	for index := range comments {
		sortRedditComments(comments[index].Replies)
	}
}

func redditFeedTopic(feed redditAtomFeed, fallback string) string {
	if strings.TrimSpace(feed.Category.Term) != "" {
		return strings.TrimSpace(feed.Category.Term)
	}
	if strings.Contains(feed.Title, "/r/") {
		parts := strings.Split(feed.Title, "/r/")
		return strings.Fields(parts[len(parts)-1])[0]
	}
	return fallback
}

func redditEntryTopic(entry redditAtomEntry, fallback string) string {
	if strings.TrimSpace(entry.Category.Term) != "" {
		return strings.TrimSpace(entry.Category.Term)
	}
	return fallback
}

func redditPostID(value string) string {
	match := redditThingIDPattern.FindString(value)
	return trimThingPrefix(match)
}

func shredCommentBody(inner []byte) string {
	match := redditSlotCommentRegex.FindSubmatch(inner)
	if len(match) == 0 {
		return plainRedditHTML(string(inner))
	}
	return plainRedditHTML(string(match[1]))
}

func parseHTMLAttrs(raw string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range redditAttrRegex.FindAllStringSubmatch(raw, -1) {
		value := match[3]
		if value == "" {
			value = match[4]
		}
		attrs[strings.ToLower(match[1])] = html.UnescapeString(value)
	}
	return attrs
}

func firstAttr(attrs map[string]string, names ...string) string {
	for _, name := range names {
		if attrs[strings.ToLower(name)] != "" {
			return attrs[strings.ToLower(name)]
		}
	}
	return ""
}

func plainRedditHTML(content string) string {
	unescaped := html.UnescapeString(content)
	withoutTags := redditHTMLTagPattern.ReplaceAllString(unescaped, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}

func parseRedditTime(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if numeric, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return int64(numeric)
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}

func parseLeadingInt(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	fields := strings.Fields(trimmed)
	parsed, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return parsed
}

func trimThingPrefix(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "t1_") || strings.HasPrefix(trimmed, "t3_") {
		return trimmed[3:]
	}
	return trimmed
}

func fullPostID(postID string) string {
	trimmed := strings.TrimSpace(postID)
	if strings.HasPrefix(trimmed, "t3_") {
		return trimmed
	}
	return "t3_" + trimmed
}

func thingName(prefix string, id string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, prefix+"_") {
		return trimmed
	}
	return prefix + "_" + trimmed
}

func absoluteRedditPermalink(permalink string) string {
	if permalink == "" {
		return ""
	}
	if strings.HasPrefix(permalink, "http://") || strings.HasPrefix(permalink, "https://") {
		return permalink
	}
	return "https://www.reddit.com" + permalink
}

func (client Client) redditUserAgent() string {
	return valueOrDefault(client.RedditUserAgent, defaultRedditUserAgent)
}

type redditSourceError struct {
	Source string
	Err    error
}

func (err redditSourceError) Error() string {
	return err.Source + ": " + redditErrorMessage(err.Err)
}

func (err redditSourceError) Unwrap() error {
	return err.Err
}

func redditCombinedError(primary error, fallback error) error {
	return fmt.Errorf("%s %s failed; %s failed", redditUnavailableMessage, redditErrorMessage(primary), redditErrorMessage(fallback))
}

func redditErrorMessage(err error) string {
	var sourceErr redditSourceError
	if errors.As(err, &sourceErr) {
		return sourceErr.Error()
	}
	var reqErr requestError
	if errors.As(err, &reqErr) {
		message := fmt.Sprintf("status %d", reqErr.StatusCode)
		if reqErr.RetryAfter != "" {
			message += "; retry after " + reqErr.RetryAfter
		}
		if reqErr.Body != "" {
			message += "; response: " + reqErr.Body
		}
		return message
	}
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}
