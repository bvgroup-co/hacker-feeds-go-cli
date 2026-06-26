package feeds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type requestError struct {
	StatusCode int
	Body       string
	RetryAfter string
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
		return nil, requestError{StatusCode: res.StatusCode, Body: responseSnippet(body), RetryAfter: res.Header.Get("Retry-After")}
	}
	return body, nil
}

func responseSnippet(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}
	var decoded struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(trimmed, &decoded) == nil {
		switch {
		case decoded.Message != "" && decoded.Error != "":
			return decoded.Error + ": " + decoded.Message
		case decoded.Message != "":
			return decoded.Message
		case decoded.Error != "":
			return decoded.Error
		}
	}
	snippet := strings.Join(strings.Fields(string(trimmed)), " ")
	if len(snippet) > 160 {
		return snippet[:160]
	}
	return snippet
}
