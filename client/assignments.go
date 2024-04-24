package api

import (
	"encoding/json"
	"errors"
)

type ResponseVariable struct {
	Name string
	Path string
}

// Only one of these fields should be set
type HTTPTest struct {
	StatusCode     *int
	BodyContains   *string
	HeadersContain *HTTPTestHeader
	JSONValue      *HTTPTestJSONValue
}

type OperatorType string

const (
	OpEquals      OperatorType = "eq"
	OpGreaterThan OperatorType = "gt"
)

type HTTPTestJSONValue struct {
	Path        string
	Operator    OperatorType
	IntValue    *int
	StringValue *string
	BoolValue   *bool
}

type HTTPTestHeader struct {
	Key   string
	Value string
}

type Assignment struct {
	Assignment struct {
		Type                    string
		AssignmentDataHTTPTests *struct {
			HttpTests struct {
				ContainsCompleteDir bool
				Requests            []struct {
					ResponseVariables []ResponseVariable
					Tests             []HTTPTest
					Request           struct {
						Method  string
						Path    string
						Actions struct {
							DelayRequestByMs *int32
						}
					}
				}
			}
		}
	}
}

func FetchAssignment(uuid string) (*Assignment, error) {
	resp, err := fetchWithAuth("GET", "/v1/assignments/"+uuid)
	if err != nil {
		return nil, err
	}

	var data Assignment
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

type HTTPTestValidationError struct {
	ErrorMessage       *string `json:"Error"`
	FailedRequestIndex *int    `json:"FailedRequestIndex"`
	FailedTestIndex    *int    `json:"FailedTestIndex"`
}

type submitHTTPTestRequest struct {
	ActualHTTPRequests []any `json:"actualHTTPRequests"`
}

func SubmitHTTPTestAssignment(uuid string, results []any) (*HTTPTestValidationError, error) {
	bytes, err := json.Marshal(submitHTTPTestRequest{ActualHTTPRequests: results})
	if err != nil {
		return nil, err
	}
	resp, code, err := fetchWithAuthAndPayload("POST", "/v1/assignments/"+uuid+"/http_tests", bytes)

	if err != nil {
		return nil, err
	}
	if code >= 500 {
		return nil, errors.New("Internal server error")
	}

	if code == 200 {
		return nil, nil
	}

	var data HTTPTestValidationError
	err = json.Unmarshal(resp, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}
