package feeds

import (
	"fmt"
	"io"
	"net/http"
)

type requestError struct {
	StatusCode int
}

func (err requestError) Error() string {
	return fmt.Sprintf("request failed with status %d", err.StatusCode)
}

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
		return nil, requestError{StatusCode: res.StatusCode}
	}
	return body, nil
}
