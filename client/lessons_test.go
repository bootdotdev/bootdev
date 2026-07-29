package api

import (
	"strings"
	"testing"

	"github.com/goccy/go-json"
)

func TestCLICommandResultOmitsStderrFromJSON(t *testing.T) {
	payload, err := json.Marshal(CLICommandResult{
		Stdout: "submitted output",
		Stderr: "local diagnostic",
	})
	if err != nil {
		t.Fatalf("marshal CLI command result: %v", err)
	}

	if strings.Contains(string(payload), "local diagnostic") {
		t.Fatalf("submission payload unexpectedly contains stderr: %s", payload)
	}
	if !strings.Contains(string(payload), "submitted output") {
		t.Fatalf("submission payload is missing stdout: %s", payload)
	}
}

func TestHTTPRequestResultSerializesFetchError(t *testing.T) {
	const fetchErr = "Failed to fetch: connection refused"

	payload, err := json.Marshal(HTTPRequestResult{Err: fetchErr})
	if err != nil {
		t.Fatalf("marshal HTTP request result: %v", err)
	}

	var submitted struct {
		FetchErr *string
	}
	if err := json.Unmarshal(payload, &submitted); err != nil {
		t.Fatalf("unmarshal submission payload: %v", err)
	}
	if submitted.FetchErr == nil || *submitted.FetchErr != fetchErr {
		t.Fatalf("FetchErr = %v, want %q; payload: %s", submitted.FetchErr, fetchErr, payload)
	}
}

func TestHTTPRequestResultOmitsEmptyFetchError(t *testing.T) {
	payload, err := json.Marshal(HTTPRequestResult{})
	if err != nil {
		t.Fatalf("marshal HTTP request result: %v", err)
	}

	var submitted map[string]any
	if err := json.Unmarshal(payload, &submitted); err != nil {
		t.Fatalf("unmarshal submission payload: %v", err)
	}
	if _, ok := submitted["FetchErr"]; ok {
		t.Fatalf("submission payload unexpectedly contains an empty FetchErr: %s", payload)
	}
}
