package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestFetchAccessToken_Success(t *testing.T) {
	expectedAccessToken := "mockAccessToken"
	expectedRefreshToken := "mockRefreshToken"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Refresh-Token") != "mockRefreshToken" {
			t.Errorf("Expected refresh token to be 'mockRefreshToken', got %v", r.Header.Get("X-Refresh-Token"))
		}
		w.WriteHeader(http.StatusOK)
		response := LoginResponse{AccessToken: expectedAccessToken, RefreshToken: expectedRefreshToken}
		jsonResponse, _ := json.Marshal(response)
		w.Write(jsonResponse)
	}))
	defer server.Close()

	viper.Set("api_url", server.URL)
	viper.Set("refresh_token", "mockRefreshToken")

	resp, err := FetchAccessToken()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected a response, got nil")
	}

	if resp.AccessToken != expectedAccessToken {
		t.Errorf("Expected access token %v, got %v", expectedAccessToken, resp.AccessToken)
	}

	if resp.RefreshToken != expectedRefreshToken {
		t.Errorf("Expected refresh token %v, got %v", expectedRefreshToken, resp.RefreshToken)
	}
}

func TestFetchAccessToken_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	viper.Set("api_url", server.URL)
	viper.Set("refresh_token", "mockRefreshToken")

	resp, err := FetchAccessToken()

	if err == nil || err.Error() != "invalid refresh token" {
		t.Errorf("Expected error 'invalid refresh token', got %v", err)
	}

	if resp != nil {
		t.Errorf("Expected no response, got %v", resp)
	}
}

func TestFetchAccessToken_RequestError(t *testing.T) {
	viper.Set("api_url", "http://invalid-url")
	viper.Set("refresh_token", "mockRefreshToken")

	resp, err := FetchAccessToken()

	if err == nil {
		t.Errorf("Expected an error, got none")
	}

	if resp != nil {
		t.Errorf("Expected no response, got %v", resp)
	}
}

