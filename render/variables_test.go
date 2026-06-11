package render

import (
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestHTTPVariableSections(t *testing.T) {
	result := api.HTTPRequestResult{
		Variables: map[string]string{
			"authToken":  "token-123",
			"resetToken": "reset-123",
			"shortCode":  "abc123",
			"sessionID":  "session-123",
		},
		Request: api.CLIStepHTTPRequest{
			ResponseVariables: []api.HTTPRequestResponseVariable{
				{Name: "shortCode", Path: ".short_code"},
				{Name: "resetToken", BodyRegex: `/password-reset/([a-z0-9]+)`},
				{Name: "missingCode", Path: ".missing_code"},
				{Name: "missingResetToken", BodyRegex: `/missing/([a-z0-9]+)`},
			},
			ResponseHeaderVariables: []api.HTTPRequestResponseHeaderVariable{
				{Name: "sessionID", Header: "Set-Cookie", Regex: "session_id=([^;]+)"},
				{Name: "missingSessionID", Header: "Set-Cookie", Regex: "missing=([^;]+)"},
			},
			Request: api.HTTPRequest{
				FullURL: "${baseURL}/api/links/${shortCode}",
				Headers: map[string]string{
					"Authorization": "Bearer ${authToken}",
				},
			},
		},
	}

	got := renderVariableSection("Variables Saved", savedVariablesForHTTPResult(result))
	got += renderVariableSection("Variables Missing", missingSaveVariablesForHTTPResult(result))
	available, expectsVariables := availableVariablesForHTTPResult(result)
	if !expectsVariables {
		t.Fatalf("expected HTTP request to use variables")
	}
	got += renderVariableSection("Variables Available", available)

	wantContains := []string{
		"Variables Saved:",
		"resetToken: reset-123 (Response Body pattern)",
		"sessionID: session-123 (Response Header Set-Cookie pattern)",
		"shortCode: abc123 (JSON Body .short_code)",
		"Variables Missing:",
		"missingCode: [not found] (JSON Body .missing_code)",
		"missingResetToken: [not found] (Response Body pattern)",
		"missingSessionID: [not found] (Response Header Set-Cookie pattern)",
		"Variables Available:",
		"authToken: token-123 (Request Header \"Authorization\")",
		"shortCode: abc123 (Request URL)",
	}

	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "baseURL") {
		t.Fatalf("did not expect special baseURL placeholder in:\n%s", got)
	}
}

func TestAvailableVariablesPrintsNotFoundWhenExpectedButUnavailable(t *testing.T) {
	result := api.CLICommandResult{
		Variables: map[string]string{},
		Command: api.CLIStepCLICommand{
			Command: "curl ${url}",
		},
	}

	available, expectsVariables := availableVariablesForCLIResult(result)
	if !expectsVariables {
		t.Fatalf("expected CLI command to use variables")
	}
	got := renderVariableSection("Variables Available", available)

	if !strings.Contains(got, "Variables Available:") {
		t.Fatalf("expected Variables Available section in:\n%s", got)
	}
	if !strings.Contains(got, "url: [not found] (Command)") {
		t.Fatalf("expected missing url in:\n%s", got)
	}
}

func TestCLIAvailableVariables(t *testing.T) {
	result := api.CLICommandResult{
		Variables: map[string]string{
			"url": "http://localhost:42069",
		},
		Command: api.CLIStepCLICommand{
			Command: "curl ${url}",
			Tests: []api.CLICommandTest{
				{StdoutContainsAll: []string{"${expected}"}},
			},
		},
	}

	available, expectsVariables := availableVariablesForCLIResult(result)
	if !expectsVariables {
		t.Fatalf("expected CLI command to use variables")
	}
	got := renderVariableSection("Variables Available", available)

	if !strings.Contains(got, "url: http://localhost:42069 (Command)") {
		t.Fatalf("expected url entry in:\n%s", got)
	}
	if !strings.Contains(got, "expected: [not found] (Stdout Contains Test)") {
		t.Fatalf("expected missing expected entry in:\n%s", got)
	}
}
