package render

import (
	"fmt"
	"strings"
	"unicode/utf8"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

func renderTestHeader(header string, spinner spinner.Model, isFinished bool, isSubmit bool, passed *bool) string {
	cmdStr := renderTest(header, spinner.View(), isFinished, &isSubmit, passed)
	box := borderBox.Render(fmt.Sprintf(" %s ", cmdStr))
	sliced := strings.Split(box, "\n")
	sliced[2] = strings.Replace(sliced[2], "─", "┬", 1)
	return strings.Join(sliced, "\n") + "\n"
}

func renderTestResponseVars(respVars []api.HTTPRequestResponseVariable) string {
	var str strings.Builder
	var edges strings.Builder

	for _, respVar := range respVars {
		varStr := gray.Render(fmt.Sprintf("  *  Saving `%s` from `%s`", respVar.Name, respVar.Path))
		height := lipgloss.Height(varStr)

		edges.Reset()
		edges.WriteString(" ├─")
		for i := 1; i < height; i++ {
			edges.WriteString("\n │ ")
		}

		str.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, edges.String(), varStr))
		str.WriteByte('\n')
	}

	return str.String()
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
	for _, step := range m.steps {
		str.WriteString(renderTestHeader(step.step, m.spinner, step.finished, m.isSubmit, step.passed))
		str.WriteString(renderTests(step.tests, s))
		str.WriteString(renderTestResponseVars(step.responseVariables))

		if step.sleepAfter != "" && step.finished {
			sleepBox := borderBox.Render(fmt.Sprintf(" %s ", step.sleepAfter))
			str.WriteString(sleepBox)
			str.WriteByte('\n')
		}

		if step.result == nil || !m.finalized {
			continue
		}

		if step.result.CLICommandResult != nil {
			for _, test := range step.tests {
				if strings.Contains(test.text, "exit code") {
					fmt.Fprintf(&str, "\n > Command exit code: %d\n", step.result.CLICommandResult.ExitCode)
					break
				}
			}
			str.WriteString(" > Command stdout:\n\n")
			sliced := strings.SplitSeq(step.result.CLICommandResult.Stdout, "\n")
			i := 0
			runeCount := 0
			const maxLines, maxRunes = 32, 5120
			for s := range sliced {
				if i >= maxLines || runeCount >= maxRunes {
					str.WriteString(gray.Render("... output truncated"))
					str.WriteByte('\n')
					break
				}
				runeCount += utf8.RuneCountInString(s)
				str.WriteString(gray.Render(s))
				str.WriteByte('\n')
				i++
			}
			str.WriteString(renderJqOutputs(step.result.CLICommandResult.JqOutputs))
		}

		if step.result.HTTPRequestResult != nil {
			str.WriteString(printHTTPRequestResult(*step.result.HTTPRequestResult))
		}
	}
	if m.failure != nil {
		str.WriteString("\n\n" + red.Render("Tests failed! ❌"))
		str.WriteString(red.Render(fmt.Sprintf("\n\nFailed Step: %v", m.failure.FailedStepIndex+1)))
		str.WriteString(red.Render("\nError: "+m.failure.ErrorMessage) + "\n\n")
	} else if m.success {
		str.WriteString("\n\n" + green.Render("All tests passed! 🎉") + "\n\n")
		str.WriteString(green.Render("Return to your browser to continue with the next lesson.") + "\n\n")
	}
	return str.String()
}
