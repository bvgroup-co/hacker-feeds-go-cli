package feeds

import (
	"net/http"
	"time"
)

type Client struct {
	HTTP            *http.Client
	GitHubBase      string
	NewsBase        string
	ProductBase     string
	RedditOAuthBase string
	RedditTokenURL  string
	RedditClientID  string
	RedditDeviceID  string
	RedditUserAgent string
	V2EXBase        string
	ProductToken    string
	Now             func() time.Time
	redditToken     redditToken
}

type GitHubRepo struct {
	Author     string
	Repo       string
	Link       string
	Desc       string
	Language   string
	Stars      int
	AddedStars int
}

type NewsItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Product struct {
	Name        string
	Description string
	URL         string
	Website     string
	Votes       int
}

type RedditPost struct {
	ID          string
	Name        string
	Title       string
	Content     string
	URL         string
	Permalink   string
	Subreddit   string
	Author      string
	Score       int
	Ups         int
	NumComments int
	CreatedUTC  int64
	IsSelf      bool
	Domain      string
}

type RedditDiscussion struct {
	Post     RedditPost
	Comments []RedditComment
}

type RedditComment struct {
	ID         string
	Name       string
	Author     string
	Body       string
	Score      int
	CreatedUTC int64
	Permalink  string
	Replies    []RedditComment
	More       bool
	Count      int
}

type V2EXTopic struct {
	Title   string
	Content string
	Comment int
	Link    string
	Votes   int
	Node    string
}

func NewClientFromEnv(getenv func(string) string) Client {
	client := Client{
		HTTP:            http.DefaultClient,
		GitHubBase:      valueOrDefault(getenv("HFEEDS_GITHUB_BASE_URL"), "https://github.com"),
		NewsBase:        valueOrDefault(getenv("HFEEDS_HN_BASE_URL"), "https://hacker-news.firebaseio.com/v0"),
		ProductBase:     valueOrDefault(getenv("HFEEDS_PRODUCT_HUNT_BASE_URL"), "https://api.producthunt.com/v2/api/graphql/"),
		RedditOAuthBase: valueOrDefault(getenv("HFEEDS_REDDIT_OAUTH_BASE_URL"), defaultRedditOAuthBase),
		RedditTokenURL:  valueOrDefault(getenv("HFEEDS_REDDIT_TOKEN_URL"), defaultRedditTokenURL),
		RedditClientID:  getenv("HFEEDS_REDDIT_CLIENT_ID"),
		RedditDeviceID:  getenv("HFEEDS_REDDIT_DEVICE_ID"),
		RedditUserAgent: getenv("HFEEDS_REDDIT_USER_AGENT"),
		V2EXBase:        valueOrDefault(getenv("HFEEDS_V2EX_BASE_URL"), "https://www.v2ex.com"),
		ProductToken:    getenv("PRODUCT_HUNT_ACCESS_TOKEN"),
		Now:             time.Now,
	}
	return client
}

func valueOrDefault(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
