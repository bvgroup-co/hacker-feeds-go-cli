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
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/r/golang/top.json" {
			t.Fatalf("path = %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"data":{"children":[{"data":{"title":"Post","selftext":"Body","num_comments":3,"permalink":"/r/golang/comments/1/post/","ups":9,"subreddit":"golang"}}]}}`))
	}))
	defer server.Close()

	posts, err := (Client{HTTP: server.Client(), RedditBase: server.URL}).FetchReddit("golang", "top")
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].Link != server.URL+"/r/golang/comments/1/post/" {
		t.Fatalf("posts = %#v", posts)
	}
}

func TestFetchV2EX(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/topics/show.json" || request.URL.Query().Get("node_name") != "programmer" {
			t.Fatalf("url = %s", request.URL.String())
		}
		_, _ = writer.Write([]byte(`[{"title":"Topic","content":"Body","replies":4,"url":"https://v2ex.example/t/1","node":{"name":"programmer"}}]`))
	}))
	defer server.Close()

	topics, err := (Client{HTTP: server.Client(), V2EXBase: server.URL}).FetchV2EX("programmer")
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 || topics[0].Node != "programmer" {
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
