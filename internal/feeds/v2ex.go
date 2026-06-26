package feeds

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type v2exTopic struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Comment int    `json:"replies"`
	Link    string `json:"url"`
	Votes   int    `json:"votes"`
	Node    struct {
		Name string `json:"name"`
	} `json:"node"`
}

func (client Client) FetchV2EX(node string) ([]V2EXTopic, error) {
	base, err := url.Parse(joinRawURL(client.V2EXBase, "api", "topics", "show.json"))
	if err != nil {
		return nil, err
	}
	query := base.Query()
	query.Set("node_name", node)
	base.RawQuery = query.Encode()
	req, err := http.NewRequest(http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	var decoded []v2exTopic
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	topics := make([]V2EXTopic, 0, len(decoded))
	for _, item := range decoded {
		topics = append(topics, V2EXTopic{
			Title:   item.Title,
			Content: item.Content,
			Comment: item.Comment,
			Link:    item.Link,
			Votes:   item.Votes,
			Node:    item.Node.Name,
		})
	}
	return topics, nil
}
