//go:build linux

package app

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
