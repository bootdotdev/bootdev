package checks

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
)

var interpolationPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func InterpolateVariables(template string, vars map[string]string) string {
	return interpolationPattern.ReplaceAllStringFunc(template, func(m string) string {
		// Extract the key from the match, which is in the form ${key}
		key := strings.TrimSuffix(strings.TrimPrefix(m, "${"), "}")
		if val, ok := vars[key]; ok {
			return val
		}
		return m
	})
}

func InterpolationNames(template string) []string {
	matches := interpolationPattern.FindAllStringSubmatch(template, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			names = append(names, match[1])
		}
	}
	return names
}

func interpolateJSONStrings(value any, variables map[string]string) any {
	switch value := value.(type) {
	case string:
		return InterpolateVariables(value, variables)
	case []any:
		interpolated := make([]any, len(value))
		for i, item := range value {
			interpolated[i] = interpolateJSONStrings(item, variables)
		}
		return interpolated
	case map[string]any:
		interpolated := make(map[string]any, len(value))
		for key, item := range value {
			interpolated[key] = interpolateJSONStrings(item, variables)
		}
		return interpolated
	default:
		return value
	}
}

func parseStdoutVariables(stdout string, vardefs []api.CLICommandStdoutVariable, variables map[string]string) error {
	for _, vardef := range vardefs {
		if vardef.Name == "" || vardef.Regex == "" {
			return errors.New("invalid stdout variable configuration")
		}
		value, found, err := regexCapture(vardef.Regex, stdout)
		if err != nil {
			return errors.New("invalid stdout variable configuration")
		}
		if found {
			variables[vardef.Name] = value
		}
	}
	return nil
}

func parseVariables(body []byte, vardefs []api.HTTPRequestResponseVariable, variables map[string]string) error {
	bodyString := string(body)
	for _, vardef := range vardefs {
		value, found, err := responseVariableValue(vardef, bodyString)
		if err != nil {
			return err
		}
		if found {
			variables[vardef.Name] = value
		}
	}
	return nil
}

func parseHeaderVariables(headers map[string]string, vardefs []api.HTTPRequestResponseHeaderVariable, variables map[string]string) error {
	for _, vardef := range vardefs {
		value, found, err := responseHeaderVariableValue(vardef, headers)
		if err != nil {
			return err
		}
		if found {
			variables[vardef.Name] = value
		}
	}
	return nil
}

// A found value may be empty. A missing match is distinct from a parsing error.
func responseVariableValue(vardef api.HTTPRequestResponseVariable, body string) (string, bool, error) {
	if (vardef.Path == "") == (vardef.BodyRegex == "") {
		return "", false, errors.New("invalid response variable configuration")
	}
	if vardef.BodyRegex != "" {
		value, found, err := regexCapture(vardef.BodyRegex, body)
		if err != nil {
			return "", false, errors.New("invalid response body variable configuration")
		}
		return value, found, nil
	}
	values, err := valsFromJqPath(vardef.Path, body)
	if err != nil {
		return "", false, err
	}
	if len(values) != 1 || values[0] == nil {
		return "", false, nil
	}
	return fmt.Sprintf("%v", values[0]), true, nil
}

func responseHeaderVariableValue(vardef api.HTTPRequestResponseHeaderVariable, headers map[string]string) (string, bool, error) {
	value, found := findHeaderValue(headers, vardef.Header)
	if !found || vardef.Regex == "" {
		return value, found, nil
	}
	value, found, err := regexCapture(vardef.Regex, value)
	if err != nil {
		return "", false, errors.New("invalid response header variable configuration")
	}
	return value, found, nil
}

func regexCapture(pattern, input string) (string, bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", false, err
	}
	if re.NumSubexp() != 1 {
		return "", false, errors.New("capture regex requires exactly one capture group")
	}
	matches := re.FindStringSubmatch(input)
	if len(matches) != 2 {
		return "", false, nil
	}
	return matches[1], true, nil
}
