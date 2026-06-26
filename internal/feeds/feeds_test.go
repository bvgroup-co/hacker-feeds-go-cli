package feeds

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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
			_, _ = writer.Write([]byte(`{"title":"One","url":"https://example.com/one"}`))
		case "/item/2.json":
			_, _ = writer.Write([]byte(`{"title":"Two"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	items, err := (Client{HTTP: server.Client(), NewsBase: server.URL}).FetchNews(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Title != "One" || items[1].URL != "" {
		t.Fatalf("items = %#v", items)
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

func TestFetchReddit(t *testing.T) {
	tokenCalls := 0
	apiCalls := 0
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			tokenCalls++
			assertRedditTokenRequest(t, request, "client", "device", "agent")
			_, _ = writer.Write([]byte(`{"access_token":"token","token_type":"bearer","expires_in":3600,"scope":"*"}`))
		case "/r/golang/top":
			apiCalls++
			if request.URL.Query().Get("limit") != "2" || request.URL.Query().Get("raw_json") != "1" {
				t.Fatalf("query = %s", request.URL.RawQuery)
			}
			if request.Header.Get("Authorization") != "Bearer token" || request.Header.Get("User-Agent") != "agent" || request.Header.Get("Accept") != "application/json" {
				t.Fatalf("headers = %#v", request.Header)
			}
			_, _ = writer.Write([]byte(`{"data":{"children":[{"kind":"t3","data":{"id":"1","name":"t3_1","title":"Post","selftext":"Body","url":"https://example.com/post","permalink":"/r/golang/comments/1/post/","subreddit":"golang","author":"alice","score":12,"ups":9,"num_comments":3,"created_utc":1770000000,"is_self":false,"domain":"example.com"}}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := &Client{
		HTTP:            server.Client(),
		RedditOAuthBase: server.URL,
		RedditTokenURL:  server.URL + "/token",
		RedditClientID:  "client",
		RedditDeviceID:  "device",
		RedditUserAgent: "agent",
		Now: func() time.Time {
			return now
		},
	}
	posts, err := client.FetchReddit("golang", "top", 2)
	if err != nil {
		t.Fatal(err)
	}
	if tokenCalls != 1 || apiCalls != 1 {
		t.Fatalf("tokenCalls=%d apiCalls=%d", tokenCalls, apiCalls)
	}
	if len(posts) != 1 || posts[0].Permalink != "http://www.reddit.com/r/golang/comments/1/post/" || posts[0].URL != "https://example.com/post" || posts[0].Author != "alice" || posts[0].Score != 12 || posts[0].NumComments != 3 || posts[0].Domain != "example.com" {
		t.Fatalf("posts = %#v", posts)
	}
	if _, err := client.FetchReddit("golang", "top", 2); err != nil {
		t.Fatal(err)
	}
	if tokenCalls != 1 || apiCalls != 2 {
		t.Fatalf("cached tokenCalls=%d apiCalls=%d", tokenCalls, apiCalls)
	}
}

func TestFetchRedditRequiresOAuthConfigBeforeNetwork(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
	}))
	defer server.Close()

	client := Client{HTTP: server.Client(), RedditOAuthBase: server.URL, RedditTokenURL: server.URL + "/token"}
	_, err := client.FetchReddit("golang", "top", 10)
	if err == nil || err.Error() != "Reddit OAuth is required. Set HFEEDS_REDDIT_CLIENT_ID, HFEEDS_REDDIT_DEVICE_ID, and HFEEDS_REDDIT_USER_AGENT." {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("network was called")
	}
}

func TestFetchRedditRefreshesTokenNearExpiry(t *testing.T) {
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	tokenCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			tokenCalls++
			_, _ = writer.Write([]byte(`{"access_token":"token` + strconv.Itoa(tokenCalls) + `","token_type":"bearer","expires_in":61,"scope":"*"}`))
		case "/r/golang/hot":
			_, _ = writer.Write([]byte(`{"data":{"children":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := &Client{
		HTTP:            server.Client(),
		RedditOAuthBase: server.URL,
		RedditTokenURL:  server.URL + "/token",
		RedditClientID:  "client",
		RedditDeviceID:  "device",
		RedditUserAgent: "agent",
		Now: func() time.Time {
			return now
		},
	}
	if _, err := client.FetchReddit("golang", "hot", 10); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if _, err := client.FetchReddit("golang", "hot", 10); err != nil {
		t.Fatal(err)
	}
	if tokenCalls != 2 {
		t.Fatalf("tokenCalls = %d", tokenCalls)
	}
}

func TestFetchRedditComments(t *testing.T) {
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			_, _ = writer.Write([]byte(`{"access_token":"token","token_type":"bearer","expires_in":3600,"scope":"*"}`))
		case "/r/golang/comments/abc":
			if request.URL.Query().Get("limit") != "10" || request.URL.Query().Get("depth") != "2" || request.URL.Query().Get("sort") != "top" || request.URL.Query().Get("raw_json") != "1" {
				t.Fatalf("query = %s", request.URL.RawQuery)
			}
			if request.Header.Get("Authorization") != "Bearer token" {
				t.Fatalf("authorization = %s", request.Header.Get("Authorization"))
			}
			_, _ = writer.Write([]byte(`[{"data":{"children":[{"kind":"t3","data":{"id":"abc","name":"t3_abc","title":"Post","permalink":"/r/golang/comments/abc/post/","subreddit":"golang","author":"alice","score":20,"ups":18,"num_comments":2}}]}},{"data":{"children":[{"kind":"t1","data":{"id":"c1","name":"t1_c1","author":"bob","body":"First","score":5,"created_utc":1770000001,"permalink":"/r/golang/comments/abc/comment/c1/","replies":{"data":{"children":[{"kind":"t1","data":{"id":"c2","name":"t1_c2","author":"cal","body":"Reply","score":2,"created_utc":1770000002,"permalink":"/r/golang/comments/abc/comment/c2/","replies":""}}]}}}},{"kind":"more","data":{"id":"more","name":"t1_more","count":4}}]}}]`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := redditTestClient(server, now)
	discussion, err := client.FetchRedditComments("golang", "abc", 10, 2, "top")
	if err != nil {
		t.Fatal(err)
	}
	if discussion.Post.ID != "abc" || len(discussion.Comments) != 2 {
		t.Fatalf("discussion = %#v", discussion)
	}
	if discussion.Comments[0].Author != "bob" || len(discussion.Comments[0].Replies) != 1 || discussion.Comments[0].Replies[0].Body != "Reply" {
		t.Fatalf("comments = %#v", discussion.Comments)
	}
	if !discussion.Comments[1].More || discussion.Comments[1].Count != 4 {
		t.Fatalf("more = %#v", discussion.Comments[1])
	}
}

func TestFetchRedditErrorMapping(t *testing.T) {
	cases := []struct {
		status     int
		retryAfter string
		fragment   string
	}{
		{status: http.StatusBadRequest, fragment: "400 invalid grant/device ID/request"},
		{status: http.StatusUnauthorized, fragment: "401 invalid Reddit client ID/auth header/token"},
		{status: http.StatusForbidden, fragment: "403 Reddit API forbidden; check app setup/scopes/user-agent"},
		{status: http.StatusNotFound, fragment: "404 subreddit/post not found"},
		{status: http.StatusTooManyRequests, retryAfter: "30", fragment: "429 rate limited; retry after 30"},
		{status: http.StatusInternalServerError, fragment: "500 Reddit server error"},
	}
	for _, testCase := range cases {
		t.Run(strconv.Itoa(testCase.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/token" {
					_, _ = writer.Write([]byte(`{"access_token":"token","token_type":"bearer","expires_in":3600,"scope":"*"}`))
					return
				}
				if testCase.retryAfter != "" {
					writer.Header().Set("Retry-After", testCase.retryAfter)
				}
				writer.WriteHeader(testCase.status)
				_, _ = writer.Write([]byte(`{"message":"bad thing"}`))
			}))
			defer server.Close()

			client := redditTestClient(server, time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC))
			_, err := client.FetchReddit("golang", "hot", 10)
			if err == nil || !strings.Contains(err.Error(), testCase.fragment) || !strings.Contains(err.Error(), "bad thing") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestFetchRedditUsesNoRSSOrJSONFallbackPath(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if strings.HasSuffix(request.URL.Path, ".rss") || strings.HasSuffix(request.URL.Path, ".json") {
			t.Fatalf("unexpected fallback path %s", request.URL.Path)
		}
		if request.URL.Path == "/token" {
			_, _ = writer.Write([]byte(`{"access_token":"token","token_type":"bearer","expires_in":3600,"scope":"*"}`))
			return
		}
		_, _ = writer.Write([]byte(`{"data":{"children":[]}}`))
	}))
	defer server.Close()

	client := redditTestClient(server, time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC))
	if _, err := client.FetchReddit("golang", "top", 10); err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/token,/r/golang/top" {
		t.Fatalf("paths = %#v", paths)
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

func assertRedditTokenRequest(t *testing.T, request *http.Request, clientID string, deviceID string, userAgent string) {
	t.Helper()
	if request.Method != http.MethodPost {
		t.Fatalf("method = %s", request.Method)
	}
	if request.Header.Get("Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":")) {
		t.Fatalf("authorization = %s", request.Header.Get("Authorization"))
	}
	if request.Header.Get("User-Agent") != userAgent || request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || request.Header.Get("Accept") != "application/json" {
		t.Fatalf("headers = %#v", request.Header)
	}
	body := readRequestBody(t, request)
	values, err := url.ParseQuery(body)
	if err != nil {
		t.Fatal(err)
	}
	if values.Get("grant_type") != redditInstalledGrant || values.Get("device_id") != deviceID {
		t.Fatalf("form = %s", body)
	}
}

func redditTestClient(server *httptest.Server, now time.Time) *Client {
	return &Client{
		HTTP:            server.Client(),
		RedditOAuthBase: server.URL,
		RedditTokenURL:  server.URL + "/token",
		RedditClientID:  "client",
		RedditDeviceID:  "device",
		RedditUserAgent: "agent",
		Now: func() time.Time {
			return now
		},
	}
}
