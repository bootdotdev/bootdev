package api

import (
	"encoding/json"
	"fmt"
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
				BaseURL             *string
				ContainsCompleteDir bool
				Requests            []struct {
					ResponseVariables []ResponseVariable
					Tests             []HTTPTest
					Request           struct {
						BasicAuth *struct {
							Username string
							Password string
						}
						Headers  map[string]string
						BodyJSON map[string]interface{}
						Method   string
						Path     string
						Actions  struct {
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
	ActualHTTPRequests any `json:"actualHTTPRequests"`
}

func SubmitHTTPTestAssignment(uuid string, results any) error {
	bytes, err := json.Marshal(submitHTTPTestRequest{ActualHTTPRequests: results})
	if err != nil {
		return err
	}
	resp, code, err := fetchWithAuthAndPayload("POST", "/v1/assignments/"+uuid+"/http_tests", bytes)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed to submit HTTP tests. code: %v: %s", code, string(resp))
	}
	return nil
}
