package render

import (
	"fmt"
	"strings"
)

func renderStepDetail(detail string) string {
	for _, label := range []string{"Request:", "Command:"} {
		if strings.HasPrefix(detail, label) {
			return subheaderStyle.Render(label) + strings.TrimPrefix(detail, label)
		}
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

func renderDetailedStep(step stepModel, number int, isSubmit bool) string {
	var str strings.Builder
	headingStyle := headerStyle.Bold(true)
	fmt.Fprintf(&str, "%s\n%s\n", headingStyle.Render(fmt.Sprintf("STEP %d · %s", number, step.description)), renderStepDetail(step.detail))
	if step.noPenaltyOnFail {
		str.WriteString(secondaryStyle.Render("Safe step") + "\n")
	}
	// Keep execution order intact while bringing failed assertions to the front.
	for _, failed := range []bool{true, false} {
		for index, test := range step.tests {
			if !isSubmit || test.passed == nil || (!*test.passed) != failed {
				continue
			}
			status := "PASS"
			if failed {
				status = "FAIL"
			}
			check := renderCheck(step, index, status)
			check = renderStatus(status) + strings.TrimPrefix(check, status)
			check = strings.ReplaceAll(check, "\n  Expected:", "\n  "+secondaryStyle.Render("Expected:"))
			check = strings.ReplaceAll(check, "\n  Actual:", "\n  "+secondaryStyle.Render("Actual:"))
			fmt.Fprintf(&str, "\n%s\n", indentOutput(check, 2))
		}
	}
	if step.result != nil {
		str.WriteString(renderStepResult(step, isSubmit))
	}
	unknownHeading := false
	for index, test := range step.tests {
		if !isSubmit || test.passed != nil {
			continue
		}
		if !unknownHeading {
			heading := "Checks"
			if isSubmit {
				heading = "Checks not evaluated"
			}
			str.WriteString("\n  " + subheaderStyle.Render(heading) + "\n")
			unknownHeading = true
		} else {
			str.WriteByte('\n')
		}
		fmt.Fprintf(&str, "%s\n", indentOutput(renderCheck(step, index, ""), 4))
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
	case "FAIL", "FAILED":
		return failStyle.Bold(true).Render(status)
	default:
		return secondaryStyle.Render(status)
	}
}

func renderCheck(step stepModel, index int, status string) string {
	statusPrefix := ""
	if status != "" {
		statusPrefix = status + " · "
	}
	if step.result != nil {
		if result := step.result.CLICommandResult; result != nil && index < len(result.Command.Tests) {
			test := result.Command.Tests[index]
			if test.ExitCode != nil && test.StdoutContainsAll == nil && test.StdoutContainsNone == nil && test.StdoutLinesGT == nil && test.StdoutJq == nil {
				return fmt.Sprintf("%sExit code\n  Expected: %d\n  Actual:   %d", statusPrefix, *test.ExitCode, result.ExitCode)
			}
		}
		if result := step.result.HTTPRequestResult; result != nil && index < len(result.Request.Tests) {
			test := result.Request.Tests[index]
			if test.JSONValue != nil && test.StatusCode == nil && test.BodyContains == nil && test.BodyContainsNone == nil && test.HeadersEqual == nil && test.HeadersContain == nil && test.TrailersEqual == nil && test.TrailersContain == nil {
				if detail := renderJSONMismatch(*test.JSONValue, *result); detail != "" {
					return statusPrefix + detail
				}
			}
			if test.StatusCode != nil && test.BodyContains == nil && test.BodyContainsNone == nil && test.HeadersEqual == nil && test.HeadersContain == nil && test.TrailersEqual == nil && test.TrailersContain == nil && test.JSONValue == nil {
				return fmt.Sprintf("%sHTTP status\n  Expected: %d\n  Actual:   %d", statusPrefix, *test.StatusCode, result.StatusCode)
			}
		}
	}
	return statusPrefix + step.tests[index].text
}
