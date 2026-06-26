package feeds

import (
	"fmt"
	"io"
	"net/http"
)

func (client Client) do(req *http.Request) ([]byte, error) {
	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("request failed with status %d", res.StatusCode)
	}
	return body, nil
}
