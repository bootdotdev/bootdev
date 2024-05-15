package render

import (
	"encoding/json"
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
)

func PrintHTTPResults(results []checks.HttpTestResult, lesson *api.Lesson, finalBaseURL string) {
	fmt.Println("=====================================")
	defer fmt.Println("=====================================")
	fmt.Printf("Running requests against: %s\n", finalBaseURL)
	for i, result := range results {
		printHTTPResult(result, i, lesson)
	}
}

func printHTTPResult(result checks.HttpTestResult, i int, lesson *api.Lesson) {
	req := lesson.Lesson.LessonDataHTTPTests.HttpTests.Requests[i]
	fmt.Printf("%v. %v %v\n", i+1, req.Request.Method, req.Request.Path)
	if result.Err != "" {
		fmt.Printf("  Err: %v\n", result.Err)
	} else {
		fmt.Println("  Request Headers:")
		for k, v := range result.RequestHeaders {
			fmt.Printf("   - %v: %v\n", k, v[0])
		}
		fmt.Printf("  Response Status Code: %v\n", result.StatusCode)
		fmt.Println("  Response Body:")
		unmarshalled := map[string]interface{}{}
		err := json.Unmarshal([]byte(result.BodyString), &unmarshalled)
		if err == nil {
			pretty, err := json.MarshalIndent(unmarshalled, "", "  ")
			if err == nil {
				fmt.Println(string(pretty))
			}
		} else {
			fmt.Println(result.BodyString)
		}
	}
}
