package render

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

const safeStepIcon = "🛡︎"

func renderTestHeader(header string, spinner spinner.Model, isFinished bool, isSubmit bool, passed *bool, noPenaltyOnFail bool) string {
	if noPenaltyOnFail {
		header = fmt.Sprintf("%s %s", header, white.Render(safeStepIcon))
	}
	cmdStr := renderTest(header, spinner.View(), isFinished, &isSubmit, passed)
	box := borderBox.Render(fmt.Sprintf(" %s ", cmdStr))
	sliced := strings.Split(box, "\n")
	sliced[2] = strings.Replace(sliced[2], "─", "┬", 1)
	return strings.Join(sliced, "\n") + "\n"
}

func renderTests(tests []testModel, spinner string) string {
	var str strings.Builder
	var edges strings.Builder

	for _, test := range tests {
		testStr := renderTest(test.text, spinner, test.finished, nil, test.passed)
		testStr = fmt.Sprintf("  %s", testStr)
		height := lipgloss.Height(testStr)

		edges.Reset()
		edges.WriteString(" ├─")
		for i := 1; i < height; i++ {
			edges.WriteString("\n │ ")
		}

		str.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, edges.String(), testStr))
		str.WriteByte('\n')
	}

	return str.String()
}

func renderTest(text string, spinner string, isFinished bool, isSubmit *bool, passed *bool) string {
	testStr := ""
	if !isFinished {
		testStr += fmt.Sprintf("%s %s", spinner, text)
	} else if isSubmit != nil && !*isSubmit {
		testStr += text
	} else if passed == nil {
		testStr += gray.Render(fmt.Sprintf("?  %s", text))
	} else if *passed {
		testStr += green.Render(fmt.Sprintf("✓  %s", text))
	} else {
		testStr += red.Render(fmt.Sprintf("X  %s", text))
	}
	return testStr
}

func renderJqOutputs(outputs []api.CLICommandJqOutput) string {
	if len(outputs) == 0 {
		return ""
	}

	var str strings.Builder
	str.WriteString("\n > jq output:\n\n")
	for _, output := range outputs {
		str.WriteString(gray.Render(fmt.Sprintf("Query: %s", output.Query)))
		str.WriteByte('\n')
		if output.Error != "" {
			str.WriteString(gray.Render(fmt.Sprintf("Error: %s", output.Error)))
			str.WriteByte('\n')
			str.WriteByte('\n')
			continue
		}
		if len(output.Results) == 0 {
			str.WriteString(gray.Render("Results: [none]"))
			str.WriteByte('\n')
			str.WriteByte('\n')
			continue
		}
		str.WriteString(gray.Render("Results:"))
		str.WriteByte('\n')
		for _, line := range output.Results {
			str.WriteString(gray.Render("  - " + line))
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
	failedStepIndex := -1
	if m.failure != nil && m.failure.FailedStepIndex >= 0 && m.failure.FailedStepIndex < len(m.steps) {
		failedStepIndex = m.failure.FailedStepIndex
	}
	for i, step := range m.steps {
		if m.finalized && !m.verbose && failedStepIndex >= 0 && i > failedStepIndex {
			break
		}

		showAllDetails := m.verbose || (!m.isSubmit && m.finalized)
		failed := step.passed != nil && !*step.passed
		if showAllDetails {
			str.WriteString(renderTestHeader(step.description, m.spinner, step.finished, m.isSubmit, step.passed, step.noPenaltyOnFail))
			fmt.Fprintf(&str, " > %s\n", step.detail)
			str.WriteString(renderTests(step.tests, s))
		} else {
			str.WriteString(renderCompactStep(step, s, m.isSubmit))
			if failed && m.finalized {
				fmt.Fprintf(&str, "\n > %s\n", step.detail)
				str.WriteString(renderTests(step.tests, s))
			}
		}

		if step.sleepAfter != "" && step.finished && showAllDetails {
			sleepBox := borderBox.Render(fmt.Sprintf(" %s ", step.sleepAfter))
			str.WriteString(sleepBox)
			str.WriteByte('\n')
		}

		if step.result == nil || !m.finalized || (!showAllDetails && !failed) {
			continue
		}

		str.WriteString(renderStepResult(step))
	}

	if m.result == api.VerificationResultSlugSuccess && m.isSubmit {
		str.WriteByte('\n')
		str.WriteByte('\n')
		str.WriteString(green.Render("All tests passed! 🎉"))
		str.WriteByte('\n')
		if m.xpReward >= 0 {
			str.WriteByte('\n')
			str.WriteString(green.Bold(true).Render(fmt.Sprintf("Gained +%d XP", m.xpReward)))
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
					str.WriteString(gray.Render(fmt.Sprintf("%s%3d XP (%-4s %s)", sign, xp, fmt.Sprintf("%.0f%%", item.Percent*100), item.Name)))
				} else {
					str.WriteString(gray.Render(fmt.Sprintf("%s%3d XP (%s)", sign, xp, item.Name)))
				}
				str.WriteByte('\n')
			}
		}
		str.WriteByte('\n')
		str.WriteString(green.Render("Return to your browser to continue with the next lesson."))
		str.WriteByte('\n')
		str.WriteByte('\n')
	} else if m.result == api.VerificationResultSlugNoop {
		str.WriteString("\n\nTests failed! ❌")
		fmt.Fprintf(&str, "\n\nFailed Step: %v", m.failure.FailedStepIndex+1)
		str.WriteString("\nError: ")
		str.WriteString(m.failure.ErrorMessage)
		str.WriteByte('\n')
		str.WriteByte('\n')
		str.WriteString(white.Render(safeStepIcon))
		str.WriteString(" This was a safe step.\n")
		str.WriteString("You haven't passed, but you also haven't lost armor or Sharpshooter progress.\n\n")
	} else if m.result == api.VerificationResultSlugFailure {
		str.WriteByte('\n')
		str.WriteByte('\n')
		str.WriteString(red.Render("Tests failed! ❌"))
		if m.failure != nil {
			str.WriteString(red.Render(fmt.Sprintf("\n\nFailed Step: %v", m.failure.FailedStepIndex+1)))
			str.WriteString(red.Render(fmt.Sprintf("\nError: %s", m.failure.ErrorMessage)))
		} else {
			str.WriteString(red.Render("\n\nFailed Step: unknown"))
			str.WriteString(red.Render("\nError: unknown"))
		}
		str.WriteByte('\n')
		str.WriteByte('\n')
		currentDate := time.Now().Format("2006-01-02")
		if strings.HasSuffix(currentDate, "04-01") {
			str.WriteString(magenta.Render(fmt.Sprintf("This incident has been reported to your system administrator. [%s]\n", currentDate)))
		}
	}

	return str.String()
}

func renderCompactStep(step stepModel, spinner string, isSubmit bool) string {
	line := renderTest(step.description, spinner, step.finished, &isSubmit, step.passed)
	if step.noPenaltyOnFail {
		line = fmt.Sprintf("%s %s", line, white.Render(safeStepIcon))
	}
	return line + "\n"
}

func renderStepResult(step stepModel) string {
	var str strings.Builder
	if step.result.CLICommandResult != nil {
		for _, test := range step.tests {
			if strings.Contains(strings.ToLower(test.text), "exit code") {
				fmt.Fprintf(&str, "\n > Command exit code: %d\n", step.result.CLICommandResult.ExitCode)
				break
			}
		}
		str.WriteString(" > Command stdout:\n\n")
		str.WriteString(gray.Render(truncateVisualOutput(step.result.CLICommandResult.Stdout)))
		str.WriteByte('\n')
		str.WriteString(renderJqOutputs(step.result.CLICommandResult.JqOutputs))
		availableVariables, expectsVariables := availableVariablesForCLIResult(*step.result.CLICommandResult)
		if expectsVariables {
			str.WriteString(renderVariableSection("Variables Available", availableVariables))
		}
	}

	if step.result.HTTPRequestResult != nil {
		str.WriteString(printHTTPRequestResult(*step.result.HTTPRequestResult))
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
		str.WriteString("... output visually truncated")
	}
	return str.String()
}
