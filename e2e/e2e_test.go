package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "hfeeds-e2e-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	binaryPath = filepath.Join(dir, "hfeeds")
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/hfeeds")
	build.Dir = ".."
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		panic(string(output))
	}
	os.Exit(m.Run())
}

func TestCLIAndConfig(t *testing.T) {
	t.Run("help and version", func(t *testing.T) {
		for _, args := range [][]string{{}, {"--help"}, {"-h"}, {"help"}, {"help", "github"}} {
			result := run(t, args, nil)
			if result.code != 0 || !strings.Contains(result.stdout, "hfeeds") && !strings.Contains(result.stdout, "Usage:") {
				t.Fatalf("args %v: %#v", args, result)
			}
		}
		result := run(t, []string{"--version"}, nil)
		if result.code != 0 || strings.TrimSpace(result.stdout) == "" {
			t.Fatalf("version: %#v", result)
		}
	})
	t.Run("unknown command", func(t *testing.T) {
		result := run(t, []string{"missing"}, nil)
		if result.code == 0 || !strings.Contains(result.stderr, "unknown command") {
			t.Fatalf("result = %#v", result)
		}
	})
	t.Run("config", func(t *testing.T) {
		home := t.TempDir()
		result := run(t, []string{"config", "--lang", "en"}, []string{"HOME=" + home})
		if result.code != 0 || !strings.Contains(result.stdout, "Config Saved") {
			t.Fatalf("en result = %#v", result)
		}
		assertConfigLang(t, home, "en")
		result = run(t, []string{"config", "--lang", "zh"}, []string{"HOME=" + home})
		if result.code != 0 || !strings.Contains(result.stdout, "配置成功") {
			t.Fatalf("zh result = %#v", result)
		}
		assertConfigLang(t, home, "zh")
		result = run(t, []string{"config", "--lang", "invalid"}, []string{"HOME=" + home})
		if result.code == 0 {
			t.Fatalf("invalid result = %#v", result)
		}
		result = run(t, []string{"config"}, []string{"HOME=" + home})
		if result.code == 0 || !strings.Contains(result.stderr, "--lang en|zh") {
			t.Fatalf("non tty result = %#v", result)
		}
	})
}

func TestGitHubE2E(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/trending/", "/trending/go":
			if request.URL.Query().Get("since") == "monthly" {
				_, _ = writer.Write([]byte(`<article class="Box-row"><h2 class="h3"><a href="/owner/monthly"> owner / monthly </a></h2><span itemprop="programmingLanguage">Go</span><a><svg aria-label="star"></svg> 99</a><span class="float-sm-right">8 stars this month</span></article>`))
				return
			}
			_, _ = writer.Write([]byte(`<article class="Box-row"><h2 class="h3"><a href="/owner/repo"> owner / repo </a></h2><p class="my-1">Useful repo</p><span itemprop="programmingLanguage">Go</span><a><svg aria-label="star"></svg> 10</a><span class="float-sm-right">2 stars today</span></article>`))
		case "/empty/trending/":
			_, _ = writer.Write([]byte(`<html></html>`))
		case "/fail/trending/":
			http.Error(writer, "no", http.StatusInternalServerError)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result := run(t, []string{"github"}, []string{"HFEEDS_GITHUB_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "GitHub Trending List") || !strings.Contains(result.stdout, "Stars this day") {
		t.Fatalf("github result = %#v", result)
	}
	result = run(t, []string{"github", "-s", "weekly", "-l", "go"}, []string{"HFEEDS_GITHUB_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "Stars this week") {
		t.Fatalf("weekly result = %#v", result)
	}
	result = run(t, []string{"github", "--since", "monthly"}, []string{"HFEEDS_GITHUB_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "Stars this month") {
		t.Fatalf("monthly result = %#v", result)
	}
	result = run(t, []string{"github", "--since", "yearly"}, []string{"HFEEDS_GITHUB_BASE_URL=" + server.URL})
	if result.code == 0 {
		t.Fatalf("invalid result = %#v", result)
	}
	home := writeLang(t, "zh")
	result = run(t, []string{"github"}, []string{"HOME=" + home, "HFEEDS_GITHUB_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "GitHub 榜单") {
		t.Fatalf("zh result = %#v", result)
	}
}

func TestNewsE2E(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/topstories.json" {
			_, _ = writer.Write([]byte(`[1,2,3,4,5,6,7,8,9,10,11]`))
			return
		}
		if request.URL.Path == "/item/2.json" && request.URL.Query().Get("fail") == "1" {
			http.Error(writer, "no", http.StatusInternalServerError)
			return
		}
		switch request.URL.Path {
		case "/item/1.json":
			_, _ = writer.Write([]byte(`{"id":1,"type":"story","by":"alice","title":"News 1","url":"https://news.example/1","score":42,"descendants":2,"kids":[12,13]}`))
			return
		case "/item/12.json":
			_, _ = writer.Write([]byte(`{"id":12,"type":"comment","by":"bob","parent":1,"text":"First &amp; comment","kids":[14]}`))
			return
		case "/item/13.json":
			_, _ = writer.Write([]byte(`{"id":13,"type":"comment","deleted":true,"parent":1}`))
			return
		case "/item/14.json":
			_, _ = writer.Write([]byte(`{"id":14,"type":"comment","by":"carol","parent":12,"text":"Nested reply"}`))
			return
		default:
			if strings.HasPrefix(request.URL.Path, "/item/") {
				id := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/item/"), ".json")
				_, _ = writer.Write([]byte(`{"id":` + id + `,"title":"News ` + id + `","url":"https://news.example/` + id + `","by":"user` + id + `","score":` + id + `,"descendants":0}`))
				return
			}
		}
		http.NotFound(writer, request)
	}))
	defer server.Close()

	result := run(t, []string{"news", "--top", "1"}, []string{"HFEEDS_HN_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "Hacker News List") || strings.Count(result.stdout, "Title:") != 1 || !strings.Contains(result.stdout, "ID: 1 | Author: alice | Score: 42 | Comments: 2") {
		t.Fatalf("top 1 = %#v", result)
	}
	result = run(t, []string{"news", "-t", "2"}, []string{"HFEEDS_HN_BASE_URL=" + server.URL})
	if result.code != 0 || strings.Count(result.stdout, "Title:") != 2 {
		t.Fatalf("top 2 = %#v", result)
	}
	result = run(t, []string{"news", "-t", "0"}, []string{"HFEEDS_HN_BASE_URL=" + server.URL})
	if result.code == 0 {
		t.Fatalf("invalid = %#v", result)
	}
	result = run(t, []string{"news", "discussion", "--id", "1", "--limit", "10", "--depth", "2"}, []string{"HFEEDS_HN_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "Hacker News Discussion") || !strings.Contains(result.stdout, "Comment: First & comment") || !strings.Contains(result.stdout, "  Author: carol | ID: 14") || !strings.Contains(result.stdout, "[deleted]") {
		t.Fatalf("discussion = %#v", result)
	}
	result = run(t, []string{"news", "comments", "-i", "1", "-c", "1", "-d", "1"}, []string{"HFEEDS_HN_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "Comment: First & comment") || strings.Contains(result.stdout, "Nested reply") || strings.Contains(result.stdout, "[deleted]") {
		t.Fatalf("comments alias = %#v", result)
	}
	home := writeLang(t, "zh")
	result = run(t, []string{"news", "--top", "1"}, []string{"HOME=" + home, "HFEEDS_HN_BASE_URL=" + server.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "Hacker News 新闻") {
		t.Fatalf("zh = %#v", result)
	}
}

func TestProductRedditV2EXE2E(t *testing.T) {
	productServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing token")
		}
		_, _ = writer.Write([]byte(`{"data":{"posts":{"edges":[{"node":{"name":"Prod","description":"Desc","url":"https://p.example/path?ref=x","website":"https://w.example/?a=b","votesCount":5}}]}}}`))
	}))
	defer productServer.Close()
	result := run(t, []string{"product", "-c", "2", "-p", "1"}, []string{"HFEEDS_PRODUCT_HUNT_BASE_URL=" + productServer.URL, "PRODUCT_HUNT_ACCESS_TOKEN=token"})
	if result.code != 0 || !strings.Contains(result.stdout, "Product Hunt List") || !strings.Contains(result.stdout, "Votes: 5") || strings.Contains(result.stdout, "?ref=x") {
		t.Fatalf("product = %#v", result)
	}
	home := writeLang(t, "zh")
	result = run(t, []string{"product"}, []string{"HOME=" + home, "HFEEDS_PRODUCT_HUNT_BASE_URL=" + productServer.URL, "PRODUCT_HUNT_ACCESS_TOKEN=token"})
	if result.code != 0 || !strings.Contains(result.stdout, "Product Hunt 榜单") || !strings.Contains(result.stdout, "投票: 5") {
		t.Fatalf("product zh = %#v", result)
	}
	result = run(t, []string{"product"}, []string{"HFEEDS_PRODUCT_HUNT_BASE_URL=" + productServer.URL})
	if result.code == 0 || !strings.Contains(result.stderr, "PRODUCT_HUNT_ACCESS_TOKEN") {
		t.Fatalf("product token = %#v", result)
	}

	redditServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/r/popular/.rss", "/r/golang/.rss":
			_, _ = writer.Write([]byte(redditE2ERSS()))
		case "/svc/shreddit/comments/r/golang/t3_1":
			_, _ = writer.Write([]byte(redditE2EShreddit()))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer redditServer.Close()
	for _, args := range [][]string{{"reddit"}, {"reddit", "-t", "golang"}} {
		result = run(t, args, redditEnv(redditServer))
		if result.code != 0 || !strings.Contains(result.stdout, "Reddit List") || strings.Contains(result.stdout, "Content:") {
			t.Fatalf("reddit %v = %#v", args, result)
		}
	}
	result = run(t, []string{"reddit", "comments", "--topic", "golang", "--post", "1"}, redditEnv(redditServer))
	if result.code != 0 || !strings.Contains(result.stdout, "Comment body") || !strings.Contains(result.stdout, "Author: bob") {
		t.Fatalf("reddit comments = %#v", result)
	}
	home = writeLang(t, "zh")
	result = run(t, []string{"reddit"}, append([]string{"HOME=" + home}, redditEnv(redditServer)...))
	if result.code != 0 || !strings.Contains(result.stdout, "Reddit 帖子") || !strings.Contains(result.stdout, "投票: 0") || !strings.Contains(result.stdout, "话题: popular") {
		t.Fatalf("reddit zh = %#v", result)
	}

	v2exServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/topics/show.json" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`[{"title":"Topic","content":"","replies":2,"url":"https://v2ex.example/t/1","votes":4,"node":{"name":"` + request.URL.Query().Get("node_name") + `"}}]`))
	}))
	defer v2exServer.Close()
	result = run(t, []string{"v2ex", "-n", "programmer"}, []string{"HFEEDS_V2EX_BASE_URL=" + v2exServer.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "V2EX Feeds List") || !strings.Contains(result.stdout, "Votes: 4") || !strings.Contains(result.stdout, "programmer") || strings.Contains(result.stdout, "Content:") {
		t.Fatalf("v2ex = %#v", result)
	}
	home = writeLang(t, "zh")
	result = run(t, []string{"v2ex"}, []string{"HOME=" + home, "HFEEDS_V2EX_BASE_URL=" + v2exServer.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "V2EX 帖子") || !strings.Contains(result.stdout, "Votes: 4") || !strings.Contains(result.stdout, "节点: create") {
		t.Fatalf("v2ex zh = %#v", result)
	}
}

type commandResult struct {
	code   int
	stdout string
	stderr string
}

func run(t *testing.T, args []string, env []string) commandResult {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Env = append(cmd.Env, env...)
	var stdout strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = exitError.ExitCode()
	}
	return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func assertConfigLang(t *testing.T, home string, lang string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(home, ".hfrc"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Lang string `json:"lang"`
	}
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Lang != lang {
		t.Fatalf("lang = %s", cfg.Lang)
	}
}

func writeLang(t *testing.T, lang string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".hfrc"), []byte(`{"lang":"`+lang+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestErrorAndEmptyE2E(t *testing.T) {
	githubServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/empty/trending/":
			_, _ = writer.Write([]byte(`<html></html>`))
		case "/fail/trending/":
			http.Error(writer, "no", http.StatusInternalServerError)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer githubServer.Close()
	result := run(t, []string{"github"}, []string{"HFEEDS_GITHUB_BASE_URL=" + githubServer.URL + "/empty"})
	if result.code != 0 || !strings.Contains(result.stdout, "ranking has not yet been updated") {
		t.Fatalf("github empty = %#v", result)
	}
	result = run(t, []string{"github"}, []string{"HFEEDS_GITHUB_BASE_URL=" + githubServer.URL + "/fail"})
	if result.code == 0 {
		t.Fatalf("github fail = %#v", result)
	}

	newsServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/topstories.json":
			_, _ = writer.Write([]byte(`[1,2]`))
		case "/item/1.json":
			_, _ = writer.Write([]byte(`{"title":"One"}`))
		case "/item/2.json":
			http.Error(writer, "no", http.StatusInternalServerError)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer newsServer.Close()
	result = run(t, []string{"news", "-t", "bad"}, []string{"HFEEDS_HN_BASE_URL=" + newsServer.URL})
	if result.code == 0 {
		t.Fatalf("news invalid = %#v", result)
	}
	result = run(t, []string{"news", "-t", "2"}, []string{"HFEEDS_HN_BASE_URL=" + newsServer.URL})
	if result.code == 0 {
		t.Fatalf("news failure = %#v", result)
	}

	productServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/empty":
			_, _ = writer.Write([]byte(`{"data":{"posts":{"edges":[]}}}`))
		case "/errors":
			_, _ = writer.Write([]byte(`{"errors":[{"message":"bad query"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer productServer.Close()
	for _, args := range [][]string{{"product", "-c", "0"}, {"product", "-p", "-1"}} {
		result = run(t, args, []string{"HFEEDS_PRODUCT_HUNT_BASE_URL=" + productServer.URL + "/empty", "PRODUCT_HUNT_ACCESS_TOKEN=token"})
		if result.code == 0 {
			t.Fatalf("product invalid %v = %#v", args, result)
		}
	}
	redditServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/r/popular/.rss":
			if request.URL.Query().Get("fail") == "1" {
				http.Error(writer, "too many", http.StatusTooManyRequests)
				return
			}
			_, _ = writer.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"></feed>`))
		case "/api/posts/search":
			http.Error(writer, "too many", http.StatusTooManyRequests)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer redditServer.Close()
	result = run(t, []string{"reddit"}, redditEnv(redditServer))
	if result.code != 0 || !strings.Contains(result.stdout, "ranking has not yet been updated") {
		t.Fatalf("reddit empty = %#v", result)
	}
	redditFailServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "too many", http.StatusTooManyRequests)
	}))
	defer redditFailServer.Close()
	result = run(t, []string{"reddit"}, redditEnv(redditFailServer))
	if result.code == 0 || !strings.Contains(result.stderr, "Reddit source unavailable without OAuth") {
		t.Fatalf("reddit rate limited = %#v", result)
	}

	redditBlockedServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "blocked", http.StatusForbidden)
	}))
	defer redditBlockedServer.Close()
	result = run(t, []string{"reddit"}, redditEnv(redditBlockedServer))
	if result.code == 0 || !strings.Contains(result.stderr, "Reddit source unavailable without OAuth") || strings.TrimSpace(result.stderr) == "request failed with status 403" {
		t.Fatalf("reddit blocked = %#v", result)
	}

	v2exServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Query().Get("node_name") {
		case "create":
			_, _ = writer.Write([]byte(`[]`))
		case "programmer":
			_, _ = writer.Write([]byte(`not-json`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer v2exServer.Close()
	result = run(t, []string{"v2ex"}, []string{"HFEEDS_V2EX_BASE_URL=" + v2exServer.URL})
	if result.code != 0 || !strings.Contains(result.stdout, "ranking has not yet been updated") {
		t.Fatalf("v2ex empty = %#v", result)
	}
	result = run(t, []string{"v2ex", "-n", "programmer"}, []string{"HFEEDS_V2EX_BASE_URL=" + v2exServer.URL})
	if result.code == 0 {
		t.Fatalf("v2ex malformed = %#v", result)
	}
}

func redditEnv(server *httptest.Server) []string {
	return []string{
		"HFEEDS_REDDIT_BASE_URL=" + server.URL,
		"HFEEDS_ARCTIC_SHIFT_BASE_URL=" + server.URL,
		"HFEEDS_REDDIT_USER_AGENT=agent",
	}
}

func redditE2ERSS() string {
	return `<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>/r/popular</title><category term="popular"></category><entry><id>https://www.reddit.com/r/popular/comments/t3_1/post/</id><title>Post</title><author><name>alice</name></author><link href="https://www.reddit.com/r/popular/comments/1/post/" /><content type="html"></content><category term="popular"></category></entry></feed>`
}

func redditE2EShreddit() string {
	return `<shreddit-comment thingId="t1_c1" author="bob" created="1770000000" depth="0" parentId="t3_1" permalink="/r/golang/comments/1/comment/c1/" score="5" postId="t3_1"><div slot="comment">Comment body</div></shreddit-comment>`
}
