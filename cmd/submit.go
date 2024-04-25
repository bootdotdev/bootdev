package cmd

import (
	"errors"
	"fmt"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/spf13/cobra"
)

var baseURL string

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().StringVarP(&baseURL, "baseurl", "b", "", "set the base URL for HTTP tests, overriding any default")
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit UUID",
	Args:   cobra.MatchAll(cobra.ExactArgs(1)),
	Short:  "Submit an assignment",
	PreRun: requireAuth,
	Run: func(cmd *cobra.Command, args []string) {
		assignmentUUID := args[0]
		assignment, err := api.FetchAssignment(assignmentUUID)
		cobra.CheckErr(err)
		if assignment.Assignment.Type == "type_http_tests" {
			results := checks.HttpTest(*assignment, &baseURL)
			fmt.Println("=====================================")
			defer fmt.Println("=====================================")
			fmt.Println("Running requests:")
			for i, result := range results {
				req := assignment.Assignment.AssignmentDataHTTPTests.HttpTests.Requests[i]
				fmt.Printf("%v. %v %v", i+1, req.Request.Method, req.Request.Path)
				if result.Err != "" {
					fmt.Printf(" - Err %v\n", result.Err)
				} else {
					fmt.Printf(" - Status Code: %v\n", result.StatusCode)
					fmt.Println(" - Response Headers:")
					for k, v := range req.Request.Headers {
						fmt.Printf("   - %v: %v\n", k, v)
					}
					fmt.Println(" - Response Body:")
					fmt.Println(result.BodyString)
				}
			}
			cobra.CheckErr(err)
			err := api.SubmitHTTPTestAssignment(assignmentUUID, results)
			cobra.CheckErr(err)
			fmt.Println("\nSubmitted! Check the lesson on Boot.dev for results")
		} else {
			cobra.CheckErr(errors.New("unsupported assignment type"))
		}
	},
}
