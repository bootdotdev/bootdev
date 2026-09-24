package checks

import (
	"fmt"
	"slices"

	api "github.com/bootdotdev/bootdev/client"
)

func captureNames(step api.CLIStep) []string {
	var names []string
	if step.CLICommand != nil {
		for _, capture := range step.CLICommand.StdoutVariables {
			names = append(names, capture.Name)
		}
	}
	if step.HTTPRequest != nil {
		for _, capture := range step.HTTPRequest.ResponseVariables {
			names = append(names, capture.Name)
		}
		for _, capture := range step.HTTPRequest.ResponseHeaderVariables {
			names = append(names, capture.Name)
		}
	}
	return names
}

// Missing and invalidated workflow bindings both block execution. Keeping the
// namespace separate from values prevents failed or forward captures being
// mistaken for shell variables.
func missingDependencies(step api.CLIStep, namespace map[string]bool, values map[string]string) []string {
	var names []string
	add := func(text string) {
		for _, name := range InterpolationNames(text) {
			if _, available := values[name]; namespace[name] && !available {
				names = append(names, name)
			}
		}
	}
	var visit func(any)
	visit = func(value any) {
		switch value := value.(type) {
		case string:
			add(value)
		case []any:
			for _, item := range value {
				visit(item)
			}
		case map[string]any:
			for _, item := range value {
				visit(item)
			}
		}
	}
	if step.CLICommand != nil {
		add(step.CLICommand.Command)
	}
	if step.HTTPRequest != nil {
		request := step.HTTPRequest.Request
		add(request.FullURL)
		for _, value := range request.Headers {
			add(value)
		}
		if request.BodyJSON != nil {
			visit(request.BodyJSON)
		} else {
			for _, value := range request.BodyForm {
				add(value)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// Finalize even on early execution errors so previous output values cannot leak
// into later steps. Extraction writes only to captured, never to values.
func publishCaptures(names []string, values, captured map[string]string, captureErr string) string {
	for _, name := range names {
		delete(values, name)
		if _, found := captured[name]; !found && captureErr == "" {
			captureErr = fmt.Sprintf("missing value for variable '%s'", name)
		}
	}
	if captureErr == "" {
		for _, name := range names {
			values[name] = captured[name]
		}
	}
	return captureErr
}
