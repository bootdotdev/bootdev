package auth

import (
	"errors"
	"net/http"
	"strings"
)

// GetAPIKey extracts an API Key from the headers of an HTTP request
// Example:
// Authorization: ApiKey {insert apikey here}
func GetAPIKey(headers http.Header) (string, error) {
	s := headers.Get("Authorization")
	if s == "" {
		return "", errors.New("not authorized")
	}

	parts := strings.Split(s, " ")
	if len(parts) != 2 {
		return "", errors.New("malformed auth header")
	}

	if parts[0] != "ApiKey" {
		return "", errors.New("malformed auth header")
	}

	return parts[1], nil
}
