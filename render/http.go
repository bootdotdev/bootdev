package render

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
)

func printHTTPRequestResult(result api.HTTPRequestResult) string {
	if result.Err != "" {
		return fmt.Sprintf("  Err: %v\n\n", result.Err)
	}

	var str strings.Builder
	fmt.Fprintf(&str, "  Response Status Code: %v\n", result.StatusCode)

	filteredHeaders := make(map[string]string)
	for respK, respV := range result.ResponseHeaders {
		for _, test := range result.Request.Tests {
			testHeader := test.HeadersEqual
			if testHeader == nil {
				testHeader = test.HeadersContain
			}
			if testHeader == nil {
				continue
			}
			interpolatedTestHeaderKey := checks.InterpolateVariables(testHeader.Key, result.Variables)
			if strings.EqualFold(respK, interpolatedTestHeaderKey) {
				filteredHeaders[respK] = respV
			}
		}
	}

	filteredTrailers := make(map[string]string)
	for respK, respV := range result.ResponseTrailers {
		for _, test := range result.Request.Tests {
			testTrailer := test.TrailersEqual
			if testTrailer == nil {
				testTrailer = test.TrailersContain
			}
			if testTrailer == nil {
				continue
			}

			interpolatedTestTrailerKey := checks.InterpolateVariables(testTrailer.Key, result.Variables)
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
		body := result.BodyString
		var unmarshalled any
		err := json.Unmarshal([]byte(result.BodyString), &unmarshalled)
		if err == nil {
			pretty, err := json.MarshalIndent(unmarshalled, "", "  ")
			if err == nil {
				body = string(pretty)
			}
		}
		str.WriteString(truncateVisualOutput(body))
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

	savedVariables, missingVariables := savedAndMissingVariablesForHTTPResult(result)
	if len(savedVariables) > 0 {
		str.WriteString(renderVariableSection("Variables Saved", savedVariables))
	}
	if len(missingVariables) > 0 {
		str.WriteString(renderVariableSection("Variables Missing", missingVariables))
	}
	availableVariables, expectsVariables := availableVariablesForHTTPResult(result)
	if expectsVariables {
		str.WriteString(renderVariableSection("Variables Available", availableVariables))
	}
	str.WriteByte('\n')

	return str.String()
}
