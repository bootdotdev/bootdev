package render

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var green lipgloss.Style
var red lipgloss.Style
var gray lipgloss.Style
var borderBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

type testModel struct {
	text     string
	passed   *bool
	finished bool
}

type startTestMsg struct {
	text string
}

type resolveTestMsg struct {
	index  int
	passed *bool
}

func renderTestHeader(header string, spinner spinner.Model, isFinished bool, isSubmit bool, passed *bool) string {
	cmdStr := renderTest(header, spinner.View(), isFinished, &isSubmit, passed)
	box := borderBox.Render(fmt.Sprintf(" %s ", cmdStr))
	sliced := strings.Split(box, "\n")
	sliced[2] = strings.Replace(sliced[2], "─", "┬", 1)
	return strings.Join(sliced, "\n") + "\n"
}

func renderTestResponseVars(respVars []api.HTTPRequestResponseVariable) string {
	var str string
	for _, respVar := range respVars {
		varStr := gray.Render(fmt.Sprintf("  *  Saving `%s` from `%s`", respVar.Name, respVar.Path))
		edges := " ├─"
		for i := 0; i < lipgloss.Height(varStr)-1; i++ {
			edges += "\n │ "
		}
		str += lipgloss.JoinHorizontal(lipgloss.Top, edges, varStr)
		str += "\n"
	}
	return str
}

func renderTests(tests []testModel, spinner string) string {
	var str string
	for _, test := range tests {
		testStr := renderTest(test.text, spinner, test.finished, nil, test.passed)
		testStr = fmt.Sprintf("  %s", testStr)

		edges := " ├─"
		for i := 0; i < lipgloss.Height(testStr)-1; i++ {
			edges += "\n │ "
		}
		str += lipgloss.JoinHorizontal(lipgloss.Top, edges, testStr)
		str += "\n"
	}
	return str
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

type doneStepMsg struct {
	failure *api.StructuredErrCLI
}

type startStepMsg struct {
	responseVariables []api.HTTPRequestResponseVariable
	cmd               string
	url               string
	method            string
}

type resolveStepMsg struct {
	index  int
	passed *bool
	result *api.CLIStepResult
}

type stepModel struct {
	responseVariables []api.HTTPRequestResponseVariable
	step              string
	passed            *bool
	result            *api.CLIStepResult
	finished          bool
	tests             []testModel
}

type rootModel struct {
	steps     []stepModel
	spinner   spinner.Model
	failure   *api.StructuredErrCLI
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
	case doneStepMsg:
		m.failure = msg.failure
		if m.failure == nil && m.isSubmit {
			m.success = true
		}
		m.clear = true
		return m, tea.Quit

	case startStepMsg:
		step := fmt.Sprintf("Running: %s", msg.cmd)
		if msg.cmd == "" {
			step = fmt.Sprintf("%s %s", msg.method, msg.url)
		}
		m.steps = append(m.steps, stepModel{
			step:              step,
			tests:             []testModel{},
			responseVariables: msg.responseVariables,
		})
		return m, nil

	case resolveStepMsg:
		m.steps[msg.index].passed = msg.passed
		m.steps[msg.index].finished = true
		m.steps[msg.index].result = msg.result
		return m, nil

	case startTestMsg:
		m.steps[len(m.steps)-1].tests = append(
			m.steps[len(m.steps)-1].tests,
			testModel{text: msg.text},
		)
		return m, nil

	case resolveTestMsg:
		m.steps[len(m.steps)-1].tests[msg.index].passed = msg.passed
		m.steps[len(m.steps)-1].tests[msg.index].finished = true
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
	var str string
	for _, step := range m.steps {
		str += renderTestHeader(step.step, m.spinner, step.finished, m.isSubmit, step.passed)
		str += renderTests(step.tests, s)
		str += renderTestResponseVars(step.responseVariables)
		if step.result == nil || !m.finalized {
			continue
		}

		if step.result.CLICommandResult != nil {
			// render the results
			for _, test := range step.tests {
				// for clarity, only show exit code if it's tested
				if strings.Contains(test.text, "exit code") {
					str += fmt.Sprintf("\n > Command exit code: %d\n", step.result.CLICommandResult.ExitCode)
					break
				}
			}
			str += " > Command stdout:\n\n"
			sliced := strings.Split(step.result.CLICommandResult.Stdout, "\n")
			for _, s := range sliced {
				str += gray.Render(s) + "\n"
			}

		}

		if step.result.HTTPRequestResult != nil {
			str += printHTTPRequestResult(*step.result.HTTPRequestResult)
		}
	}
	if m.failure != nil {
		str += red.Render("\n\nError: "+m.failure.ErrorMessage) + "\n\n"
	} else if m.success {
		str += "\n\n" + green.Render("All tests passed! 🎉") + "\n\n"
		str += green.Render("Return to your browser to continue with the next lesson.") + "\n\n"
	}
	return str
}

func prettyPrintCLICommand(test api.CLICommandTest, variables map[string]string) string {
	if test.ExitCode != nil {
		return fmt.Sprintf("Expect exit code %d", *test.ExitCode)
	}
	if test.StdoutLinesGt != nil {
		return fmt.Sprintf("Expect > %d lines on stdout", *test.StdoutLinesGt)
	}
	if test.StdoutContainsAll != nil {
		str := "Expect stdout to contain all of:"
		for _, contains := range test.StdoutContainsAll {
			interpolatedContains := checks.InterpolateVariables(contains, variables)
			str += fmt.Sprintf("\n      - '%s'", interpolatedContains)
		}
		return str
	}
	if test.StdoutContainsNone != nil {
		str := "Expect stdout to contain none of:"
		for _, containsNone := range test.StdoutContainsNone {
			interpolatedContainsNone := checks.InterpolateVariables(containsNone, variables)
			str += fmt.Sprintf("\n      - '%s'", interpolatedContainsNone)
		}
		return str
	}
	return ""
}

func pointerToBool(a bool) *bool {
	return &a
}

func printHTTPRequestResult(result api.HTTPRequestResult) string {
	if result.Err != "" {
		return fmt.Sprintf("  Err: %v\n\n", result.Err)
	}

	str := ""

	str += fmt.Sprintf("  Response Status Code: %v\n", result.StatusCode)

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

	if len(filteredHeaders) > 0 {
		str += "  Response Headers: \n"
		for k, v := range filteredHeaders {
			str += fmt.Sprintf("   - %v: %v\n", k, v)
		}
	}

	str += "  Response Body: \n"
	bytes := []byte(result.BodyString)
	contentType := http.DetectContentType(bytes)
	if contentType == "application/json" || strings.HasPrefix(contentType, "text/") {
		var unmarshalled interface{}
		err := json.Unmarshal([]byte(result.BodyString), &unmarshalled)
		if err == nil {
			pretty, err := json.MarshalIndent(unmarshalled, "", "  ")
			if err == nil {
				str += string(pretty)
			} else {
				str += result.BodyString
			}
		} else {
			str += result.BodyString
		}
	} else {
		str += fmt.Sprintf("Binary %s file", contentType)
	}
	str += "\n"

	if len(result.Variables) > 0 {
		str += "  Variables available: \n"
		for k, v := range result.Variables {
			if v != "" {
				str += fmt.Sprintf("   - %v: %v\n", k, v)
			} else {
				str += fmt.Sprintf("   - %v: [not found]\n", k)
			}
		}
	}
	str += "\n"

	return str
}

func RenderRun(
	data api.CLIData,
	results []api.CLIStepResult,
) {
	renderer(data, results, nil, false)
}

func RenderSubmission(
	data api.CLIData,
	results []api.CLIStepResult,
	failure *api.StructuredErrCLI,
) {
	renderer(data, results, failure, true)
}

func renderer(
	data api.CLIData,
	results []api.CLIStepResult,
	failure *api.StructuredErrCLI,
	isSubmit bool,
) {
	var wg sync.WaitGroup
	ch := make(chan tea.Msg, 1)
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
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i, step := range data.Steps {
			switch {
			case step.CLICommand != nil && results[i].CLICommandResult != nil:
				renderCLICommand(*step.CLICommand, *results[i].CLICommandResult, failure, isSubmit, ch, i)
			case step.HTTPRequest != nil && results[i].HTTPRequestResult != nil:
				renderHTTPRequest(*step.HTTPRequest, *results[i].HTTPRequestResult, failure, isSubmit, data.BaseURL, ch, i)
			default:
				cobra.CheckErr("unable to run lesson: missing results")
			}
		}
		time.Sleep(500 * time.Millisecond)

		ch <- doneStepMsg{failure: failure}
	}()
	wg.Wait()
}

func renderCLICommand(
	cmd api.CLIStepCLICommand,
	result api.CLICommandResult,
	failure *api.StructuredErrCLI,
	isSubmit bool,
	ch chan tea.Msg,
	index int,
) {
	ch <- startStepMsg{cmd: result.FinalCommand}

	for _, test := range cmd.Tests {
		ch <- startTestMsg{text: prettyPrintCLICommand(test, result.Variables)}
	}

	time.Sleep(500 * time.Millisecond)

	earlierCmdFailed := false
	if failure != nil {
		earlierCmdFailed = failure.FailedStepIndex < index
	}
	for j := range cmd.Tests {
		earlierTestFailed := false
		if failure != nil {
			if earlierCmdFailed {
				earlierTestFailed = true
			} else if failure.FailedStepIndex == index {
				earlierTestFailed = failure.FailedTestIndex < j
			}
		}
		if !isSubmit {
			ch <- resolveTestMsg{index: j}
		} else if earlierTestFailed {
			ch <- resolveTestMsg{index: j}
		} else {
			time.Sleep(350 * time.Millisecond)
			passed := failure == nil || failure.FailedStepIndex != index || failure.FailedTestIndex != j
			ch <- resolveTestMsg{
				index:  j,
				passed: pointerToBool(passed),
			}
		}
	}

	if !isSubmit {
		ch <- resolveStepMsg{
			index: index,
			result: &api.CLIStepResult{
				CLICommandResult: &result,
			},
		}
	} else if earlierCmdFailed {
		ch <- resolveStepMsg{index: index}
	} else {
		passed := failure == nil || failure.FailedStepIndex != index
		if passed {
			ch <- resolveStepMsg{
				index:  index,
				passed: pointerToBool(passed),
			}
		} else {
			ch <- resolveStepMsg{
				index:  index,
				passed: pointerToBool(passed),
				result: &api.CLIStepResult{
					CLICommandResult: &result,
				},
			}
		}
	}
}

func renderHTTPRequest(
	req api.CLIStepHTTPRequest,
	result api.HTTPRequestResult,
	failure *api.StructuredErrCLI,
	isSubmit bool,
	baseURL *string,
	ch chan tea.Msg,
	index int,
) {
	url := ""
	overrideBaseURL := viper.GetString("override_base_url")
	if overrideBaseURL != "" {
		url = overrideBaseURL
	} else if baseURL != nil && *baseURL != "" {
		url = *baseURL
	}
	if req.Request.FullURL != "" {
		url = req.Request.FullURL
	}
	url += req.Request.Path
	ch <- startStepMsg{
		url:               checks.InterpolateVariables(url, result.Variables),
		method:            req.Request.Method,
		responseVariables: req.ResponseVariables,
	}

	for _, test := range req.Tests {
		ch <- startTestMsg{text: prettyPrintHTTPTest(test, result.Variables)}
	}

	time.Sleep(500 * time.Millisecond)

	for j := range req.Tests {
		if !isSubmit {
			ch <- resolveTestMsg{index: j}
		} else if failure != nil && (failure.FailedStepIndex < index || (failure.FailedStepIndex == index && failure.FailedTestIndex < j)) {
			ch <- resolveTestMsg{index: j}
		} else {
			time.Sleep(350 * time.Millisecond)
			ch <- resolveTestMsg{index: j, passed: pointerToBool(failure == nil || !(failure.FailedStepIndex == index && failure.FailedTestIndex == j))}
		}
	}

	if !isSubmit {
		ch <- resolveStepMsg{
			index: index,
			result: &api.CLIStepResult{
				HTTPRequestResult: &result,
			},
		}
	} else if failure != nil && failure.FailedStepIndex < index {
		ch <- resolveStepMsg{index: index}
	} else {
		passed := failure == nil || failure.FailedStepIndex != index
		if passed {
			ch <- resolveStepMsg{
				index:  index,
				passed: pointerToBool(passed),
			}
		} else {
			ch <- resolveStepMsg{
				index:  index,
				passed: pointerToBool(passed),
				result: &api.CLIStepResult{
					HTTPRequestResult: &result,
				},
			}
		}
	}
}

func prettyPrintHTTPTest(test api.HTTPRequestTest, variables map[string]string) string {
	if test.StatusCode != nil {
		return fmt.Sprintf("Expecting status code: %d", *test.StatusCode)
	}
	if test.BodyContains != nil {
		interpolated := checks.InterpolateVariables(*test.BodyContains, variables)
		return fmt.Sprintf("Expecting body to contain: %s", interpolated)
	}
	if test.BodyContainsNone != nil {
		interpolated := checks.InterpolateVariables(*test.BodyContainsNone, variables)
		return fmt.Sprintf("Expecting JSON body to not contain: %s", interpolated)
	}
	if test.HeadersContain != nil {
		interpolatedKey := checks.InterpolateVariables(test.HeadersContain.Key, variables)
		interpolatedValue := checks.InterpolateVariables(test.HeadersContain.Value, variables)
		return fmt.Sprintf("Expecting headers to contain: '%s: %v'", interpolatedKey, interpolatedValue)
	}
	if test.JSONValue != nil {
		var val any
		var op any
		if test.JSONValue.IntValue != nil {
			val = *test.JSONValue.IntValue
		} else if test.JSONValue.StringValue != nil {
			val = *test.JSONValue.StringValue
		} else if test.JSONValue.BoolValue != nil {
			val = *test.JSONValue.BoolValue
		}
		if test.JSONValue.Operator == api.OpEquals {
			op = "to be equal to"
		} else if test.JSONValue.Operator == api.OpGreaterThan {
			op = "to be greater than"
		} else if test.JSONValue.Operator == api.OpContains {
			op = "contains"
		} else if test.JSONValue.Operator == api.OpNotContains {
			op = "to not contain"
		}
		expecting := fmt.Sprintf("Expecting JSON at %v %s %v", test.JSONValue.Path, op, val)
		return checks.InterpolateVariables(expecting, variables)
	}
	return ""
}
