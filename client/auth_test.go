package api

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestFetchAccessTokenHasOverallTimeout(t *testing.T) {
	originalClient := apiHTTPClient
	originalAPIURL := viper.GetString("api_url")
	originalRefreshToken := viper.GetString("refresh_token")
	t.Cleanup(func() {
		apiHTTPClient = originalClient
		viper.Set("api_url", originalAPIURL)
		viper.Set("refresh_token", originalRefreshToken)
	})

	const timeout = 20 * time.Millisecond
	apiHTTPClient = &http.Client{
		Timeout: timeout,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}),
	}
	viper.Set("api_url", "http://api.example")
	viper.Set("refresh_token", "refresh-token")

	start := time.Now()
	_, err := FetchAccessToken()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("FetchAccessToken() unexpectedly succeeded")
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("FetchAccessToken() error = %v, want timeout error", err)
	}
	if elapsed > time.Second {
		t.Fatalf("FetchAccessToken() took %v, want a prompt timeout", elapsed)
	}
}

func TestAPIHTTPClientUsesConfiguredTimeout(t *testing.T) {
	if apiHTTPClient.Timeout != apiRequestTimeout {
		t.Fatalf("API client timeout = %v, want %v", apiHTTPClient.Timeout, apiRequestTimeout)
	}
}
