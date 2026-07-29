package checks

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

func TestCLIChecksInterpolatesResolvedBaseURLInCommands(t *testing.T) {
	tests := []struct {
		name            string
		defaultBaseURL  string
		overrideBaseURL string
		want            string
	}{
		{
			name:           "manifest default",
			defaultBaseURL: "http://localhost:3000",
			want:           "http://localhost:3000",
		},
		{
			name:           "manifest default with trailing slash",
			defaultBaseURL: "http://localhost:3000/",
			want:           "http://localhost:3000",
		},
		{
			name:            "configured override",
			defaultBaseURL:  "http://localhost:3000",
			overrideBaseURL: "http://localhost:4000/",
			want:            "http://localhost:4000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cliData := api.CLIData{
				BaseURLDefault: tt.defaultBaseURL,
				Steps: []api.CLIStep{{
					CLICommand: &api.CLIStepCLICommand{
						Command: `echo '${baseURL}'`,
					},
				}},
			}
			ch := make(chan tea.Msg, 10)

			results := CLIChecks(cliData, tt.overrideBaseURL, ch)

			if got := results[0].CLICommandResult.Stdout; got != tt.want {
				t.Fatalf("command stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCLIChecksUsesOverrideParameterForHTTPRequestPreview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	previousOverride := viper.GetString("override_base_url")
	viper.Set("override_base_url", "http://localhost:1")
	t.Cleanup(func() {
		viper.Set("override_base_url", previousOverride)
	})

	cliData := api.CLIData{
		BaseURLDefault: "http://localhost:3000",
		Steps: []api.CLIStep{{
			HTTPRequest: &api.CLIStepHTTPRequest{
				Request: api.HTTPRequest{
					Method:  http.MethodGet,
					FullURL: api.BaseURLPlaceholder + "/health",
				},
			},
		}},
	}
	messageChannel := make(chan tea.Msg, 10)

	results := CLIChecks(cliData, server.URL+"/", messageChannel)

	startMessage, ok := (<-messageChannel).(messages.StartStepMsg)
	if !ok {
		t.Fatal("expected start step message")
	}
	if want := server.URL + "/health"; startMessage.URL != want {
		t.Fatalf("preview URL = %q, want %q", startMessage.URL, want)
	}
	if got := results[0].HTTPRequestResult.StatusCode; got != http.StatusNoContent {
		t.Fatalf("response status = %d, want %d", got, http.StatusNoContent)
	}
}

func TestApplySubmissionResultsMarksAllStepsAndTestsPassedWhenNoFailure(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}, {}}}},
		{HTTPRequest: &api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{}}}},
	}}

	got := applySubmissionResultsMessages(cliData, nil)
	want := []tea.Msg{
		messages.ResolveStepMsg{Index: 0, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 0, TestIndex: 0, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 0, TestIndex: 1, Passed: boolPtr(true)},
		messages.ResolveStepMsg{Index: 1, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 1, TestIndex: 0, Passed: boolPtr(true)},
	}

	assertMessages(t, got, want)
}

func TestApplySubmissionResultsStopsAfterFailedCLITest(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}}}},
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}, {}, {}}}},
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}}}},
	}}
	failure := &api.StructuredErrCLI{FailedStepIndex: 1, FailedTestIndex: 1}

	got := applySubmissionResultsMessages(cliData, failure)
	want := []tea.Msg{
		messages.ResolveStepMsg{Index: 0, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 0, TestIndex: 0, Passed: boolPtr(true)},
		messages.ResolveStepMsg{Index: 1, Passed: boolPtr(false)},
		messages.ResolveTestMsg{StepIndex: 1, TestIndex: 0, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 1, TestIndex: 1, Passed: boolPtr(false)},
	}

	assertMessages(t, got, want)
}

func TestApplySubmissionResultsStopsAfterFailedHTTPTest(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}}}},
		{HTTPRequest: &api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{}, {}, {}}}},
	}}
	failure := &api.StructuredErrCLI{FailedStepIndex: 1, FailedTestIndex: 1}

	got := applySubmissionResultsMessages(cliData, failure)
	want := []tea.Msg{
		messages.ResolveStepMsg{Index: 0, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 0, TestIndex: 0, Passed: boolPtr(true)},
		messages.ResolveStepMsg{Index: 1, Passed: boolPtr(false)},
		messages.ResolveTestMsg{StepIndex: 1, TestIndex: 0, Passed: boolPtr(true)},
		messages.ResolveTestMsg{StepIndex: 1, TestIndex: 1, Passed: boolPtr(false)},
	}

	assertMessages(t, got, want)
}

func applySubmissionResultsMessages(cliData api.CLIData, failure *api.StructuredErrCLI) []tea.Msg {
	ch := make(chan tea.Msg)
	go func() {
		defer close(ch)
		ApplySubmissionResults(cliData, failure, ch)
	}()

	var msgs []tea.Msg
	for msg := range ch {
		msgs = append(msgs, msg)
	}
	return msgs
}

func assertMessages(t *testing.T, got []tea.Msg, want []tea.Msg) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("messages = %#v, want %#v", got, want)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
