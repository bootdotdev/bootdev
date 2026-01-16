package render

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/messages"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-json"
	"github.com/muesli/termenv"
	"github.com/spf13/viper"
)

var (
	green     lipgloss.Style
	red       lipgloss.Style
	gray      lipgloss.Style
	borderBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
)

type testModel struct {
	text     string
	passed   *bool
	finished bool
}

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

		edges.Reset() // Reset and reuse edges builder
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

		edges.Reset() // Reset and reuse edges builder
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

type stepModel struct {
	responseVariables []api.HTTPRequestResponseVariable
	step              string
	passed            *bool
	result            *api.CLIStepResult
	finished          bool
	tests             []testModel
	sleepAfter        string
}

type rootModel struct {
	steps     []stepModel
	spinner   spinner.Model
	failure   *api.VerificationResultStructuredErrCLI
	isSubmit  bool
	success   bool
	finalized bool
	clear     bool
}

func initModel(isSubmit bool) rootModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return rootModel{
		spinner:  s,
		isSubmit: isSubmit,
		steps:    []stepModel{},
	}
}

func (m rootModel) Init() tea.Cmd {
	green = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.green")))
	red = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.red")))
	gray = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.gray")))
	return m.spinner.Tick
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.DoneStepMsg:
		m.failure = msg.Failure
		if m.failure == nil && m.isSubmit {
			m.success = true
		}
		m.clear = true
		return m, tea.Quit

	case messages.StartStepMsg:
		step := fmt.Sprintf("Running: %s", msg.CMD)
		if msg.CMD == "" {
			step = fmt.Sprintf("%s %s", msg.Method, msg.URL)
		}
		m.steps = append(m.steps, stepModel{
			step:              step,
			tests:             []testModel{},
			responseVariables: msg.ResponseVariables,
		})
		return m, nil

	case messages.SleepMsg:
		if len(m.steps) > 0 {
			lastStepIdx := len(m.steps) - 1
			durationSec := float64(msg.DurationMs) / 1000.0
			sleepText := ""
			if durationSec >= 1.0 {
				sleepText = fmt.Sprintf("Waiting %.1fs...", durationSec)
			} else {
				sleepText = fmt.Sprintf("Waiting %dms...", msg.DurationMs)
			}
			m.steps[lastStepIdx].sleepAfter = sleepText
		}
		return m, nil

	case messages.ResolveStepMsg:
		m.steps[msg.Index].passed = msg.Passed
		m.steps[msg.Index].finished = true
		if msg.Result != nil {
			m.steps[msg.Index].result = msg.Result
		}
		return m, nil

	case messages.StartTestMsg:
		m.steps[len(m.steps)-1].tests = append(
			m.steps[len(m.steps)-1].tests,
			testModel{text: msg.Text},
		)
		return m, nil

	case messages.ResolveTestMsg:
		m.steps[msg.StepIndex].tests[msg.TestIndex].passed = msg.Passed
		m.steps[msg.StepIndex].tests[msg.TestIndex].finished = true
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
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
			// render the results
			for _, test := range step.tests {
				// for clarity, only show exit code if it's tested
				if strings.Contains(test.text, "exit code") {
					fmt.Fprintf(&str, "\n > Command exit code: %d\n", step.result.CLICommandResult.ExitCode)
					break
				}
			}
			str.WriteString(" > Command stdout:\n\n")
			sliced := strings.SplitSeq(step.result.CLICommandResult.Stdout, "\n")
			for s := range sliced {
				str.WriteString(gray.Render(s))
				str.WriteByte('\n')
			}
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

func printHTTPRequestResult(result api.HTTPRequestResult) string {
	if result.Err != "" {
		return fmt.Sprintf("  Err: %v\n\n", result.Err)
	}

	var str strings.Builder
	fmt.Fprintf(&str, "  Response Status Code: %v\n", result.StatusCode)

	filteredHeaders := make(map[string]string)
	for respK, respV := range result.ResponseHeaders {
		for _, test := range result.Request.Tests {
			if test.HeadersContain == nil {
				continue
			}
			interpolatedTestHeaderKey := checks.InterpolateVariables(test.HeadersContain.Key, result.Variables)
			if strings.EqualFold(respK, interpolatedTestHeaderKey) {
				filteredHeaders[respK] = respV
			}
		}
	}

	filteredTrailers := make(map[string]string)
	for respK, respV := range result.ResponseTrailers {
		for _, test := range result.Request.Tests {
			if test.TrailersContain == nil {
				continue
			}

			interpolatedTestTrailerKey := checks.InterpolateVariables(test.TrailersContain.Key, result.Variables)
			if strings.EqualFold(respK, interpolatedTestTrailerKey) {
				filteredTrailers[respK] = respV
			}
		}
	}

	if len(filteredHeaders) > 0 {
		str.WriteString("  Response Headers: \n")
		for k, v := range filteredHeaders {
			fmt.Fprintf(&str, "   - %v: %v\n", k, v)
		}
	}

	str.WriteString("  Response Body: \n")
	bytes := []byte(result.BodyString)
	contentType := http.DetectContentType(bytes)
	if contentType == "application/json" || strings.HasPrefix(contentType, "text/") {
		var unmarshalled any
		err := json.Unmarshal([]byte(result.BodyString), &unmarshalled)
		if err == nil {
			pretty, err := json.MarshalIndent(unmarshalled, "", "  ")
			if err == nil {
				str.Write(pretty)
			} else {
				str.WriteString(result.BodyString)
			}
		} else {
			str.WriteString(result.BodyString)
		}
	} else {
		fmt.Fprintf(
			&str,
			"Binary %s file. Raw data hidden. To manually debug, use curl -o myfile.bin and inspect the file",
			contentType,
		)
	}
	str.WriteByte('\n')

	if len(filteredTrailers) > 0 {
		str.WriteString("  Response Trailers: \n")
		for k, v := range filteredTrailers {
			fmt.Fprintf(&str, "   - %v: %v\n", k, v)
		}
	}

	if len(result.Variables) > 0 {
		str.WriteString("  Variables available: \n")
		for k, v := range result.Variables {
			if v != "" {
				fmt.Fprintf(&str, "   - %v: %v\n", k, v)
			} else {
				fmt.Fprintf(&str, "   - %v: [not found]\n", k)
			}
		}
	}
	str.WriteByte('\n')

	return str.String()
}

func StartRenderer(data api.CLIData, isSubmit bool, ch chan tea.Msg) func(*api.VerificationResultStructuredErrCLI) {
	var wg sync.WaitGroup
	p := tea.NewProgram(initModel(isSubmit), tea.WithoutSignalHandler())

	wg.Add(1)
	go func() {
		defer wg.Done()
		if model, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if r, ok := model.(rootModel); ok {
			r.clear = false
			r.finalized = true
			output := termenv.NewOutput(os.Stdout)
			output.WriteString(r.View())
		}
	}()

	go func() {
		for {
			msg := <-ch
			p.Send(msg)
		}
	}()

	return func(failure *api.VerificationResultStructuredErrCLI) {
		ch <- messages.DoneStepMsg{Failure: failure}
		wg.Wait()
	}
}
