package feeds

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type productRequest struct {
	Query string `json:"query"`
}

type productResponse struct {
	Data struct {
		Posts struct {
			Edges []struct {
				Node struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					URL         string `json:"url"`
					Website     string `json:"website"`
					Votes       int    `json:"votesCount"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"posts"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (client Client) FetchProducts(count int, past int, now time.Time) ([]Product, error) {
	if client.ProductToken == "" {
		return nil, errors.New("PRODUCT_HUNT_ACCESS_TOKEN is required")
	}
	postedAfter := now.AddDate(0, 0, -past).Format(time.DateOnly)
	postedBefore := now.AddDate(0, 0, 1).Format(time.DateOnly)
	query := fmt.Sprintf(`query { posts(first: %d, order: VOTES, postedAfter: "%s", postedBefore: "%s") { edges{ cursor node{ id name tagline description url votesCount thumbnail{ type url } website reviewsRating }}}}`, count, postedAfter, postedBefore)
	encoded, err := json.Marshal(productRequest{Query: query})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, client.ProductBase, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+client.ProductToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	var decoded productResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Errors) > 0 {
		return nil, errors.New(decoded.Errors[0].Message)
	}
	products := make([]Product, 0, len(decoded.Data.Posts.Edges))
	for _, edge := range decoded.Data.Posts.Edges {
		node := edge.Node
		products = append(products, Product{
			Name:        node.Name,
			Description: node.Description,
			URL:         stripQuery(node.URL),
			Website:     stripQuery(node.Website),
			Votes:       node.Votes,
		})
	}
	return products, nil
}

func stripQuery(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return strings.Split(value, "?")[0]
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
