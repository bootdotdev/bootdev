package checks

import (
	"testing"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	tea "github.com/charmbracelet/bubbletea"
)

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
	done := make(chan struct{})
	go func() {
		defer close(ch)
		defer close(done)
		ApplySubmissionResults(cliData, failure, ch)
	}()

	var msgs []tea.Msg
	for msg := range ch {
		msgs = append(msgs, msg)
	}
	<-done
	return msgs
}

func assertMessages(t *testing.T, got []tea.Msg, want []tea.Msg) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d\ngot: %#v\nwant: %#v", len(got), len(want), got, want)
	}
	for i := range want {
		assertMessage(t, i, got[i], want[i])
	}
}

func assertMessage(t *testing.T, index int, got tea.Msg, want tea.Msg) {
	t.Helper()

	switch want := want.(type) {
	case messages.ResolveStepMsg:
		got, ok := got.(messages.ResolveStepMsg)
		if !ok {
			t.Fatalf("message %d = %T, want %T", index, got, want)
		}
		if got.Index != want.Index || !sameBoolPtr(got.Passed, want.Passed) {
			t.Fatalf("message %d = %#v, want %#v", index, got, want)
		}
	case messages.ResolveTestMsg:
		got, ok := got.(messages.ResolveTestMsg)
		if !ok {
			t.Fatalf("message %d = %T, want %T", index, got, want)
		}
		if got.StepIndex != want.StepIndex || got.TestIndex != want.TestIndex || !sameBoolPtr(got.Passed, want.Passed) {
			t.Fatalf("message %d = %#v, want %#v", index, got, want)
		}
	default:
		t.Fatalf("unsupported wanted message type %T", want)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func sameBoolPtr(a *bool, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
