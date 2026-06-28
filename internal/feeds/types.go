package feeds

import "net/http"

type Client struct {
	HTTP             *http.Client
	GitHubBase       string
	NewsBase         string
	ProductBase      string
	ProductWebBase   string
	RedditBase       string
	ArcticShiftBase  string
	RedditUserAgent  string
	V2EXBase         string
	ProductUserAgent string
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
	ID          int
	Title       string
	URL         string
	Author      string
	Score       int
	Descendants int
}

type NewsDiscussion struct {
	Item     NewsItem
	Comments []NewsComment
}

type NewsComment struct {
	ID       int
	ParentID int
	Author   string
	TextHTML string
	Text     string
	Deleted  bool
	Dead     bool
	Time     int64
	Depth    int
	Children []NewsComment
}

type Product struct {
	Name        string
	Description string
	URL         string
	Website     string
	Votes       int
	VotesKnown  bool
	Source      string
}

type ProductDetailsInput struct {
	URL  string
	Slug string
}

type ProductDetails struct {
	Slug           string
	ProductID      string
	PostID         string
	PostSlug       string
	Name           string
	LaunchName     string
	LaunchState    string
	LaunchNumber   int
	DailyRank      int
	WeeklyRank     int
	MonthlyRank    int
	Tagline        string
	Description    string
	ProductURL     string
	WebsiteURL     string
	CleanDomain    string
	LogoURL        string
	ThumbnailURL   string
	Makers         []ProductMaker
	Topics         []ProductTopic
	Media          []ProductMedia
	Votes          int
	VotesKnown     bool
	VotesHidden    bool
	CommentsCount  int
	PostsCount     int
	ReviewsCount   int
	ReviewsRating  float64
	FollowersCount int
	CreatedAt      string
	PublishedAt    string
	ScheduledAt    string
	UpdatedAt      string
	Source         string
}

type ProductCommentsInput struct {
	URL         string
	Slug        string
	Limit       int
	Depth       int
	IncludeHTML bool
}

type ProductComments struct {
	ProductName   string
	ProductURL    string
	CommentsCount int
	ShownComments int
	Complete      bool
	IncludeHTML   bool
	Comments      []ProductComment
	Source        string
}

type ProductComment struct {
	ID         string
	ParentID   string
	AuthorName string
	Username   string
	BodyText   string
	BodyHTML   string
	Votes      int
	CreatedAt  string
	Hidden     bool
	Deleted    bool
	Depth      int
	Replies    []ProductComment
}

type ProductMaker struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	URL      string `json:"url"`
	Headline string `json:"headline"`
	ImageURL string `json:"imageUrl"`
}

type ProductTopic struct {
	Name string
	Slug string
}

type ProductMedia struct {
	URL  string
	Type string
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
		HTTP:             http.DefaultClient,
		GitHubBase:       valueOrDefault(getenv("HFEEDS_GITHUB_BASE_URL"), "https://github.com"),
		NewsBase:         valueOrDefault(getenv("HFEEDS_HN_BASE_URL"), "https://hacker-news.firebaseio.com/v0"),
		ProductBase:      valueOrDefault(getenv("HFEEDS_PRODUCT_HUNT_BASE_URL"), defaultProductBase),
		ProductWebBase:   valueOrDefault(getenv("HFEEDS_PRODUCT_HUNT_WEB_BASE_URL"), defaultProductWebBase),
		RedditBase:       valueOrDefault(getenv("HFEEDS_REDDIT_BASE_URL"), defaultRedditBase),
		ArcticShiftBase:  valueOrDefault(getenv("HFEEDS_ARCTIC_SHIFT_BASE_URL"), defaultArcticShiftBase),
		RedditUserAgent:  getenv("HFEEDS_REDDIT_USER_AGENT"),
		V2EXBase:         valueOrDefault(getenv("HFEEDS_V2EX_BASE_URL"), "https://www.v2ex.com"),
		ProductUserAgent: getenv("HFEEDS_PRODUCT_HUNT_USER_AGENT"),
	}
	return client
}

func valueOrDefault(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
