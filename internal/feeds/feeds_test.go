package feeds

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchGitHub(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/trending/go" {
			t.Fatalf("path = %s", request.URL.Path)
		}
		if request.URL.RawQuery != "since=weekly&spoken_language_code=" {
			t.Fatalf("query = %s", request.URL.RawQuery)
		}
		_, _ = writer.Write([]byte(`<article class="Box-row"><h2 class="h3"><a href="/owner/repo"> owner / repo </a></h2><p class="my-1">Useful repo</p><span itemprop="programmingLanguage">Go</span><a><svg aria-label="star"></svg> 1,234</a><span class="float-sm-right">56 stars this week</span></article>`))
	}))
	defer server.Close()

	items, err := (Client{HTTP: server.Client(), GitHubBase: server.URL}).FetchGitHub("go", "weekly")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Repo != "repo" || items[0].AddedStars != 56 || items[0].Stars != 1234 {
		t.Fatalf("items = %#v", items)
	}
}

func TestFetchNews(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/topstories.json":
			_, _ = writer.Write([]byte(`[1,2,3]`))
		case "/item/1.json":
			_, _ = writer.Write([]byte(`{"id":1,"title":"One","url":"https://example.com/one","by":"alice","score":42,"descendants":3}`))
		case "/item/2.json":
			_, _ = writer.Write([]byte(`{"id":2,"title":"Two","by":"bob","score":7,"descendants":0}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	items, err := (Client{HTTP: server.Client(), NewsBase: server.URL}).FetchNews(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[0].Author != "alice" || items[0].Score != 42 || items[0].Descendants != 3 || items[1].URL != "" {
		t.Fatalf("items = %#v", items)
	}
}

func TestFetchNewsDiscussion(t *testing.T) {
	requests := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.URL.Path)
		switch request.URL.Path {
		case "/item/1.json":
			_, _ = writer.Write([]byte(`{"id":1,"type":"story","by":"alice","title":"Story","url":"https://example.com/story","score":99,"descendants":4,"kids":[2,3,4]}`))
		case "/item/2.json":
			_, _ = writer.Write([]byte(`{"id":2,"type":"comment","by":"bob","parent":1,"text":"First &amp; <b>bold</b>","kids":[5]}`))
		case "/item/3.json":
			_, _ = writer.Write([]byte(`{"id":3,"type":"comment","deleted":true,"parent":1}`))
		case "/item/4.json":
			_, _ = writer.Write([]byte(`{"id":4,"type":"comment","dead":true,"parent":1,"text":"dead text"}`))
		case "/item/5.json":
			_, _ = writer.Write([]byte(`{"id":5,"type":"comment","by":"carol","parent":2,"text":"Nested<p>reply"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	discussion, err := (Client{HTTP: server.Client(), NewsBase: server.URL}).FetchNewsDiscussion(1, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if discussion.Item.ID != 1 || discussion.Item.Author != "alice" || len(discussion.Comments) != 3 {
		t.Fatalf("discussion = %#v", discussion)
	}
	if discussion.Comments[0].Text != "First & bold" || len(discussion.Comments[0].Children) != 1 || discussion.Comments[0].Children[0].Depth != 1 {
		t.Fatalf("comments = %#v", discussion.Comments)
	}
	if !discussion.Comments[1].Deleted || !discussion.Comments[2].Dead {
		t.Fatalf("deleted/dead = %#v", discussion.Comments)
	}
	if strings.Join(requests, ",") != "/item/1.json,/item/2.json,/item/5.json,/item/3.json,/item/4.json" {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestFetchNewsDiscussionLimitAndDepth(t *testing.T) {
	requests := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.URL.Path)
		switch request.URL.Path {
		case "/item/1.json":
			_, _ = writer.Write([]byte(`{"id":1,"type":"story","kids":[2,3]}`))
		case "/item/2.json":
			_, _ = writer.Write([]byte(`{"id":2,"type":"comment","parent":1,"text":"first","kids":[4]}`))
		case "/item/3.json":
			_, _ = writer.Write([]byte(`{"id":3,"type":"comment","parent":1,"text":"second"}`))
		case "/item/4.json":
			_, _ = writer.Write([]byte(`{"id":4,"type":"comment","parent":2,"text":"nested"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	discussion, err := (Client{HTTP: server.Client(), NewsBase: server.URL}).FetchNewsDiscussion(1, 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(discussion.Comments) != 1 || discussion.Comments[0].ID != 2 || len(discussion.Comments[0].Children) != 0 {
		t.Fatalf("limit discussion = %#v", discussion)
	}
	requests = requests[:0]
	discussion, err = (Client{HTTP: server.Client(), NewsBase: server.URL}).FetchNewsDiscussion(1, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(discussion.Comments) != 2 || len(discussion.Comments[0].Children) != 0 || strings.Contains(strings.Join(requests, ","), "/item/4.json") {
		t.Fatalf("depth discussion = %#v requests=%#v", discussion, requests)
	}
}

func TestFetchProducts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s", request.Method)
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %s", request.Header.Get("Authorization"))
		}
		body := readRequestBody(t, request)
		for _, fragment := range []string{"first: 2", `postedAfter: \"2026-06-25\"`, `postedBefore: \"2026-06-27\"`} {
			if !strings.Contains(body, fragment) {
				t.Fatalf("body missing %s: %s", fragment, body)
			}
		}
		_, _ = writer.Write([]byte(`{"data":{"posts":{"edges":[{"node":{"name":"Prod","description":"Desc","url":"https://p.example/path?ref=x","website":"https://w.example/?a=b","votesCount":7}}]}}}`))
	}))
	defer server.Close()

	items, err := (Client{HTTP: server.Client(), ProductBase: server.URL, ProductToken: "token"}).FetchProducts(2, 1, time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].URL != "https://p.example/path" || items[0].Website != "https://w.example/" {
		t.Fatalf("items = %#v", items)
	}
}

func TestFetchProductsRequiresToken(t *testing.T) {
	_, err := (Client{}).FetchProducts(1, 0, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchV2EX(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/topics/show.json" || request.URL.Query().Get("node_name") != "programmer" {
			t.Fatalf("url = %s", request.URL.String())
		}
		_, _ = writer.Write([]byte(`[{"title":"Topic","content":"Body","replies":4,"url":"https://v2ex.example/t/1","votes":6,"node":{"name":"programmer"}}]`))
	}))
	defer server.Close()

	topics, err := (Client{HTTP: server.Client(), V2EXBase: server.URL}).FetchV2EX("programmer")
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 || topics[0].Node != "programmer" || topics[0].Votes != 6 {
		t.Fatalf("topics = %#v", topics)
	}
}

func readRequestBody(t *testing.T, request *http.Request) string {
	t.Helper()
	body := make([]byte, request.ContentLength)
	_, err := request.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		t.Fatal(err)
	}
	return string(body)
}

func TestFetchRedditUsesRSSFirst(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if request.URL.Path != "/r/golang/.rss" {
			http.NotFound(writer, request)
			return
		}
		if request.URL.Query().Get("limit") != "2" || !strings.Contains(request.Header.Get("Accept"), "application/atom+xml") || request.Header.Get("User-Agent") == "" {
			t.Fatalf("request = %s headers=%#v", request.URL.String(), request.Header)
		}
		_, _ = writer.Write([]byte(redditRSSFixture()))
	}))
	defer server.Close()

	posts, err := (&Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}).FetchReddit("golang", 2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/r/golang/.rss" {
		t.Fatalf("paths = %#v", paths)
	}
	if len(posts) != 1 || posts[0].Source != "reddit-rss" || posts[0].ID != "abc123" || posts[0].Title != "Post & Title" || posts[0].Author != "alice" || posts[0].AuthorURI != "https://www.reddit.com/user/alice" || posts[0].Subreddit != "golang" || posts[0].Content != "Body & text" || posts[0].CreatedUTC != 1770004800 || posts[0].UpdatedUTC != 1770004860 {
		t.Fatalf("posts = %#v", posts)
	}
}

func TestFetchRedditFallsBackToArcticShiftPosts(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		switch request.URL.Path {
		case "/r/golang/.rss":
			http.Error(writer, "blocked", http.StatusForbidden)
		case "/api/posts/search":
			if request.URL.Query().Get("subreddit") != "golang" || request.URL.Query().Get("limit") != "3" || request.URL.Query().Get("sort") != "desc" || request.URL.Query().Get("sort_type") != "created_utc" {
				t.Fatalf("query = %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"data":[{"id":"def456","name":"t3_def456","title":"Fallback","selftext":"Self text","url":"https://example.com","permalink":"/r/golang/comments/def456/fallback/","subreddit":"golang","author":"bob","score":12,"ups":10,"num_comments":4,"created_utc":1770000100,"is_self":false,"domain":"example.com"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	posts, err := (&Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}).FetchReddit("golang", 3)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/r/golang/.rss,/api/posts/search" {
		t.Fatalf("paths = %#v", paths)
	}
	if len(posts) != 1 || posts[0].Source != "arctic-shift" || posts[0].ID != "def456" || posts[0].Score != 12 || posts[0].NumComments != 4 || posts[0].Permalink != "https://www.reddit.com/r/golang/comments/def456/fallback/" {
		t.Fatalf("posts = %#v", posts)
	}
}

func TestFetchRedditFallsBackOnInvalidRSS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/r/golang/.rss":
			_, _ = writer.Write([]byte(`not xml`))
		case "/api/posts/search":
			_, _ = writer.Write([]byte(`{"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	posts, err := (&Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}).FetchReddit("golang", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 0 {
		t.Fatalf("posts = %#v", posts)
	}
}

func TestFetchRedditCommentsUsesShredditFirst(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if request.URL.Path != "/svc/shreddit/comments/r/golang/t3_abc123" {
			http.NotFound(writer, request)
			return
		}
		if request.URL.Query().Get("limit") != "10" || request.URL.Query().Get("depth") != "2" || !strings.Contains(request.Header.Get("Accept"), "text/html") {
			t.Fatalf("request = %s headers=%#v", request.URL.String(), request.Header)
		}
		_, _ = writer.Write([]byte(shredditFixture()))
	}))
	defer server.Close()

	discussion, err := (&Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}).FetchRedditComments("golang", "abc123", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/svc/shreddit/comments/r/golang/t3_abc123" {
		t.Fatalf("paths = %#v", paths)
	}
	if discussion.Post.Source != "reddit-shreddit" || len(discussion.Comments) != 1 {
		t.Fatalf("discussion = %#v", discussion)
	}
	root := discussion.Comments[0]
	if root.ID != "c1" || root.Name != "t1_c1" || root.Author != "alice" || root.Body != "Root & body" || root.Score != 7 || root.Depth != 0 || root.CreatedUTC != 1770005000 || len(root.Replies) != 1 {
		t.Fatalf("root = %#v", root)
	}
	if root.Replies[0].ID != "c2" || root.Replies[0].ParentID != "t1_c1" || root.Replies[0].Body != "Reply body" || root.Replies[0].Depth != 1 {
		t.Fatalf("reply = %#v", root.Replies[0])
	}
}

func TestFetchRedditCommentsFallsBackToArcticShift(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		switch request.URL.Path {
		case "/svc/shreddit/comments/r/golang/t3_abc123":
			_, _ = writer.Write([]byte(`<html><title>verification required</title></html>`))
		case "/api/comments/search":
			if request.URL.Query().Get("link_id") != "t3_abc123" || request.URL.Query().Get("limit") != "10" {
				t.Fatalf("query = %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(arcticCommentsFixture()))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	discussion, err := (&Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}).FetchRedditComments("golang", "abc123", 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/svc/shreddit/comments/r/golang/t3_abc123,/api/comments/search" {
		t.Fatalf("paths = %#v", paths)
	}
	if discussion.Post.Source != "arctic-shift" || len(discussion.Comments) != 1 || len(discussion.Comments[0].Replies) != 1 {
		t.Fatalf("discussion = %#v", discussion)
	}
	if discussion.Comments[0].Source != "arctic-shift" || discussion.Comments[0].Replies[0].ParentID != "t1_c1" {
		t.Fatalf("comments = %#v", discussion.Comments)
	}
}

func TestBuildRedditCommentTreePreservesMultipleLevels(t *testing.T) {
	comments := buildRedditCommentTree([]RedditComment{
		{ID: "c1", Name: "t1_c1", ParentID: "t3_abc", Body: "root", CreatedUTC: 1},
		{ID: "c2", Name: "t1_c2", ParentID: "t1_c1", Body: "child", CreatedUTC: 2},
		{ID: "c3", Name: "t1_c3", ParentID: "t1_c2", Body: "grandchild", CreatedUTC: 3},
		{ID: "c4", Name: "t1_c4", ParentID: "t1_c3", Body: "great-grandchild", CreatedUTC: 4},
	})
	if len(comments) != 1 || len(comments[0].Replies) != 1 || len(comments[0].Replies[0].Replies) != 1 || len(comments[0].Replies[0].Replies[0].Replies) != 1 {
		t.Fatalf("comments = %#v", comments)
	}
	leaf := comments[0].Replies[0].Replies[0].Replies[0]
	if leaf.ID != "c4" || leaf.Body != "great-grandchild" {
		t.Fatalf("leaf = %#v", leaf)
	}
}

func TestFetchRedditNoOAuthOrJSON(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if strings.Contains(request.URL.Path, "access_token") || strings.HasSuffix(request.URL.Path, ".json") || request.Header.Get("Authorization") != "" {
			t.Fatalf("unexpected oauth/json request path=%s headers=%#v", request.URL.Path, request.Header)
		}
		switch request.URL.Path {
		case "/r/golang/.rss":
			_, _ = writer.Write([]byte(redditRSSFixture()))
		case "/svc/shreddit/comments/r/golang/t3_abc123":
			_, _ = writer.Write([]byte(shredditFixture()))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := &Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}
	if _, err := client.FetchReddit("golang", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := client.FetchRedditComments("golang", "abc123", 10, 2); err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/r/golang/.rss,/svc/shreddit/comments/r/golang/t3_abc123" {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestFetchRedditAllSourcesUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Retry-After", "30")
		http.Error(writer, "blocked html body that should be short", http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := (&Client{HTTP: server.Client(), RedditBase: server.URL, ArcticShiftBase: server.URL}).FetchReddit("golang", 1)
	if err == nil || !strings.Contains(err.Error(), "Reddit source unavailable without OAuth") || !strings.Contains(err.Error(), "retry after 30") {
		t.Fatalf("err = %v", err)
	}
}

func redditRSSFixture() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>/r/golang</title>
  <category term="golang"></category>
  <entry>
    <id>https://www.reddit.com/r/golang/comments/t3_abc123/post/</id>
    <title>Post &amp;amp; Title</title>
    <author><name>alice</name><uri>https://www.reddit.com/user/alice</uri></author>
    <link href="https://www.reddit.com/r/golang/comments/abc123/post/" />
    <published>2026-02-02T04:00:00Z</published>
    <updated>2026-02-02T04:01:00Z</updated>
    <content type="html">&lt;p&gt;Body &amp;amp; text&lt;/p&gt;</content>
    <category term="golang"></category>
  </entry>
</feed>`
}

func shredditFixture() string {
	return `<shreddit-comment thingId="t1_c1" author="alice" created="2026-02-02T04:03:20Z" depth="0" parentId="t3_abc123" permalink="/r/golang/comments/abc123/comment/c1/" score="7" postId="t3_abc123"><div slot="comment"><p>Root &amp;amp; body</p></div></shreddit-comment><shreddit-comment thingId="t1_c2" author="bob" created="2026-02-02T04:04:20Z" depth="1" parentId="t1_c1" permalink="/r/golang/comments/abc123/comment/c2/" score="2" postId="t3_abc123"><div slot="comment"><p>Reply body</p></div></shreddit-comment>`
}

func arcticCommentsFixture() string {
	return `{"data":[{"id":"c1","name":"t1_c1","link_id":"t3_abc123","parent_id":"t3_abc123","author":"alice","body":"Root","score":4,"permalink":"/r/golang/comments/abc123/comment/c1/","created_utc":1770000300},{"id":"c2","name":"t1_c2","link_id":"t3_abc123","parent_id":"t1_c1","author":"bob","body":"Reply","score":2,"permalink":"/r/golang/comments/abc123/comment/c2/","created_utc":1770000360}]}`
}
