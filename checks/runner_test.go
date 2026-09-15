package checks

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
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
			results, err := CLIChecks(cliData, RunOptions{OverrideBaseURL: tt.overrideBaseURL}, func(tea.Msg) {})
			if err != nil {
				t.Fatalf("CLIChecks() error = %v", err)
			}

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
	var sent []tea.Msg
	results, err := CLIChecks(cliData, RunOptions{OverrideBaseURL: server.URL + "/"}, func(msg tea.Msg) {
		sent = append(sent, msg)
	})
	if err != nil {
		t.Fatalf("CLIChecks() error = %v", err)
	}

	startMessage, ok := sent[0].(messages.StartStepMsg)
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

func TestCLIChecksReturnsManifestErrors(t *testing.T) {
	tests := []struct {
		name string
		data api.CLIData
		want string
	}{
		{
			name: "missing required base URL override",
			data: api.CLIData{BaseURLDefault: api.BaseURLOverrideRequired},
			want: "lesson requires a base URL override",
		},
		{
			name: "missing step type",
			data: api.CLIData{Steps: []api.CLIStep{{}}},
			want: "must contain exactly one command or HTTP request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CLIChecks(tt.data, RunOptions{}, func(tea.Msg) {})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CLIChecks() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestApplySubmissionResultsMarksAllStepsAndTestsPassedWhenNoFailure(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}, {}}}},
		{HTTPRequest: &api.CLIStepHTTPRequest{Tests: []api.HTTPRequestTest{{}}}},
	}}

	steps, tests := submissionStatuses(cliData, nil)
	if !maps.Equal(steps, map[int]bool{0: true, 1: true}) {
		t.Fatalf("step statuses = %v, want both steps passed", steps)
	}
	if !maps.Equal(tests, map[[2]int]bool{{0, 0}: true, {0, 1}: true, {1, 0}: true}) {
		t.Fatalf("test statuses = %v, want all three tests passed", tests)
	}
}

func TestApplySubmissionResultsStopsAfterFailedCLITest(t *testing.T) {
	cliData := api.CLIData{Steps: []api.CLIStep{
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}}}},
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}, {}, {}}}},
		{CLICommand: &api.CLIStepCLICommand{Tests: []api.CLICommandTest{{}}}},
	}}
	failure := &api.StructuredErrCLI{FailedStepIndex: 1, FailedTestIndex: 1}

	steps, tests := submissionStatuses(cliData, failure)
	if !maps.Equal(steps, map[int]bool{0: true, 1: false}) {
		t.Fatalf("step statuses = %v, want first passed, second failed, third unresolved", steps)
	}
	if !maps.Equal(tests, map[[2]int]bool{{0, 0}: true, {1, 0}: true, {1, 1}: false}) {
		t.Fatalf("test statuses = %v, want tests before failure passed and later tests unresolved", tests)
	}
}

func submissionStatuses(data api.CLIData, failure *api.StructuredErrCLI) (map[int]bool, map[[2]int]bool) {
	steps := map[int]bool{}
	tests := map[[2]int]bool{}
	ApplySubmissionResults(data, failure, func(msg tea.Msg) {
		switch msg := msg.(type) {
		case messages.ResolveStepMsg:
			if msg.Passed != nil {
				steps[msg.Index] = *msg.Passed
			}
		case messages.ResolveTestMsg:
			if msg.Passed != nil {
				tests[[2]int{msg.StepIndex, msg.TestIndex}] = *msg.Passed
			}
		}
	})
	return steps, tests
}

func TestCLIChecksExplicitShell(t *testing.T) {
	for _, tt := range []struct {
		shell   string
		command string
	}{
		{shell: "sh", command: "printf '%s' 'hello from shell'"},
		{shell: "pwsh", command: "Write-Output ('hello from ' + 'shell')"},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			if _, err := exec.LookPath(tt.shell); err != nil {
				t.Skipf("%s is not installed: %v", tt.shell, err)
			}
			data := api.CLIData{Steps: []api.CLIStep{{
				CLICommand: &api.CLIStepCLICommand{Command: tt.command},
			}}}
			results, err := CLIChecks(data, RunOptions{Shell: tt.shell}, func(tea.Msg) {})
			if err != nil {
				t.Fatal(err)
			}
			got := results[0].CLICommandResult
			if got.Err != "" || got.ExitCode != 0 || got.Stdout != "hello from shell" {
				t.Fatalf("unexpected shell result: %#v", got)
			}
		})
	}
}

func TestCLIChecksRejectsShellBeforeRunningSteps(t *testing.T) {
	for _, tt := range []struct {
		shell string
		want  string
	}{
		{shell: "bash", want: "unsupported shell"},
		{shell: "powershell", want: "unsupported shell"},
		{shell: "sh", want: "is unavailable"},
		{shell: "pwsh", want: "is unavailable"},
	} {
		t.Run(tt.shell, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			data := api.CLIData{Steps: []api.CLIStep{{
				HTTPRequest: &api.CLIStepHTTPRequest{},
			}, {
				CLICommand: &api.CLIStepCLICommand{Command: "echo should not run"},
			}}}
			_, err := CLIChecks(data, RunOptions{Shell: tt.shell}, func(tea.Msg) {
				t.Fatal("step started before shell validation")
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
