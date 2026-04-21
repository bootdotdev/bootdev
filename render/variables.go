package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
)

type variableEntry struct {
	name        string
	value       string
	description string
}

func renderVariableSection(title string, entries []variableEntry) string {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].name == entries[j].name {
			return entries[i].description < entries[j].description
		}
		return entries[i].name < entries[j].name
	})

	var str strings.Builder
	fmt.Fprintf(&str, "  %s: \n", title)
	for _, entry := range entries {
		fmt.Fprintf(&str, "   - %s: %s (%s)\n", entry.name, formatVariableValue(entry.value), entry.description)
	}
	return str.String()
}

func formatVariableValue(value string) string {
	if value == "" {
		return "[not found]"
	}
	return value
}

func savedVariablesForHTTPResult(result api.HTTPRequestResult) []variableEntry {
	var entries []variableEntry
	for _, responseVariable := range result.Request.ResponseVariables {
		value := result.Variables[responseVariable.Name]
		if value == "" {
			continue
		}
		entries = append(entries, variableEntry{
			name:        responseVariable.Name,
			value:       value,
			description: "JSON Body " + responseVariable.Path,
		})
	}
	return entries
}

func missingSaveVariablesForHTTPResult(result api.HTTPRequestResult) []variableEntry {
	var entries []variableEntry
	for _, responseVariable := range result.Request.ResponseVariables {
		if result.Variables[responseVariable.Name] != "" {
			continue
		}
		entries = append(entries, variableEntry{
			name:        responseVariable.Name,
			description: "JSON Body " + responseVariable.Path,
		})
	}
	return entries
}

func availableVariablesForHTTPResult(result api.HTTPRequestResult) (entries []variableEntry, expectsVariables bool) {
	seen := map[string]bool{}

	add := func(name, description string) {
		if name == "baseURL" {
			return
		}
		expectsVariables = true
		value := result.Variables[name]
		key := name + "\x00" + description
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, variableEntry{
			name:        name,
			value:       value,
			description: description,
		})
	}

	addInterpolationNames := func(value, description string) {
		for _, name := range checks.InterpolationNames(value) {
			add(name, description)
		}
	}

	addInterpolationNames(result.Request.Request.FullURL, "Request URL")
	for key, value := range result.Request.Request.Headers {
		addInterpolationNames(value, fmt.Sprintf("Request Header %q", key))
	}
	for key, value := range result.Request.Request.BodyForm {
		addInterpolationNames(value, fmt.Sprintf("Request Form Field %q", key))
	}
	if result.Request.Request.BodyJSON != nil {
		if body, err := json.Marshal(result.Request.Request.BodyJSON); err == nil {
			addInterpolationNames(string(body), "Request JSON Body")
		}
	}
	for _, test := range result.Request.Tests {
		if test.BodyContains != nil {
			addInterpolationNames(*test.BodyContains, "Body Contains Test")
		}
		if test.BodyContainsNone != nil {
			addInterpolationNames(*test.BodyContainsNone, "Body Excludes Test")
		}
		if test.HeadersContain != nil {
			addInterpolationNames(test.HeadersContain.Key, "Header Test Key")
			addInterpolationNames(test.HeadersContain.Value, "Header Test Value")
		}
		if test.TrailersContain != nil {
			addInterpolationNames(test.TrailersContain.Key, "Trailer Test Key")
			addInterpolationNames(test.TrailersContain.Value, "Trailer Test Value")
		}
		if test.JSONValue != nil && test.JSONValue.StringValue != nil {
			addInterpolationNames(*test.JSONValue.StringValue, "JSON Value Test")
		}
	}

	return entries, expectsVariables
}

func availableVariablesForCLIResult(result api.CLICommandResult) (entries []variableEntry, expectsVariables bool) {
	seen := map[string]bool{}

	add := func(name, description string) {
		expectsVariables = true
		value := result.Variables[name]
		key := name + "\x00" + description
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, variableEntry{
			name:        name,
			value:       value,
			description: description,
		})
	}

	addInterpolationNames := func(value, description string) {
		for _, name := range checks.InterpolationNames(value) {
			add(name, description)
		}
	}

	addInterpolationNames(result.Command.Command, "Command")
	for _, test := range result.Command.Tests {
		for _, contains := range test.StdoutContainsAll {
			addInterpolationNames(contains, "Stdout Contains Test")
		}
		for _, contains := range test.StdoutContainsNone {
			addInterpolationNames(contains, "Stdout Excludes Test")
		}
		if test.StdoutJq != nil {
			addInterpolationNames(test.StdoutJq.Query, "JQ Query")
			for _, expected := range test.StdoutJq.ExpectedResults {
				if expected.Type != api.JqTypeString {
					continue
				}
				if value, ok := expected.Value.(string); ok {
					addInterpolationNames(value, "JQ Expected Value")
				}
			}
		}
	}

	return entries, expectsVariables
}
