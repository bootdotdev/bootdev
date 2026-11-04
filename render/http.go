package render

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/goccy/go-json"
)

func httpErrorHeading(category api.HTTPErrorCategory) string {
	switch category {
	case api.HTTPErrorRequest:
		return "Request error"
	case api.HTTPErrorResponse:
		return "Response error"
	default:
		return "HTTP error"
	}
}

func printHTTPRequestResult(result api.HTTPRequestResult) string {
	if result.Err != "" {
		return fmt.Sprintf("%s\n%s\n", warningStyle.Render(httpErrorHeading(result.ErrorCategory)), indentOutput(result.Err, 2))
	}

	var str strings.Builder
	fmt.Fprintf(&str, "%s %v\n", subheaderStyle.Render("Response status:"), result.StatusCode)

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
		str.WriteString("\n" + subheaderStyle.Render("Response headers") + "\n")
		for k, v := range filteredHeaders {
			fmt.Fprintf(&str, "   - %v: %v\n", k, v)
		}
	}

	str.WriteString("\n" + subheaderStyle.Render("Response body") + "\n")
	if result.BodyString == "" {
		str.WriteString("  " + secondaryStyle.Render("<empty>") + "\n")
	} else {
		var pretty bytes.Buffer
		bytes := []byte(result.BodyString)
		contentType := http.DetectContentType(bytes)
		if contentType == "application/json" || strings.HasPrefix(contentType, "text/") {
			body := result.BodyString
			if err := json.Indent(&pretty, bytes, "", "  "); err == nil {
				body = pretty.String()
			}
			str.WriteString(indentOutput(truncateVisualOutput(body), 2))
		} else {
			str.WriteString(informationStyle.Render(fmt.Sprintf(
				"Binary %s file. Raw data hidden. To manually debug, use curl -o myfile.bin and inspect the file",
				contentType,
			)))
		}
		str.WriteByte('\n')
	}

	if len(filteredTrailers) > 0 {
		str.WriteString("\n" + subheaderStyle.Render("Response trailers") + "\n")
		for k, v := range filteredTrailers {
			fmt.Fprintf(&str, "   - %v: %v\n", k, v)
		}
	}

	savedVariables, missingVariables := savedAndMissingVariablesForHTTPResult(result)
	if len(savedVariables) > 0 {
		str.WriteString(renderVariableSection("Variables Saved", savedVariables, subheaderStyle))
	}
	if len(missingVariables) > 0 {
		str.WriteString(renderVariableSection("Variables Missing", missingVariables, warningStyle))
	}
	availableVariables, expectsVariables := availableVariablesForHTTPResult(result)
	if expectsVariables {
		str.WriteString(renderVariableSection("Variables Available", availableVariables, subheaderStyle))
	}
	str.WriteByte('\n')

	return str.String()
}
