//go:build linux

package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/feeds"
)

func TestSelectLanguageRequiresTTY(t *testing.T) {
	var out bytes.Buffer
	app := App{Out: &out, Stdin: nil}
	_, err := app.selectLanguage()
	if err == nil {
		t.Fatal("expected non-TTY error")
	}
}

func TestSelectLanguageTTYPicker(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString("2\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	app := App{
		Out:        &out,
		Stdin:      file,
		IsTerminal: func(*os.File) bool { return true },
	}
	lang, err := app.selectLanguage()
	if err != nil {
		t.Fatal(err)
	}
	if lang != "zh" {
		t.Fatalf("lang = %s", lang)
	}
	if !strings.Contains(out.String(), "EN（English）") || !strings.Contains(out.String(), "ZH（简体中文）") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestIsTerminalDetectsRegularFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if isTerminal(file) {
		t.Fatal("regular file reported as terminal")
	}
}

func TestHelpCommands(t *testing.T) {
	commands := [][]string{
		{"--help"},
		{"-h"},
		{"help"},
		{"help", "news"},
		{"help", "reddit"},
		{"config", "--help"},
		{"github", "--help"},
		{"news", "--help"},
		{"news", "-h"},
		{"news", "help"},
		{"product", "--help"},
		{"product", "details", "--help"},
		{"help", "product", "details"},
		{"product", "comments", "--help"},
		{"help", "product", "comments"},
		{"reddit", "--help"},
		{"v2ex", "--help"},
		{"news", "discussion", "--help"},
		{"news", "comments", "--help"},
		{"help", "news", "discussion"},
		{"reddit", "comments", "--help"},
		{"help", "reddit", "comments"},
	}
	for _, command := range commands {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			var out bytes.Buffer
			var stderr bytes.Buffer
			code := (App{Out: &out, Err: &stderr, Stdin: nil}).Run(command)
			if code != 0 || !strings.Contains(out.String(), "Usage:") || stderr.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
			}
		})
	}
}

func TestProductCommentsFlagErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`<html><title>Product | Product Hunt</title></html>`))
	}))
	defer server.Close()
	client := feeds.Client{HTTP: server.Client(), ProductWebBase: server.URL}
	tests := [][]string{
		{"product", "comments"},
		{"product", "comments", "--url", "https://www.producthunt.com/products/folio-ai", "--slug", "folio-ai"},
		{"product", "comments", "--slug", "folio-ai", "--limit", "0"},
		{"product", "comments", "--slug", "folio-ai", "--depth", "0"},
		{"product", "comments", "--slug", "folio-ai", "--count", "1"},
		{"product", "comments", "--url", "https://example.com/products/folio-ai"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out bytes.Buffer
			var stderr bytes.Buffer
			code := (App{Out: &out, Err: &stderr, Client: &client, Stdin: nil}).Run(args)
			if code == 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
			}
		})
	}
}

func TestProductDetailsFlagErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`<html><title>Product | Product Hunt</title></html>`))
	}))
	defer server.Close()
	client := feeds.Client{HTTP: server.Client(), ProductWebBase: server.URL}
	tests := [][]string{
		{"product", "--details"},
		{"product", "--details", "--url", "https://www.producthunt.com/products/folio-ai", "--slug", "folio-ai"},
		{"product", "--details", "--slug", "folio-ai", "--count", "1"},
		{"product", "details", "--slug", "folio-ai", "--past", "1"},
		{"product", "details", "--url", "https://example.com/products/folio-ai"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out bytes.Buffer
			var stderr bytes.Buffer
			code := (App{Out: &out, Err: &stderr, Client: &client, Stdin: nil}).Run(args)
			if code == 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
			}
		})
	}
}

func TestNewsCommentsAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/item/1.json":
			_, _ = writer.Write([]byte(`{"id":1,"type":"story","by":"alice","title":"Story","score":1,"descendants":1,"kids":[2]}`))
		case "/item/2.json":
			_, _ = writer.Write([]byte(`{"id":2,"type":"comment","by":"bob","parent":1,"text":"Alias works"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := feeds.Client{HTTP: server.Client(), NewsBase: server.URL}
	var out bytes.Buffer
	var stderr bytes.Buffer
	code := (App{Out: &out, Err: &stderr, Client: &client, Stdin: nil}).Run([]string{"news", "comments", "-i", "1", "-c", "1", "-d", "1"})
	if code != 0 || !strings.Contains(out.String(), "Hacker News Discussion") || !strings.Contains(out.String(), "Alias works") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
	}
}
