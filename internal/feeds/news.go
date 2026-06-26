package feeds

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

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
		var item NewsItem
		if err := json.Unmarshal(body, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
