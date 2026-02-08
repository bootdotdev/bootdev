package checks

import (
	"net/http"
	"strings"
	"time"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func CLIChecks(cliData api.CLIData, overrideBaseURL string, ch chan tea.Msg) (results []api.CLIStepResult) {
	client := &http.Client{}
	variables := make(map[string]string)
	results = make([]api.CLIStepResult, len(cliData.Steps))

	if cliData.BaseURLDefault == api.BaseURLOverrideRequired && overrideBaseURL == "" {
		cobra.CheckErr("lesson requires a base URL override: `bootdev configure base_url <url>`")
	}

	baseURL := overrideBaseURL
	if overrideBaseURL == "" {
		baseURL = cliData.BaseURLDefault
	}

	for i, step := range cliData.Steps {
		// This is the magic of the initial message sent before executing the test
		if step.CLICommand != nil {
			ch <- messages.StartStepMsg{
				CMD:       step.CLICommand.Command,
				TmdlQuery: step.CLICommand.StdoutFilterTmdl,
			}
		} else if step.HTTPRequest != nil {
			finalBaseURL := baseURL
			overrideURL := viper.GetString("override_base_url")
			if overrideURL != "" {
				finalBaseURL = overrideURL
			}
			fullURL := strings.Replace(step.HTTPRequest.Request.FullURL, api.BaseURLPlaceholder, finalBaseURL, 1)
			interpolatedURL := InterpolateVariables(fullURL, variables)

			ch <- messages.StartStepMsg{
				URL:               interpolatedURL,
				Method:            step.HTTPRequest.Request.Method,
				ResponseVariables: step.HTTPRequest.ResponseVariables,
			}
		}

		switch {
		case step.CLICommand != nil:
			result := runCLICommand(*step.CLICommand, variables)
			result.JqOutputs = collectStdoutJqOutputs(*step.CLICommand, result)
			results[i].CLICommandResult = &result

			sendCLICommandResults(ch, *step.CLICommand, result, i)
			handleSleep(step.CLICommand, ch)

		case step.HTTPRequest != nil:
			result := runHTTPRequest(client, baseURL, variables, *step.HTTPRequest)
			results[i].HTTPRequestResult = &result
			sendHTTPRequestResults(ch, *step.HTTPRequest, result, i)
			handleSleep(step.HTTPRequest, ch)

		default:
			cobra.CheckErr("unable to run lesson: missing step")
		}
	}
	return results
}

func sendCLICommandResults(ch chan tea.Msg, cmd api.CLIStepCLICommand, result api.CLICommandResult, index int) {
	for _, test := range cmd.Tests {
		ch <- messages.StartTestMsg{Text: prettyPrintCLICommand(test, result.Variables)}
	}

	for j := range cmd.Tests {
		ch <- messages.ResolveTestMsg{
			StepIndex: index,
			TestIndex: j,
		}
	}

	ch <- messages.ResolveStepMsg{
		Index: index,
		Result: &api.CLIStepResult{
			CLICommandResult: &result,
		},
	}
}

func sendHTTPRequestResults(ch chan tea.Msg, req api.CLIStepHTTPRequest, result api.HTTPRequestResult, index int) {
	for _, test := range req.Tests {
		ch <- messages.StartTestMsg{Text: prettyPrintHTTPTest(test, result.Variables)}
	}

	for j := range req.Tests {
		ch <- messages.ResolveTestMsg{
			StepIndex: index,
			TestIndex: j,
		}
	}

	ch <- messages.ResolveStepMsg{
		Index: index,
		Result: &api.CLIStepResult{
			HTTPRequestResult: &result,
		},
	}
}

func ApplySubmissionResults(cliData api.CLIData, failure *api.VerificationResultStructuredErrCLI, ch chan tea.Msg) {
	for i, step := range cliData.Steps {
		stepPass := true
		isFailedStep := false
		if failure != nil {
			stepPass = i < failure.FailedStepIndex
			isFailedStep = i == failure.FailedStepIndex
		}

		ch <- messages.ResolveStepMsg{
			Index:  i,
			Passed: &stepPass,
		}

		if step.CLICommand != nil {
			for j := range step.CLICommand.Tests {
				if isFailedStep && j > failure.FailedTestIndex {
					break
				}

				testPass := stepPass || (isFailedStep && j < failure.FailedTestIndex)
				ch <- messages.ResolveTestMsg{
					StepIndex: i,
					TestIndex: j,
					Passed:    &testPass,
				}
			}
		}
		if step.HTTPRequest != nil {
			for j := range step.HTTPRequest.Tests {
				if isFailedStep && j > failure.FailedTestIndex {
					break
				}

				testPass := stepPass || (isFailedStep && j < failure.FailedTestIndex)
				ch <- messages.ResolveTestMsg{
					StepIndex: i,
					TestIndex: j,
					Passed:    &testPass,
				}
			}
		}

		if !stepPass {
			break
		}
	}
}

func handleSleep(s api.Sleepable, ch chan tea.Msg) {
	sleepMs := s.GetSleepAfterMs()
	if sleepMs != nil && *sleepMs > 0 {
		ch <- messages.SleepMsg{DurationMs: *sleepMs}
		time.Sleep(time.Duration(*sleepMs) * time.Millisecond)
	}
}
