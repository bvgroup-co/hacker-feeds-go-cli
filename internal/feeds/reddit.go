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
	"strings"
)

const defaultRedditUserAgent = "hfeeds/0.4.4 by bvgroup-co"

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string `json:"title"`
				Content   string `json:"selftext"`
				Comment   int    `json:"num_comments"`
				Permalink string `json:"permalink"`
				Votes     int    `json:"ups"`
				Topic     string `json:"subreddit"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditAtomFeed struct {
	Entries []redditAtomEntry `xml:"entry"`
}

type redditAtomEntry struct {
	Title      string         `xml:"title"`
	Content    string         `xml:"content"`
	Link       redditAtomLink `xml:"link"`
	Categories []struct {
		Term string `xml:"term,attr"`
	} `xml:"category"`
}

type redditAtomLink struct {
	Href string `xml:"href,attr"`
}

func ValidRedditSort(sort string) bool {
	return sort == "hot" || sort == "new" || sort == "top" || sort == "best"
}

func (client Client) FetchReddit(topic string, sort string) ([]RedditPost, error) {
	if !ValidRedditSort(sort) {
		return nil, fmt.Errorf("sort must be hot, new, top, or best")
	}
	posts, err := client.fetchRedditJSON(topic, sort)
	if err == nil {
		return posts, nil
	}
	if !shouldFallbackToRedditRSS(err) {
		return nil, err
	}
	rssPosts, rssErr := client.fetchRedditRSS(topic, sort)
	if rssErr == nil {
		return rssPosts, nil
	}
	return nil, fmt.Errorf("reddit rejected JSON (%v) and RSS fallback failed (%v). Reddit may be blocking this network or rate limiting unauthenticated requests. Try again later, set HFEEDS_REDDIT_USER_AGENT to a descriptive value, or run from another network; see README Reddit notes", err, rssErr)
}

func (client Client) fetchRedditJSON(topic string, sort string) ([]RedditPost, error) {
	base, err := url.Parse(client.RedditBase)
	if err != nil {
		return nil, err
	}
	base.Path = joinURLPath(base.Path, "r", topic, sort+".json")
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	client.setRedditHeaders(req, "application/json")
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	var decoded redditResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	posts := make([]RedditPost, 0, len(decoded.Data.Children))
	for _, child := range decoded.Data.Children {
		data := child.Data
		posts = append(posts, RedditPost{
			Title:   data.Title,
			Content: data.Content,
			Comment: data.Comment,
			Link:    strings.TrimRight(client.RedditBase, "/") + data.Permalink,
			Votes:   data.Votes,
			Topic:   data.Topic,
		})
	}
	return posts, nil
}

func (client Client) fetchRedditRSS(topic string, sort string) ([]RedditPost, error) {
	base, err := url.Parse(client.RedditBase)
	if err != nil {
		return nil, err
	}
	base.Path = joinURLPath(base.Path, "r", topic, sort+".rss")
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	client.setRedditHeaders(req, "application/atom+xml, application/rss+xml, text/xml")
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	var decoded redditAtomFeed
	if err := xml.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	posts := make([]RedditPost, 0, len(decoded.Entries))
	for _, entry := range decoded.Entries {
		posts = append(posts, RedditPost{
			Title:   strings.TrimSpace(entry.Title),
			Content: plainRedditContent(entry.Content),
			Link:    entry.Link.Href,
			Topic:   redditEntryTopic(entry, topic),
		})
	}
	return posts, nil
}

func (client Client) setRedditHeaders(req *http.Request, accept string) {
	req.Header.Set("User-Agent", valueOrDefault(client.RedditUserAgent, defaultRedditUserAgent))
	req.Header.Set("Accept", accept)
	req.Header.Set("Cache-Control", "no-cache")
}

func shouldFallbackToRedditRSS(err error) bool {
	var reqErr requestError
	if !errors.As(err, &reqErr) {
		return false
	}
	return reqErr.StatusCode == http.StatusForbidden || reqErr.StatusCode == http.StatusTooManyRequests
}

func redditEntryTopic(entry redditAtomEntry, fallback string) string {
	if len(entry.Categories) == 0 || strings.TrimSpace(entry.Categories[0].Term) == "" {
		return fallback
	}
	return strings.TrimSpace(entry.Categories[0].Term)
}

func plainRedditContent(content string) string {
	unescaped := html.UnescapeString(content)
	withoutTags := htmlTagPattern.ReplaceAllString(unescaped, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}
