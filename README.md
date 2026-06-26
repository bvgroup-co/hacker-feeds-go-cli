# hacker-feeds-go-cli

A Go rewrite of [`Mayandev/hacker-feeds-cli`](https://github.com/Mayandev/hacker-feeds-cli).

`hfeeds` fetches hacker-oriented feeds from GitHub Trending, Hacker News, Product Hunt, Reddit, and V2EX with English and Simplified Chinese output labels.

## Install

```sh
go install github.com/bvgroup-co/hacker-feeds-go-cli/cmd/hfeeds@latest
```

Build locally:

```sh
go build -o hfeeds ./cmd/hfeeds
```

The original Node package also exposed an `hf` binary. For Go packaging, create a shell alias or symlink if desired:

```sh
alias hf=hfeeds
```

## Usage

```sh
hfeeds --help
hfeeds --version
hfeeds github
```

Commands:

```text
hfeeds config --lang en|zh
hfeeds github [-s daily|weekly|monthly] [-l language]
hfeeds news [-t top]
hfeeds product [-c count] [-p past]
hfeeds reddit [-t topic] [-s hot|new|top|best]
hfeeds v2ex [-n node]
```

Examples:

```sh
hfeeds github
hfeeds github --since weekly --lang go
hfeeds news --top 5
hfeeds product --count 5 --past 1
hfeeds reddit --topic golang --sort top
hfeeds v2ex --node programmer
```

## Configuration

Language configuration is stored in `$HOME/.hfrc` as JSON:

```json
{"lang":"en"}
```

Supported languages are `en` and `zh`.

```sh
hfeeds config --lang en
hfeeds config --lang zh
```

Running `hfeeds config` without `--lang` in a non-interactive environment exits with an instruction to pass `--lang en|zh`.

## Environment variables

Product Hunt requires an access token:

```sh
export PRODUCT_HUNT_ACCESS_TOKEN=your-token
```

The HTTP base URLs can be overridden for tests:

```sh
HFEEDS_GITHUB_BASE_URL=http://127.0.0.1:8080
HFEEDS_HN_BASE_URL=http://127.0.0.1:8081
HFEEDS_PRODUCT_HUNT_BASE_URL=http://127.0.0.1:8082/graphql
HFEEDS_REDDIT_BASE_URL=http://127.0.0.1:8083
HFEEDS_V2EX_BASE_URL=http://127.0.0.1:8084
```

Set `NO_COLOR=1` in scripts/tests to keep output plain.

## Development

Run tests:

```sh
CGO_ENABLED=0 go test ./...
```

Format and vet:

```sh
gofmt -w .
CGO_ENABLED=0 go vet ./...
```
