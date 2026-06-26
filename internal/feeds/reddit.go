package feeds

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string `json:"title"`
				Content   string `json:"selftext"`
				Comment   int    `json:"num_comments"`
				Permalink string `json:"permalink"`
				Votes     int    `json:"ups"`
				Topic     string `json:"subreddit"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func ValidRedditSort(sort string) bool {
	return sort == "hot" || sort == "new" || sort == "top" || sort == "best"
}

func (client Client) FetchReddit(topic string, sort string) ([]RedditPost, error) {
	if !ValidRedditSort(sort) {
		return nil, fmt.Errorf("sort must be hot, new, top, or best")
	}
	base, err := url.Parse(client.RedditBase)
	if err != nil {
		return nil, err
	}
	base.Path = joinURLPath(base.Path, "r", topic, sort+".json")
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "hacker-feeds-go-cli")
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	var decoded redditResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	posts := make([]RedditPost, 0, len(decoded.Data.Children))
	for _, child := range decoded.Data.Children {
		data := child.Data
		posts = append(posts, RedditPost{
			Title:   data.Title,
			Content: data.Content,
			Comment: data.Comment,
			Link:    strings.TrimRight(client.RedditBase, "/") + data.Permalink,
			Votes:   data.Votes,
			Topic:   data.Topic,
		})
	}
	return posts, nil
}
