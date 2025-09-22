package auth

import (
	"testing"
)

func TestGetAPIKey(t *testing.T) {
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"ApiKey some-api-key"}

	key, err := GetAPIKey(headers)
	if err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}

	if key != "some-api-key" {
		t.Fatalf("expected key 'wrong-api-key', but got '%s'", key)
	}
}

func TestGetAPIKeyNoAuth(t *testing.T) {
	headers := make(map[string][]string)

	key, err := GetAPIKey(headers)
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}

	if key != "" {
		t.Fatalf("expected empty key, but got '%s'", key)
	}
}
