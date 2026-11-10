package render

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	api "github.com/bootdotdev/bootdev/client"
)

const safeStepIcon = "🛡︎"

func renderJqOutputs(outputs []api.CLICommandJqOutput) string {
	if len(outputs) == 0 {
		return ""
	}

	var str strings.Builder
	str.WriteString("\n " + subheaderStyle.Render("> jq output:") + "\n\n")
	for _, output := range outputs {
		str.WriteString(secondaryStyle.Render(fmt.Sprintf("Query: %s", output.Query)))
		str.WriteByte('\n')
		if output.Error != "" {
			str.WriteString(warningStyle.Render("Error:") + " " + output.Error)
			str.WriteByte('\n')
			str.WriteByte('\n')
			continue
		}
		if len(output.Results) == 0 {
			str.WriteString(secondaryStyle.Render("Results: [none]"))
			str.WriteByte('\n')
			str.WriteByte('\n')
			continue
		}
		str.WriteString(secondaryStyle.Render("Results:"))
		str.WriteByte('\n')
		for _, line := range output.Results {
			str.WriteString(secondaryStyle.Render("  - " + line))
			str.WriteByte('\n')
		}
		str.WriteByte('\n')
	}
	return str.String()
}

func (m rootModel) View() string {
	if m.clear {
		return ""
	}
	s := m.spinner.View()
	var str strings.Builder
	str.WriteString(renderRunHeader(m.runContext))
	failedStepIndex := -1
	if m.isSubmit && m.failure != nil && m.failure.FailedStepIndex >= 0 && m.failure.FailedStepIndex < len(m.steps) {
		failedStepIndex = m.failure.FailedStepIndex
	}
	for i, step := range m.steps {
		if m.finalized && !m.verbose && failedStepIndex >= 0 && i > failedStepIndex {
			break
		}

		showAllDetails := m.finalized && (m.verbose || !m.isSubmit)
		failed := step.passed != nil && !*step.passed
		if showAllDetails || (failed && m.finalized) {
			dividerWidth := m.terminalWidth
			if dividerWidth <= 0 {
				dividerWidth = 80
			}
			str.WriteString(secondaryStyle.Render(strings.Repeat("─", dividerWidth)) + "\n")
			str.WriteString(renderDetailedStep(step, i+1, m.isSubmit))
		} else {
			step.description = fmt.Sprintf("Step %d · %s", i+1, step.description)
			str.WriteString(renderCompactStep(step, s, m.isSubmit))
		}
	}

	if m.result == api.VerificationResultSlugSuccess && m.isSubmit {
		str.WriteByte('\n')
		str.WriteByte('\n')
		str.WriteString(passStyle.Render("All tests passed! 🎉"))
		str.WriteByte('\n')
		if m.xpReward >= 0 {
			str.WriteByte('\n')
			str.WriteString(passStyle.Bold(true).Render(fmt.Sprintf("Gained +%d XP", m.xpReward)))
			str.WriteByte('\n')
			for _, item := range m.xpBreakdown {
				if item.XP == 0 {
					continue
				}
				sign := "+"
				xp := item.XP
				if xp < 0 {
					sign = "-"
					xp = -xp
				}
				if item.Percent > 0 {
					str.WriteString(secondaryStyle.Render(fmt.Sprintf("%s%3d XP (%-4s %s)", sign, xp, fmt.Sprintf("%.0f%%", item.Percent*100), item.Name)))
				} else {
					str.WriteString(secondaryStyle.Render(fmt.Sprintf("%s%3d XP (%s)", sign, xp, item.Name)))
				}
				str.WriteByte('\n')
			}
		}
		str.WriteByte('\n')
		str.WriteString(passStyle.Render("Return to your browser to continue with the next lesson."))
		str.WriteByte('\n')
		str.WriteByte('\n')
		if m.showOmitLessonIDTip {
			str.WriteString(informationStyle.Render("Tip:"))
			str.WriteString(" When you reach your next CLI lesson, you can skip copying its ID:\n  ")
			str.WriteString(informationStyle.Render("bootdev run"))
			str.WriteByte('\n')
			str.WriteByte('\n')
		}
	} else if m.result == api.VerificationResultSlugSystemError {
		str.WriteByte('\n')
		str.WriteByte('\n')
		str.WriteString(failStyle.Render("Unable to verify this lesson due to a system error."))
		if m.failure != nil && m.failure.ErrorMessage != "" {
			str.WriteString(failStyle.Render(fmt.Sprintf("\nError: %s", m.failure.ErrorMessage)))
		}
		str.WriteByte('\n')
		str.WriteString(failStyle.Render("Please try again."))
		str.WriteByte('\n')
		str.WriteByte('\n')
	} else if m.isSubmit && m.result == api.VerificationResultSlugNoop {
		str.WriteString(m.renderFailureSummary())
		str.WriteByte('\n')
		str.WriteByte('\n')
		str.WriteString(informationStyle.Render(safeStepIcon) + " " + informationStyle.Render("This was a safe step.") + "\n")
		str.WriteString("You haven't passed, but you also haven't lost armor or Sharpshooter progress.\n\n")
	} else if m.isSubmit && m.result == api.VerificationResultSlugFailure {
		str.WriteString(m.renderFailureSummary())
		str.WriteByte('\n')
		str.WriteByte('\n')
	}
	currentDate := time.Now().Format("2006-01-02")
	if m.isSubmit && m.result == api.VerificationResultSlugFailure && strings.HasSuffix(currentDate, "04-01") {
		str.WriteString(accentStyle.Render(fmt.Sprintf("This incident has been reported to your system administrator. [%s]\n", currentDate)))
	}

	return str.String()
}

func (m rootModel) renderFailureSummary() string {
	if m.failure == nil {
		return "\n" + renderStatus("FAILED") + " · Step unknown\nError: unknown"
	}
	var str strings.Builder
	index := m.failure.FailedStepIndex
	if index >= 0 && index < len(m.steps) {
		step := m.steps[index]
		fmt.Fprintf(&str, "\n%s · Step %d · %s\n%s", renderStatus("FAILED"), index+1, step.description, renderStepDetail(step.detail))
	} else {
		str.WriteString("\n" + renderStatus("FAILED") + " · Step unknown")
	}
	fmt.Fprintf(&str, "\nError: %s", m.failure.ErrorMessage)
	return str.String()
}

func renderCompactStep(step stepModel, spinner string, isSubmit bool) string {
	line := headerStyle.Bold(true).Render(step.description)
	if !step.finished {
		line = spinner + " " + line
	} else if isSubmit {
		status := "?"
		if step.passed != nil {
			status = "PASS"
			if !*step.passed {
				status = "FAIL"
			}
		}
		line = renderStatus(status) + "  " + line
	}
	if step.noPenaltyOnFail {
		line = fmt.Sprintf("%s %s", line, informationStyle.Render(safeStepIcon))
	}
	return line + "\n"
}

func renderStepResult(step stepModel, isSubmit bool) string {
	if failure := step.result.DependencyFailure; failure != nil {
		return "\n  " + warningStyle.Render("Skipped:") + " unavailable variables: " + strings.Join(failure.Names, ", ") + "\n"
	}
	var str strings.Builder
	if step.result.CLICommandResult != nil {
		exitCodeShown := false
		for index, test := range step.result.CLICommandResult.Command.Tests {
			if isSubmit && test.ExitCode != nil && test.StdoutContainsAll == nil && test.StdoutContainsNone == nil && test.StdoutLinesGT == nil && test.StdoutJq == nil && index < len(step.tests) {
				exitCodeShown = true
			}
		}
		if !exitCodeShown {
			fmt.Fprintf(&str, "\n  %s %d\n", subheaderStyle.Render("Command exit code:"), step.result.CLICommandResult.ExitCode)
		}
		if step.result.CLICommandResult.Stdout != "" {
			str.WriteString("\n  " + subheaderStyle.Render("Standard output") + "\n")
			str.WriteString(indentOutput(truncateVisualOutput(step.result.CLICommandResult.Stdout), 4))
			str.WriteByte('\n')
		}
		if step.result.CLICommandResult.Stderr != "" {
			str.WriteString("\n  " + subheaderStyle.Render("Standard error") + "\n")
			str.WriteString(indentOutput(truncateVisualOutput(step.result.CLICommandResult.Stderr), 4))
			str.WriteByte('\n')
		}
		str.WriteString(renderJqOutputs(step.result.CLICommandResult.JqOutputs))
		availableVariables, expectsVariables := availableVariablesForCLIResult(*step.result.CLICommandResult)
		if expectsVariables {
			str.WriteString(indentOutput(renderVariableSection("Variables Available", availableVariables), 2))
			str.WriteByte('\n')
		}
	}

	if step.result.HTTPRequestResult != nil {
		str.WriteByte('\n')
		str.WriteString(indentOutput(printHTTPRequestResult(*step.result.HTTPRequestResult), 2))
		str.WriteByte('\n')
	}
	return str.String()
}

func truncateVisualOutput(output string) string {
	const maxLines, maxRunes = 32, 5120
	var str strings.Builder
	str.Grow(min(len(output), maxRunes*utf8.UTFMax))
	lineCount := 1
	runeCount := 0
	offset := 0
	endsWithNewline := false

	for offset < len(output) {
		r, size := utf8.DecodeRuneInString(output[offset:])
		if r == '\n' {
			if lineCount >= maxLines {
				break
			}
			str.WriteByte('\n')
			offset += size
			lineCount++
			endsWithNewline = true
			continue
		}
		if runeCount >= maxRunes {
			break
		}

		str.WriteString(output[offset : offset+size])
		offset += size
		runeCount++
		endsWithNewline = false
	}

	if offset < len(output) {
		if str.Len() > 0 && !endsWithNewline {
			str.WriteByte('\n')
		}
		str.WriteString(informationStyle.Render("... output visually truncated"))
	}
	return str.String()
}
