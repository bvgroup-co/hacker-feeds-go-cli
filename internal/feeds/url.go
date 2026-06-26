package feeds

import "strings"

func joinRawURL(base string, segments ...string) string {
	cleaned := make([]string, 0, len(segments)+1)
	cleaned = append(cleaned, strings.TrimRight(base, "/"))
	for _, segment := range segments {
		cleaned = append(cleaned, strings.Trim(segment, "/"))
	}
	return strings.Join(cleaned, "/")
}

func joinURLPath(basePath string, segments ...string) string {
	parts := []string{strings.Trim(basePath, "/")}
	for _, segment := range segments {
		parts = append(parts, strings.Trim(segment, "/"))
	}
	joined := strings.Join(parts, "/")
	return "/" + strings.Trim(joined, "/")
}
