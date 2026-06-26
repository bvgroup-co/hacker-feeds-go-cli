# hacker-feeds-go-cli

A Go rewrite of [`Mayandev/hacker-feeds-cli`](https://github.com/Mayandev/hacker-feeds-cli).

`hfeeds` fetches hacker-oriented feeds from GitHub Trending, Hacker News, Product Hunt, Reddit, and V2EX with English and Simplified Chinese output labels.

## Install

### Homebrew

```sh
brew tap bvgroup-co/tap
brew install hfeeds
```

Verify the installed binary:

```sh
hfeeds --version
```

Homebrew releases are published through the `bvgroup-co/homebrew-tap` repository. Release automation requires the repository secret `HOMEBREW_TAP_TOKEN` with permission to push Homebrew formula updates to that tap.

### Go install

```sh
go install github.com/bvgroup-co/hacker-feeds-go-cli/cmd/hfeeds@latest
```

### GitHub release archives

Download a prebuilt archive from the GitHub Releases page for your platform:

| Platform | Archive suffix |
| --- | --- |
| macOS Intel | `darwin_amd64` |
| macOS Apple Silicon | `darwin_arm64` |
| Linux x86_64 | `linux_amd64` |
| Linux ARM64 | `linux_arm64` |

Each release includes `checksums.txt` for SHA256 verification. Extract the matching archive and place the `hfeeds` binary on your `PATH`.

### Local build

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
hfeeds reddit [-t topic] [-s hot|new|top|best] [-c limit]
hfeeds reddit comments --topic topic --post post_id [--limit n] [--depth n] [--sort confidence|top|new|controversial|old|qa]
hfeeds v2ex [-n node]
```

Examples:

```sh
hfeeds github
hfeeds github --since weekly --lang go
hfeeds news --top 5
hfeeds product --count 5 --past 1
hfeeds reddit --topic golang --sort top
hfeeds reddit comments --topic golang --post abc123 --limit 10 --depth 2 --sort top
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

### Reddit access

Reddit OAuth is required. The CLI uses Reddit app-only OAuth for an installed app and does not access user accounts. No client secret is needed for installed-client mode.

Create a Reddit installed app, then export:

```sh
export HFEEDS_REDDIT_CLIENT_ID='your installed-app client id'
export HFEEDS_REDDIT_DEVICE_ID='stable-20-to-30-char-id'
export HFEEDS_REDDIT_USER_AGENT='hfeeds/0.5.0 by your-reddit-username'
```

`hfeeds reddit` fetches authenticated listings from `https://oauth.reddit.com/r/{subreddit}/{sort}` and prints post title, selftext when available, external URL, Reddit permalink, score/votes, comment count, subreddit, author, and domain.

`hfeeds reddit comments --topic golang --post abc123 --limit 10 --depth 2 --sort top` fetches authenticated discussions from `https://oauth.reddit.com/r/{subreddit}/comments/{post_id}` and prints post details plus nested comments. Supported comment sorts are `confidence`, `top`, `new`, `controversial`, `old`, and `qa`.

Missing Reddit config fails before any network request with:

```text
Reddit OAuth is required. Set HFEEDS_REDDIT_CLIENT_ID, HFEEDS_REDDIT_DEVICE_ID, and HFEEDS_REDDIT_USER_AGENT.
```

Token and API errors are mapped to actionable messages for invalid credentials, invalid grant/device ID, forbidden app setup or user agent, missing subreddit/post, rate limiting with `Retry-After` when provided, and Reddit server errors.

The prior RSS and unauthenticated `.json` fallback was removed because it is unreliable, incomplete, and lacks votes/comments.

The HTTP base URLs can be overridden for tests:

```sh
HFEEDS_GITHUB_BASE_URL=http://127.0.0.1:8080
HFEEDS_HN_BASE_URL=http://127.0.0.1:8081
HFEEDS_PRODUCT_HUNT_BASE_URL=http://127.0.0.1:8082/graphql
HFEEDS_REDDIT_OAUTH_BASE_URL=http://127.0.0.1:8083
HFEEDS_REDDIT_TOKEN_URL=http://127.0.0.1:8083/token
HFEEDS_V2EX_BASE_URL=http://127.0.0.1:8084
```

Set `NO_COLOR=1` in scripts/tests to keep output plain.

## Releases

Release automation runs when a semantic version tag is pushed:

```sh
git tag v0.5.0
git push origin v0.5.0
```

Choose the next unused semantic version tag for each release. Release builds inject the tag version with Go ldflags, so a `v0.5.0` binary reports `0.5.0` from `hfeeds --version`. Local development builds report `dev`.

The release workflow uses GoReleaser to build and publish:

- `hfeeds_<version>_linux_amd64.tar.gz`
- `hfeeds_<version>_linux_arm64.tar.gz`
- `hfeeds_<version>_darwin_amd64.tar.gz`
- `hfeeds_<version>_darwin_arm64.tar.gz`
- `checksums.txt`

Homebrew formula updates are committed to `bvgroup-co/homebrew-tap` under `Formula/hfeeds.rb`. Configure the `HOMEBREW_TAP_TOKEN` repository secret before cutting a release; the token must be able to push to the tap repository.

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

Check the release configuration locally:

```sh
goreleaser check
```
