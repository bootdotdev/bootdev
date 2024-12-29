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
	"github.com/spf13/viper"
)

type doneHttpMsg struct {
	failure *api.HTTPTestValidationError
}

type startHttpMsg struct {
	url               string
	method            string
	responseVariables []api.ResponseVariable
}

type resolveHttpMsg struct {
	index   int
	passed  *bool
	results *checks.HttpTestResult
}
type httpReqModel struct {
	responseVariables []api.ResponseVariable
	request           string
	passed            *bool
	results           *checks.HttpTestResult
	finished          bool
	tests             []testModel
}

type httpRootModel struct {
	reqs      []httpReqModel
	spinner   spinner.Model
	failure   *api.HTTPTestValidationError
	isSubmit  bool
	success   bool
	finalized bool
	clear     bool
}

func initialModelHTTP(isSubmit bool) httpRootModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return httpRootModel{
		spinner:  s,
		isSubmit: isSubmit,
		reqs:     []httpReqModel{},
	}
}

func (m httpRootModel) Init() tea.Cmd {
	green = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.green")))
	red = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.red")))
	gray = lipgloss.NewStyle().Foreground(lipgloss.Color(viper.GetString("color.gray")))
	return m.spinner.Tick
}

func (m httpRootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneHttpMsg:
		m.failure = msg.failure
		if m.failure == nil && m.isSubmit {
			m.success = true
		}
		m.clear = true
		return m, tea.Quit

	case startHttpMsg:
		m.reqs = append(m.reqs, httpReqModel{
			request:           fmt.Sprintf("%s %s", msg.method, msg.url),
			tests:             []testModel{},
			responseVariables: msg.responseVariables,
		})
		return m, nil

	case resolveHttpMsg:
		m.reqs[msg.index].passed = msg.passed
		m.reqs[msg.index].finished = true
		m.reqs[msg.index].results = msg.results
		return m, nil

	case startTestMsg:
		m.reqs[len(m.reqs)-1].tests = append(
			m.reqs[len(m.reqs)-1].tests,
			testModel{text: msg.text},
		)
		return m, nil

	case resolveTestMsg:
		m.reqs[len(m.reqs)-1].tests[msg.index].passed = msg.passed
		m.reqs[len(m.reqs)-1].tests[msg.index].finished = true
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m httpRootModel) View() string {
	if m.clear {
		return ""
	}
	s := m.spinner.View()
	var str string
	for _, req := range m.reqs {
		str += renderTestHeader(req.request, m.spinner, req.finished, m.isSubmit, req.passed)
		str += renderTests(req.tests, s)
		str += renderTestResponseVars(req.responseVariables)
		if req.results != nil && m.finalized {
			str += printHTTPResult(*req.results)
		}
	}
	if m.failure != nil {
		str += red.Render("\n\nError: "+*m.failure.ErrorMessage) + "\n\n"
	} else if m.success {
		str += "\n\n" + green.Render("All tests passed! 🎉") + "\n\n"
		str += green.Render("Return to your browser to continue with the next lesson.") + "\n\n"
	}
	return str
}

func printHTTPResult(result checks.HttpTestResult) string {
	if result.Err != "" {
		return fmt.Sprintf("  Err: %v\n\n", result.Err)
	}

	str := ""

	str += fmt.Sprintf("  Response Status Code: %v\n", result.StatusCode)

	filteredHeaders := make(map[string]string)
	for respK, respV := range result.ResponseHeaders {
		for reqK := range result.Request.Request.Headers {
			if strings.ToLower(respK) == strings.ToLower(reqK) {
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

func HTTPRun(
	data api.LessonDataHTTPTests,
	results []checks.HttpTestResult,
) {
	httpRenderer(data, results, nil, false)
}

func HTTPSubmission(
	data api.LessonDataHTTPTests,
	results []checks.HttpTestResult,
	failure *api.HTTPTestValidationError,
) {
	httpRenderer(data, results, failure, true)
}

func httpRenderer(
	data api.LessonDataHTTPTests,
	results []checks.HttpTestResult,
	failure *api.HTTPTestValidationError,
	isSubmit bool,
) {
	var wg sync.WaitGroup
	ch := make(chan tea.Msg, 1)
	p := tea.NewProgram(initialModelHTTP(isSubmit), tea.WithoutSignalHandler())
	wg.Add(1)
	go func() {
		defer wg.Done()
		if model, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if r, ok := model.(httpRootModel); ok {
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

		for i, req := range data.HttpTests.Requests {
			url := req.Request.Path
			if req.Request.FullURL != "" {
				url = req.Request.FullURL
			}
			ch <- startHttpMsg{
				url:               checks.InterpolateVariables(url, results[i].Variables),
				method:            req.Request.Method,
				responseVariables: req.ResponseVariables,
			}
			for _, test := range req.Tests {
				ch <- startTestMsg{
					text: prettyPrintHTTPTest(test, results[i].Variables),
				}
			}
			time.Sleep(500 * time.Millisecond)
			for j := range req.Tests {
				if !isSubmit {
					ch <- resolveTestMsg{index: j}
				} else if failure != nil && (*failure.FailedRequestIndex < i || (*failure.FailedRequestIndex == i && *failure.FailedTestIndex < j)) {
					ch <- resolveTestMsg{index: j}
				} else {
					time.Sleep(350 * time.Millisecond)
					ch <- resolveTestMsg{index: j, passed: pointerToBool(failure == nil || !(*failure.FailedRequestIndex == i && *failure.FailedTestIndex == j))}
				}
			}
			if !isSubmit {
				ch <- resolveHttpMsg{index: i, results: &results[i]}
			} else if failure != nil && *failure.FailedRequestIndex < i {
				ch <- resolveHttpMsg{index: i}
			} else {
				passed := failure == nil || *failure.FailedRequestIndex != i
				if passed {
					ch <- resolveHttpMsg{index: i, passed: pointerToBool(passed)}
				} else {
					ch <- resolveHttpMsg{index: i, passed: pointerToBool(passed), results: &results[i]}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)

		ch <- doneHttpMsg{failure: failure}
	}()
	wg.Wait()
}

func prettyPrintHTTPTest(test api.HTTPTest, variables map[string]string) string {
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
