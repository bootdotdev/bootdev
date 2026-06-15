package version

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestFromProxy(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{name: "ok", status: http.StatusOK, body: `{"Version":"v1.2.3"}`, want: "v1.2.3"},
		{name: "non-200", status: http.StatusInternalServerError, body: "boom", wantErr: true},
		{name: "invalid json", status: http.StatusOK, body: "not json", wantErr: true},
		{name: "empty version", status: http.StatusOK, body: `{"Version":""}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			got, err := fetchLatestFromProxy(server.Client(), server.URL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("fetchLatestFromProxy() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("fetchLatestFromProxy() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("fetchLatestFromProxy() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A single transient failure should not fail the version check; the retry
// must recover once the proxy responds successfully.
func TestFetchLatestWithRetryRecoversFromTransientFailure(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			// Simulate a dropped/erroring proxy response.
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, err := hj.Hijack()
				if err == nil {
					conn.Close()
					return
				}
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"Version":"v1.2.3"}`))
	}))
	defer server.Close()

	got, err := fetchLatestWithRetry(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("fetchLatestWithRetry() unexpected error: %v", err)
	}
	if got != "v1.2.3" {
		t.Fatalf("fetchLatestWithRetry() = %q, want %q", got, "v1.2.3")
	}
	if calls < 2 {
		t.Fatalf("expected a retry, got %d call(s)", calls)
	}
}
