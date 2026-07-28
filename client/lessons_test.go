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
