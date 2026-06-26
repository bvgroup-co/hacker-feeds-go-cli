package feeds

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var hackerNewsHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)

type hackerNewsItem struct {
	ID          int    `json:"id"`
	Deleted     bool   `json:"deleted"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Text        string `json:"text"`
	Dead        bool   `json:"dead"`
	Parent      int    `json:"parent"`
	Kids        []int  `json:"kids"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Title       string `json:"title"`
	Descendants int    `json:"descendants"`
}

func ParsePositiveInt(name string, value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

func ParseNonNegativeInt(name string, value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return parsed, nil
}

func (client Client) FetchNews(top int) ([]NewsItem, error) {
	base, err := url.Parse(joinRawURL(client.NewsBase, "topstories.json"))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, err
	}
	if top > len(ids) {
		top = len(ids)
	}
	items := make([]NewsItem, 0, top)
	for _, id := range ids[:top] {
		itemURL := joinRawURL(client.NewsBase, "item", strconv.Itoa(id)+".json")
		req, err := http.NewRequest(http.MethodGet, itemURL, nil)
		if err != nil {
			return nil, err
		}
		body, err := client.do(req)
		if err != nil {
			return nil, err
		}
		var item hackerNewsItem
		if err := json.Unmarshal(body, &item); err != nil {
			return nil, err
		}
		items = append(items, newsItemFromHackerNews(item))
	}
	return items, nil
}

func (client Client) FetchNewsDiscussion(id int, limit int, depth int) (NewsDiscussion, error) {
	if id <= 0 {
		return NewsDiscussion{}, fmt.Errorf("id must be greater than 0")
	}
	if limit <= 0 {
		return NewsDiscussion{}, fmt.Errorf("limit must be greater than 0")
	}
	if depth <= 0 {
		return NewsDiscussion{}, fmt.Errorf("depth must be greater than 0")
	}
	root, err := client.fetchHackerNewsItem(id)
	if err != nil {
		return NewsDiscussion{}, err
	}
	if root.Type != "story" {
		return NewsDiscussion{}, fmt.Errorf("hacker news item %d is not a story", id)
	}
	if root.Deleted {
		return NewsDiscussion{}, fmt.Errorf("hacker news story %d is deleted", id)
	}
	if root.Dead {
		return NewsDiscussion{}, fmt.Errorf("hacker news story %d is dead", id)
	}
	remaining := limit
	discussion := NewsDiscussion{Item: newsItemFromHackerNews(root)}
	for _, childID := range root.Kids {
		if remaining == 0 {
			break
		}
		comment, ok, err := client.fetchNewsComment(childID, 1, depth, &remaining)
		if err != nil {
			return NewsDiscussion{}, err
		}
		if ok {
			discussion.Comments = append(discussion.Comments, comment)
		}
	}
	return discussion, nil
}

func (client Client) fetchNewsComment(id int, currentDepth int, maxDepth int, remaining *int) (NewsComment, bool, error) {
	if *remaining == 0 {
		return NewsComment{}, false, nil
	}
	item, err := client.fetchHackerNewsItem(id)
	if err != nil {
		return NewsComment{}, false, err
	}
	*remaining = *remaining - 1
	comment := newsCommentFromHackerNews(item, currentDepth-1)
	if currentDepth < maxDepth {
		for _, childID := range item.Kids {
			if *remaining == 0 {
				break
			}
			child, ok, err := client.fetchNewsComment(childID, currentDepth+1, maxDepth, remaining)
			if err != nil {
				return NewsComment{}, false, err
			}
			if ok {
				comment.Children = append(comment.Children, child)
			}
		}
	}
	return comment, true, nil
}

func (client Client) fetchHackerNewsItem(id int) (hackerNewsItem, error) {
	itemURL := joinRawURL(client.NewsBase, "item", strconv.Itoa(id)+".json")
	req, err := http.NewRequest(http.MethodGet, itemURL, nil)
	if err != nil {
		return hackerNewsItem{}, err
	}
	body, err := client.do(req)
	if err != nil {
		return hackerNewsItem{}, err
	}
	if strings.TrimSpace(string(body)) == "null" {
		return hackerNewsItem{}, fmt.Errorf("hacker news item %d not found", id)
	}
	var item hackerNewsItem
	if err := json.Unmarshal(body, &item); err != nil {
		return hackerNewsItem{}, err
	}
	if item.ID == 0 {
		item.ID = id
	}
	return item, nil
}

func newsItemFromHackerNews(item hackerNewsItem) NewsItem {
	return NewsItem{
		ID:          item.ID,
		Title:       item.Title,
		URL:         item.URL,
		Author:      item.By,
		Score:       item.Score,
		Descendants: item.Descendants,
	}
}

func newsCommentFromHackerNews(item hackerNewsItem, depth int) NewsComment {
	return NewsComment{
		ID:       item.ID,
		ParentID: item.Parent,
		Author:   item.By,
		TextHTML: item.Text,
		Text:     plainHackerNewsHTML(item.Text),
		Deleted:  item.Deleted,
		Dead:     item.Dead,
		Time:     item.Time,
		Depth:    depth,
	}
}

func plainHackerNewsHTML(content string) string {
	withLineBreaks := strings.NewReplacer("<p>", "\n\n", "<P>", "\n\n", "<br>", "\n", "<br/>", "\n", "<br />", "\n").Replace(content)
	unescaped := html.UnescapeString(withLineBreaks)
	withoutTags := hackerNewsHTMLTagPattern.ReplaceAllString(unescaped, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}
