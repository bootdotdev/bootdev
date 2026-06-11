package checks

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestInterpolateVariables(t *testing.T) {
	got := InterpolateVariables(
		"${baseURL}/users/${id}?missing=${missing}",
		map[string]string{"baseURL": "http://localhost:8080", "id": "42"},
	)
	want := "http://localhost:8080/users/42?missing=${missing}"
	if got != want {
		t.Fatalf("InterpolateVariables() = %q, want %q", got, want)
	}
}

func TestInterpolationNames(t *testing.T) {
	got := InterpolationNames("${baseURL}/users/${id}/${id}")
	want := []string{"baseURL", "id", "id"}
	if len(got) != len(want) {
		t.Fatalf("InterpolationNames() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("InterpolationNames() = %#v, want %#v", got, want)
		}
	}
}

func TestRunHTTPRequestInterpolatesRequestAndCapturesResponseVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
			return
		}
		if r.URL.Path != "/users/42" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/users/42")
			return
		}
		if r.Header.Get("X-User-ID") != "42" {
			t.Errorf("X-User-ID = %q, want %q", r.Header.Get("X-User-ID"), "42")
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "pass" {
			t.Errorf("BasicAuth() = %q, %q, %v; want user, pass, true", username, password, ok)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed reading request body: %v", err)
			return
		}
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("failed unmarshalling request body %q: %v", string(body), err)
			return
		}
		if payload["message"] != "hello Theo" {
			t.Errorf("message = %q, want %q", payload["message"], "hello Theo")
			return
		}

		w.Header().Set("X-Request-OK", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":"abc123"}`))
	}))
	defer server.Close()

	variables := map[string]string{
		"id":   "42",
		"name": "Theo",
	}
	requestStep := api.CLIStepHTTPRequest{
		ResponseVariables: []api.HTTPRequestResponseVariable{{Name: "token", Path: ".token"}},
		Request: api.HTTPRequest{
			Method:  http.MethodPost,
			FullURL: api.BaseURLPlaceholder + "/users/${id}",
			Headers: map[string]string{
				"X-User-ID": "${id}",
			},
			BodyJSON: map[string]any{
				"message": "hello ${name}",
			},
			BasicAuth: &api.HTTPBasicAuth{Username: "user", Password: "pass"},
		},
	}

	result := runHTTPRequest(server.Client(), server.URL, variables, requestStep)
	if result.Err != "" {
		t.Fatalf("unexpected request error: %s", result.Err)
	}
	if result.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusCreated)
	}
	if result.ResponseHeaders["X-Request-Ok"] != "yes" {
		t.Fatalf("ResponseHeaders[X-Request-Ok] = %q, want %q", result.ResponseHeaders["X-Request-Ok"], "yes")
	}
	if result.BodyString != `{"token":"abc123"}` {
		t.Fatalf("BodyString = %q, want token response", result.BodyString)
	}
	if result.Variables["token"] != "abc123" {
		t.Fatalf("captured token = %q, want %q", result.Variables["token"], "abc123")
	}
	if result.Variables["id"] != "42" {
		t.Fatalf("original variable id = %q, want %q", result.Variables["id"], "42")
	}
}

func TestTruncateAndStringifyBodyCapsBinaryBody(t *testing.T) {
	body := []byte(strings.Repeat("a", 20*1024))
	body[0] = 0

	got := truncateAndStringifyBody(body)
	if len(got) != 16*1024 {
		t.Fatalf("len(truncateAndStringifyBody(binary)) = %d, want %d", len(got), 16*1024)
	}
}

func TestRunHTTPRequestCapturesResponseHeaderVariableAndDoesNotFollowRedirect(t *testing.T) {
	followRedirects := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Errorf("path = %q, want /login", r.URL.Path)
			return
		}

		w.Header().Set("Set-Cookie", "session_id=abc123; Path=/; HttpOnly")
		w.Header().Set("Location", "/account")
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte("Found. Redirecting to /account"))
	}))
	defer server.Close()

	variables := map[string]string{}
	requestStep := api.CLIStepHTTPRequest{
		ResponseHeaderVariables: []api.HTTPRequestResponseHeaderVariable{{
			Name:   "sessionID",
			Header: "Set-Cookie",
			Regex:  "session_id=([^;]+)",
		}},
		Request: api.HTTPRequest{
			Method:          http.MethodPost,
			FullURL:         api.BaseURLPlaceholder + "/login",
			FollowRedirects: &followRedirects,
			BodyForm: map[string]string{
				"email":    "pacifica@example.com",
				"password": "password123",
				"returnTo": "/account",
			},
		},
	}

	result := runHTTPRequest(server.Client(), server.URL, variables, requestStep)
	if result.Err != "" {
		t.Fatalf("unexpected request error: %s", result.Err)
	}
	if result.StatusCode != http.StatusFound {
		t.Fatalf("StatusCode = %d, want %d", result.StatusCode, http.StatusFound)
	}
	if result.ResponseHeaders["Set-Cookie"] == "" {
		t.Fatalf("expected Set-Cookie response header")
	}
	if result.Variables["sessionID"] != "abc123" {
		t.Fatalf("captured sessionID = %q, want abc123", result.Variables["sessionID"])
	}
}

func TestParseVariablesLeavesMissingValuesUnset(t *testing.T) {
	variables := map[string]string{}
	err := parseVariables(
		[]byte(`{"token":"abc123","missing":null}`),
		[]api.HTTPRequestResponseVariable{
			{Name: "token", Path: ".token"},
			{Name: "missing", Path: ".missing"},
			{Name: "notFound", Path: ".not_found"},
		},
		variables,
	)
	if err != nil {
		t.Fatalf("unexpected parseVariables error: %v", err)
	}
	if variables["token"] != "abc123" {
		t.Fatalf("token = %q, want abc123", variables["token"])
	}
	if _, ok := variables["missing"]; ok {
		t.Fatalf("expected null variable to remain unset")
	}
	if _, ok := variables["notFound"]; ok {
		t.Fatalf("expected missing variable to remain unset")
	}
}

func TestParseVariablesCapturesBodyRegex(t *testing.T) {
	variables := map[string]string{}
	err := parseVariables(
		[]byte(`<a href="/password-reset/abc123">reset</a>`),
		[]api.HTTPRequestResponseVariable{
			{Name: "resetToken", BodyRegex: `/password-reset/([a-z0-9]+)`},
		},
		variables,
	)
	if err != nil {
		t.Fatalf("unexpected parseVariables error: %v", err)
	}
	if variables["resetToken"] != "abc123" {
		t.Fatalf("resetToken = %q, want abc123", variables["resetToken"])
	}
}

func TestParseVariablesRequiresCaptureSource(t *testing.T) {
	variables := map[string]string{}
	err := parseVariables(
		[]byte(`{"token":"abc123"}`),
		[]api.HTTPRequestResponseVariable{{Name: "token"}},
		variables,
	)
	if err == nil {
		t.Fatal("expected parseVariables error")
	}
	if err.Error() != "invalid response variable configuration" {
		t.Fatalf("error = %q, want invalid response variable configuration", err.Error())
	}
}

func TestParseHeaderVariablesLeavesMissingValuesUnset(t *testing.T) {
	variables := map[string]string{}
	err := parseHeaderVariables(
		map[string]string{"Set-Cookie": "session_id=abc123; Path=/; HttpOnly"},
		[]api.HTTPRequestResponseHeaderVariable{
			{Name: "sessionID", Header: "Set-Cookie", Regex: "session_id=([^;]+)"},
			{Name: "missingHeader", Header: "X-Missing"},
			{Name: "missingMatch", Header: "Set-Cookie", Regex: "missing=([^;]+)"},
		},
		variables,
	)
	if err != nil {
		t.Fatalf("unexpected parseHeaderVariables error: %v", err)
	}
	if variables["sessionID"] != "abc123" {
		t.Fatalf("sessionID = %q, want abc123", variables["sessionID"])
	}
	if _, ok := variables["missingHeader"]; ok {
		t.Fatalf("expected missing header variable to remain unset")
	}
	if _, ok := variables["missingMatch"]; ok {
		t.Fatalf("expected non-matching header variable to remain unset")
	}
}
