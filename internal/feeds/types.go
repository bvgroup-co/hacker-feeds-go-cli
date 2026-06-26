package feeds

import "net/http"

type Client struct {
	HTTP            *http.Client
	GitHubBase      string
	NewsBase        string
	ProductBase     string
	RedditBase      string
	ArcticShiftBase string
	RedditUserAgent string
	V2EXBase        string
	ProductToken    string
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
	AuthorURI   string
	Score       int
	Ups         int
	NumComments int
	CreatedUTC  int64
	UpdatedUTC  int64
	IsSelf      bool
	Domain      string
	Source      string
}

type RedditDiscussion struct {
	Post     RedditPost
	Comments []RedditComment
}

type RedditComment struct {
	ID         string
	Name       string
	ParentID   string
	PostID     string
	Author     string
	Body       string
	Score      int
	CreatedUTC int64
	Permalink  string
	Depth      int
	Source     string
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
		RedditBase:      valueOrDefault(getenv("HFEEDS_REDDIT_BASE_URL"), defaultRedditBase),
		ArcticShiftBase: valueOrDefault(getenv("HFEEDS_ARCTIC_SHIFT_BASE_URL"), defaultArcticShiftBase),
		RedditUserAgent: getenv("HFEEDS_REDDIT_USER_AGENT"),
		V2EXBase:        valueOrDefault(getenv("HFEEDS_V2EX_BASE_URL"), "https://www.v2ex.com"),
		ProductToken:    getenv("PRODUCT_HUNT_ACCESS_TOKEN"),
	}
	return client
}

func valueOrDefault(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
