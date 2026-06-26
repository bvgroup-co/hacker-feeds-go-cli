package feeds

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var articlePattern = regexp.MustCompile(`(?s)<article[^>]*class="[^"]*Box-row[^"]*"[^>]*>(.*?)</article>`)
var linkPattern = regexp.MustCompile(`(?s)<h2[^>]*class="[^"]*h3[^"]*"[^>]*>.*?<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
var paragraphPattern = regexp.MustCompile(`(?s)<p[^>]*class="[^"]*my-1[^"]*"[^>]*>(.*?)</p>`)
var languagePattern = regexp.MustCompile(`(?s)<span[^>]*itemprop="programmingLanguage"[^>]*>(.*?)</span>`)
var starPattern = regexp.MustCompile(`(?s)<svg[^>]*aria-label="star"[^>]*>.*?</svg>\s*([^<]+)</a>`)
var periodStarsPattern = regexp.MustCompile(`(?s)<span[^>]*class="[^"]*float-sm-right[^"]*"[^>]*>(.*?)</span>`)
var tagPattern = regexp.MustCompile(`(?s)<[^>]+>`)

func ValidSince(since string) bool {
	return since == "daily" || since == "weekly" || since == "monthly"
}

func (client Client) FetchGitHub(language string, since string) ([]GitHubRepo, error) {
	if !ValidSince(since) {
		return nil, fmt.Errorf("since must be daily, weekly, or monthly")
	}
	base, err := url.Parse(client.GitHubBase)
	if err != nil {
		return nil, err
	}
	if language == "" {
		base.Path = joinURLPath(base.Path, "trending") + "/"
	} else {
		base.Path = joinURLPath(base.Path, "trending", language)
	}
	base.RawQuery = "since=" + url.QueryEscape(since) + "&spoken_language_code="
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	return parseGitHubRepos(string(body), client.GitHubBase), nil
}

func parseGitHubRepos(document string, githubBase string) []GitHubRepo {
	matches := articlePattern.FindAllStringSubmatch(document, -1)
	repos := make([]GitHubRepo, 0, len(matches))
	for _, match := range matches {
		article := match[1]
		linkMatch := linkPattern.FindStringSubmatch(article)
		if len(linkMatch) != 3 {
			continue
		}
		title := cleanHTMLText(linkMatch[2])
		parts := strings.Split(title, "/")
		if len(parts) != 2 {
			continue
		}
		author := strings.TrimSpace(parts[0])
		repoName := strings.TrimSpace(parts[1])
		repos = append(repos, GitHubRepo{
			Author:     author,
			Repo:       repoName,
			Link:       strings.TrimRight(githubBase, "/") + linkMatch[1],
			Desc:       firstCleanMatch(paragraphPattern, article),
			Language:   firstCleanMatch(languagePattern, article),
			Stars:      parseCount(firstCleanMatch(starPattern, article)),
			AddedStars: parseCount(firstCleanMatch(periodStarsPattern, article)),
		})
	}
	return repos
}

func firstCleanMatch(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return cleanHTMLText(match[1])
}

func cleanHTMLText(value string) string {
	withoutTags := tagPattern.ReplaceAllString(value, " ")
	decoded := html.UnescapeString(withoutTags)
	return strings.Join(strings.Fields(decoded), " ")
}

func parseCount(value string) int {
	fields := strings.Fields(strings.ReplaceAll(value, ",", ""))
	if len(fields) == 0 {
		return 0
	}
	count, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return count
}
