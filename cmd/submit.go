package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
)

var sbumitBaseURL string

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().StringVarP(&sbumitBaseURL, "baseurl", "b", "", "set the base URL for HTTP tests, overriding any default")
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit UUID",
	Args:   cobra.MatchAll(cobra.ExactArgs(1)),
	Short:  "Submit an lesson",
	PreRun: compose(requireUpdated, requireAuth),
	Run: func(cmd *cobra.Command, args []string) {
		lessonUUID := args[0]
		lesson, err := api.FetchLesson(lessonUUID)
		cobra.CheckErr(err)
		fmt.Println(lesson.Lesson.Type)
		if lesson.Lesson.Type == "type_http_tests" {
			results, finalBaseURL := checks.HttpTest(*lesson, &sbumitBaseURL)
			printResults(results, lesson, finalBaseURL)
			cobra.CheckErr(err)
			err := api.SubmitHTTPTestLesson(lessonUUID, results)
			cobra.CheckErr(err)
			fmt.Println("\nSubmitted! Check the lesson on Boot.dev for results")
		} else {
			cobra.CheckErr("unsupported lesson type")
		}
	},
}

func printResults(results []checks.HttpTestResult, lesson *api.Lesson, finalBaseURL string) {
	fmt.Println("=====================================")
	defer fmt.Println("=====================================")
	fmt.Printf("Running requests against: %s\n", finalBaseURL)
	for i, result := range results {
		printResult(result, i, lesson)
	}
}

func printResult(result checks.HttpTestResult, i int, lesson *api.Lesson) {
	req := lesson.Lesson.LessonDataHTTPTests.HttpTests.Requests[i]
	fmt.Printf("%v. %v %v\n", i+1, req.Request.Method, req.Request.Path)
	if result.Err != "" {
		fmt.Printf("  Err: %v\n", result.Err)
	} else {
		fmt.Printf("  Response Status Code: %v\n", result.StatusCode)
		fmt.Println("  Response Headers:")
		for k, v := range req.Request.Headers {
			fmt.Printf("   - %v: %v\n", k, v)
		}
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
