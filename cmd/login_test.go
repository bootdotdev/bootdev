package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testFrontendURL = "https://boot.dev"

func TestLoginHTTPHandlerAcceptsConfiguredOrigin(t *testing.T) {
	inputChan := make(chan string, 1)
	handler := newLoginHTTPHandler(inputChan, testFrontendURL)
	request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("valid-code"))
	request.Header.Set("Origin", testFrontendURL)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if code := <-inputChan; code != "valid-code" {
		t.Fatalf("login code = %q, want valid-code", code)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != testFrontendURL {
		t.Fatalf("allowed origin = %q, want %q", origin, testFrontendURL)
	}
}

func TestLoginHTTPHandlerRejectsUnexpectedOrigin(t *testing.T) {
	inputChan := make(chan string, 1)
	handler := newLoginHTTPHandler(inputChan, testFrontendURL)
	request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("attacker-code"))
	request.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	select {
	case code := <-inputChan:
		t.Fatalf("unexpected login code accepted: %q", code)
	default:
	}
}

func TestLoginHTTPHandlerRejectsMissingOrigin(t *testing.T) {
	inputChan := make(chan string, 1)
	handler := newLoginHTTPHandler(inputChan, testFrontendURL)
	request := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("attacker-code"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestLoginHTTPHandlerLimitsCodeSize(t *testing.T) {
	inputChan := make(chan string, 1)
	handler := newLoginHTTPHandler(inputChan, testFrontendURL)
	body := strings.NewReader(strings.Repeat("a", maxLoginCodeBytes+1))
	request := httptest.NewRequest(http.MethodPost, "/submit", body)
	request.Header.Set("Origin", testFrontendURL)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	select {
	case code := <-inputChan:
		t.Fatalf("unexpected login code accepted: %q", code)
	default:
	}
}
