package render

import (
	"fmt"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
)

func renderStepDetail(label, detail string) string {
	if label != "" {
		return subheaderStyle.Render(label+":") + " " + detail
	}
	return detail
}

func indentOutput(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for index, line := range lines {
		if line != "" {
			lines[index] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func renderDetailedStep(step stepModel, number int, isSubmit bool, failure *api.StructuredErrCLI) string {
	var str strings.Builder
	headingStyle := headerStyle.Bold(true)
	fmt.Fprintf(&str, "%s\n%s\n", headingStyle.Render(fmt.Sprintf("STEP %d · %s", number, step.description)), renderStepDetail(step.detailLabel, step.detail))
	if step.noPenaltyOnFail {
		str.WriteString(secondaryStyle.Render("Safe step") + "\n")
	}
	gradingFailureHandled := false
	if isSubmit && len(step.tests) > 0 {
		str.WriteString("\n  " + subheaderStyle.Render("Checks:") + "\n")
		for index, test := range step.tests {
			check, explainsFailure := renderCheckDetails(step, index, test.passed)
			fmt.Fprintf(&str, "\n%s\n", indentOutput(check, 4))
			if test.passed != nil && !*test.passed && failure != nil &&
				failure.FailedStepIndex == number-1 && failure.FailedTestIndex == index {
				gradingFailureHandled = true
				if !explainsFailure && failure.ErrorMessage != "" {
					str.WriteString(renderGradingError(failure.ErrorMessage, 6))
				}
			}
		}
	}
	if isSubmit && !gradingFailureHandled && failure != nil && failure.FailedStepIndex == number-1 && failure.ErrorMessage != "" {
		stepErrorShown := false
		if result := step.result; result != nil {
			stepErrorShown = result.DependencyFailure != nil ||
				(result.CLICommandResult != nil && result.CLICommandResult.Err != "") ||
				(result.HTTPRequestResult != nil && result.HTTPRequestResult.Err != "")
		}
		if !stepErrorShown {
			str.WriteByte('\n')
			str.WriteString(renderGradingError(failure.ErrorMessage, 2))
		}
	}
	if step.result != nil {
		str.WriteString(renderStepResult(step, isSubmit))
	}
	if step.sleepAfter != "" {
		fmt.Fprintf(&str, "\n  %s\n", informationStyle.Render(step.sleepAfter))
	}
	str.WriteByte('\n')
	return str.String()
}

func renderStatus(status string) string {
	switch status {
	case "PASS":
		return passStyle.Bold(true).Render(status)
	case "FAIL":
		return failStyle.Bold(true).Render(status)
	default:
		return secondaryStyle.Render(status)
	}
}

func renderGradingError(message string, spaces int) string {
	return indentOutput(warningStyle.Render("Error:")+" "+truncateVisualOutput(message), spaces) + "\n"
}

// Only simple numeric comparisons fully explain the grader's failure.
// JSON extraction, types, and truncated values may need additional diagnostics.
func renderCheckDetails(step stepModel, index int, passed *bool) (string, bool) {
	if step.result != nil {
		if result := step.result.CLICommandResult; result != nil && index < len(result.Command.Tests) {
			test := result.Command.Tests[index]
			if isExitCodeOnlyCheck(test) {
				return renderCheckValues(renderCheckTitle("Exit code", index, passed), fmt.Sprint(*test.ExitCode), fmt.Sprint(result.ExitCode), passed != nil), passed != nil
			}
		}
		if result := step.result.HTTPRequestResult; result != nil && index < len(result.Request.Tests) {
			test := result.Request.Tests[index]
			if test.JSONValue != nil && test.StatusCode == nil && test.BodyContains == nil && test.BodyContainsNone == nil && test.HeadersEqual == nil && test.HeadersContain == nil && test.TrailersEqual == nil && test.TrailersContain == nil {
				if detail := renderJSONMismatch(*test.JSONValue, *result, passed != nil, renderCheckTitle("JSON "+test.JSONValue.Path, index, passed)); detail != "" {
					return detail, false
				}
			}
			if test.StatusCode != nil && test.BodyContains == nil && test.BodyContainsNone == nil && test.HeadersEqual == nil && test.HeadersContain == nil && test.TrailersEqual == nil && test.TrailersContain == nil && test.JSONValue == nil {
				return renderCheckValues(renderCheckTitle("HTTP status", index, passed), fmt.Sprint(*test.StatusCode), fmt.Sprint(result.StatusCode), passed != nil), passed != nil
			}
		}
	}
	return renderCheckTitle(step.tests[index].text, index, passed), false
}

func isExitCodeOnlyCheck(test api.CLICommandTest) bool {
	return test.ExitCode != nil && test.StdoutContainsAll == nil && test.StdoutContainsNone == nil && test.StdoutLinesGT == nil && test.StdoutJq == nil
}

func renderCheckTitle(title string, index int, passed *bool) string {
	numberedTitle := fmt.Sprintf("%d. %s", index+1, title)
	if passed == nil {
		return numberedTitle
	}
	status := "PASS"
	if !*passed {
		status = "FAIL"
	}
	return renderStatus(status) + " · " + numberedTitle
}

func renderCheckValues(title, expected, actual string, evaluated bool) string {
	detail := fmt.Sprintf("%s\n  %s %s", title, secondaryStyle.Render("Expected:"), expected)
	if evaluated {
		detail += fmt.Sprintf("\n  %s   %s", secondaryStyle.Render("Actual:"), actual)
	}
	return detail
}
