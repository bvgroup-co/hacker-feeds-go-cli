package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/i18n"
)

const fileName = ".hfrc"

type Config struct {
	Lang string `json:"lang"`
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fileName), nil
}

func Read() Config {
	path, err := Path()
	if err != nil {
		return Config{Lang: "en"}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{Lang: "en"}
	}
	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil || !i18n.ValidLang(cfg.Lang) {
		return Config{Lang: "en"}
	}
	return cfg
}

func WriteLang(lang string) error {
	if !i18n.ValidLang(lang) {
		return errors.New("language must be en or zh")
	}
	path, err := Path()
	if err != nil {
		return err
	}
	cfg := map[string]any{}
	content, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(content, &cfg)
	}
	cfg["lang"] = lang
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o600)
}
