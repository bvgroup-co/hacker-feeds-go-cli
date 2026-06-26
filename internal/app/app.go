package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/config"
	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/feeds"
	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/i18n"
	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/output"
)

var Version = "dev"

type App struct {
	Out        io.Writer
	Err        io.Writer
	Stdin      *os.File
	Client     *feeds.Client
	Now        func() time.Time
	IsTerminal func(*os.File) bool
}

func New() App {
	client := feeds.NewClientFromEnv(os.Getenv)
	return App{
		Out:        os.Stdout,
		Err:        os.Stderr,
		Stdin:      os.Stdin,
		Client:     &client,
		Now:        time.Now,
		IsTerminal: isTerminal,
	}
}

func (app App) Run(args []string) int {
	if len(args) == 0 {
		app.printHelp(app.Out)
		return 0
	}
	command := args[0]
	if command == "--help" || command == "-h" || command == "help" {
		return app.help(args[1:])
	}
	if command == "--version" || command == "-v" {
		fmt.Fprintln(app.Out, Version)
		return 0
	}
	labels := i18n.For(config.Read().Lang)
	var err error
	switch command {
	case "config":
		err = app.config(args[1:], labels)
	case "github":
		err = app.github(args[1:], labels)
	case "news":
		err = app.news(args[1:], labels)
	case "product":
		err = app.product(args[1:], labels)
	case "reddit":
		err = app.reddit(args[1:], labels)
	case "v2ex":
		err = app.v2ex(args[1:], labels)
	default:
		fmt.Fprintf(app.Err, "unknown command: %s\n", command)
		return 1
	}
	if err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	return 0
}

func (app App) help(args []string) int {
	if len(args) == 0 {
		app.printHelp(app.Out)
		return 0
	}
	app.printCommandHelp(app.Out, args[0])
	return 0
}

func (app App) config(args []string, labels i18n.Labels) error {
	flags, err := parseStringFlags(args, map[string]string{"--lang": ""}, flagSpec{Short: "-l", Long: "--lang"})
	if err != nil {
		return err
	}
	lang := flags["--lang"]
	if lang == "" {
		selected, err := app.selectLanguage()
		if err != nil {
			return err
		}
		lang = selected
	}
	if err := config.WriteLang(lang); err != nil {
		fmt.Fprintln(app.Err, labels.Config.Failed)
		return err
	}
	fmt.Fprintln(app.Out, i18n.For(lang).Config.Saved)
	return nil
}

func (app App) selectLanguage() (string, error) {
	if !app.stdinIsTerminal() {
		return "", fmt.Errorf("non-interactive config requires --lang en|zh")
	}
	fmt.Fprintln(app.Out, "Please select a language(Default EN):")
	fmt.Fprintln(app.Out, "1) EN（English）")
	fmt.Fprintln(app.Out, "2) ZH（简体中文）")
	fmt.Fprint(app.Out, "> ")
	var choice string
	if _, err := fmt.Fscanln(app.Stdin, &choice); err != nil {
		return "", err
	}
	switch choice {
	case "", "1", "en", "EN":
		return "en", nil
	case "2", "zh", "ZH":
		return "zh", nil
	default:
		return "", fmt.Errorf("language must be en or zh")
	}
}

func (app App) stdinIsTerminal() bool {
	terminal := app.IsTerminal
	if terminal == nil {
		terminal = isTerminal
	}
	return terminal(app.Stdin)
}

func (app App) github(args []string, labels i18n.Labels) error {
	flags, err := parseStringFlags(args, map[string]string{"--since": "daily", "--lang": ""}, flagSpec{Short: "-s", Long: "--since"}, flagSpec{Short: "-l", Long: "--lang"})
	if err != nil {
		return err
	}
	since := flags["--since"]
	language := flags["--lang"]
	if !feeds.ValidSince(since) {
		return errors.New("--since must be daily, weekly, or monthly")
	}
	items, err := app.client().FetchGitHub(language, since)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		output.NoItems(app.Out, labels)
		return nil
	}
	output.GitHub(app.Out, labels, since, items)
	return nil
}

func (app App) news(args []string, labels i18n.Labels) error {
	flags, err := parseStringFlags(args, map[string]string{"--top": "10"}, flagSpec{Short: "-t", Long: "--top"})
	if err != nil {
		return err
	}
	topValue := flags["--top"]
	top, err := feeds.ParsePositiveInt("--top", topValue)
	if err != nil {
		return err
	}
	items, err := app.client().FetchNews(top)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		output.NoItems(app.Out, labels)
		return nil
	}
	output.News(app.Out, labels, items)
	return nil
}

func (app App) product(args []string, labels i18n.Labels) error {
	flags, err := parseStringFlags(args, map[string]string{"--count": "10", "--past": "0"}, flagSpec{Short: "-c", Long: "--count"}, flagSpec{Short: "-p", Long: "--past"})
	if err != nil {
		return err
	}
	countValue := flags["--count"]
	pastValue := flags["--past"]
	count, err := feeds.ParsePositiveInt("--count", countValue)
	if err != nil {
		return err
	}
	past, err := feeds.ParseNonNegativeInt("--past", pastValue)
	if err != nil {
		return err
	}
	products, err := app.client().FetchProducts(count, past, app.Now())
	if err != nil {
		return err
	}
	if len(products) == 0 {
		output.NoItems(app.Out, labels)
		return nil
	}
	output.Product(app.Out, labels, products)
	return nil
}

func (app App) reddit(args []string, labels i18n.Labels) error {
	if len(args) > 0 && args[0] == "comments" {
		return app.redditComments(args[1:], labels)
	}
	flags, err := parseStringFlags(args, map[string]string{"--topic": "popular", "--limit": "10"}, flagSpec{Short: "-t", Long: "--topic"}, flagSpec{Short: "-c", Long: "--limit"})
	if err != nil {
		return err
	}
	topic := flags["--topic"]
	limit, err := feeds.ParsePositiveInt("--limit", flags["--limit"])
	if err != nil {
		return err
	}
	posts, err := app.client().FetchReddit(topic, limit)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		output.NoItems(app.Out, labels)
		return nil
	}
	output.Reddit(app.Out, labels, posts)
	return nil
}

func (app App) redditComments(args []string, labels i18n.Labels) error {
	flags, err := parseStringFlags(args, map[string]string{"--topic": "popular", "--post": "", "--limit": "10", "--depth": "2"}, flagSpec{Short: "-t", Long: "--topic"}, flagSpec{Short: "-p", Long: "--post"}, flagSpec{Short: "-c", Long: "--limit"}, flagSpec{Short: "-d", Long: "--depth"})
	if err != nil {
		return err
	}
	limit, err := feeds.ParsePositiveInt("--limit", flags["--limit"])
	if err != nil {
		return err
	}
	depth, err := feeds.ParsePositiveInt("--depth", flags["--depth"])
	if err != nil {
		return err
	}
	discussion, err := app.client().FetchRedditComments(flags["--topic"], flags["--post"], limit, depth)
	if err != nil {
		return err
	}
	output.RedditDiscussion(app.Out, labels, discussion)
	return nil
}

func (app App) v2ex(args []string, labels i18n.Labels) error {
	flags, err := parseStringFlags(args, map[string]string{"--node": "create"}, flagSpec{Short: "-n", Long: "--node"})
	if err != nil {
		return err
	}
	node := flags["--node"]
	topics, err := app.client().FetchV2EX(node)
	if err != nil {
		return err
	}
	if len(topics) == 0 {
		output.NoItems(app.Out, labels)
		return nil
	}
	output.V2EX(app.Out, labels, topics)
	return nil
}

func (app App) client() *feeds.Client {
	if app.Client != nil {
		return app.Client
	}
	client := feeds.NewClientFromEnv(os.Getenv)
	return &client
}

func (app App) printHelp(writer io.Writer) {
	fmt.Fprintln(writer, "hfeeds - hacker feeds CLI")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  hfeeds <command> [options]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Commands:")
	fmt.Fprintln(writer, "  config   config cli")
	fmt.Fprintln(writer, "  github   get the github trending list")
	fmt.Fprintln(writer, "  news     get the hacker news list")
	fmt.Fprintln(writer, "  product  get the product hunt list")
	fmt.Fprintln(writer, "  reddit   get reddit post list")
	fmt.Fprintln(writer, "  v2ex     get v2ex post list")
	fmt.Fprintln(writer, "  help     display help for command")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Options:")
	fmt.Fprintln(writer, "  -h, --help      display help")
	fmt.Fprintln(writer, "  -v, --version   output the version number")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Example:")
	fmt.Fprintln(writer, "  $ hfeeds github")
}

func (app App) printCommandHelp(writer io.Writer, command string) {
	switch command {
	case "config":
		fmt.Fprintln(writer, "Usage: hfeeds config --lang en|zh")
	case "github":
		fmt.Fprintln(writer, "Usage: hfeeds github [-s daily|weekly|monthly] [-l language]")
	case "news":
		fmt.Fprintln(writer, "Usage: hfeeds news [-t top]")
	case "product":
		fmt.Fprintln(writer, "Usage: hfeeds product [-c count] [-p past]")
	case "reddit":
		fmt.Fprintln(writer, "Usage: hfeeds reddit [-t topic] [-c limit]")
		fmt.Fprintln(writer, "       hfeeds reddit comments --topic topic --post post_id [--limit n] [--depth n]")
	case "v2ex":
		fmt.Fprintln(writer, "Usage: hfeeds v2ex [-n node]")
	default:
		app.printHelp(writer)
	}
}
