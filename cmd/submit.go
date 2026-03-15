package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bootdotdev/bootdev/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	forceSubmit     bool
	debugSubmission bool
)

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().BoolVar(&debugSubmission, "debug", false, "log submission request/response debug output")
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit UUID",
	Args:   cobra.MatchAll(cobra.RangeArgs(1, 10)),
	Short:  "Submit a lesson",
	PreRun: compose(requireUpdated, requireAuth),
	RunE:   submissionHandler,
}

func submissionHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	isSubmit := cmd.Name() == "submit" || forceSubmit
	lessonUUID := args[0]

	lesson, err := api.FetchLesson(lessonUUID)
	if err != nil {
		return err
	}
	if lesson.Lesson.Type != "type_cli" {
		return errors.New("unable to run lesson: unsupported lesson type")
	}
	if lesson.Lesson.LessonDataCLI == nil {
		return errors.New("unable to run lesson: missing lesson data")
	}

	data := lesson.Lesson.LessonDataCLI.CLIData

	isAllowedOS := false
	for _, system := range data.AllowedOperatingSystems {
		if system == runtime.GOOS {
			isAllowedOS = true
		}
	}

	if !isAllowedOS {
		return fmt.Errorf("lesson is not supported for your operating system (%s); try again with one of the following: %v", runtime.GOOS, data.AllowedOperatingSystems)
	}

	overrideBaseURL := viper.GetString("override_base_url")
	if overrideBaseURL != "" {
		fmt.Printf("Using overridden base_url: %v\n", overrideBaseURL)
		fmt.Printf("You can reset to the default with `bootdev config base_url --reset`\n\n")
	}

	ch := make(chan tea.Msg, 1)
	// StartRenderer and returns immediately, finalise function blocks the execution until the renderer is closed.
	finalise := render.StartRenderer(data, isSubmit, ch)

	cliResults := checks.CLIChecks(data, overrideBaseURL, ch)

	if isSubmit {
		submissionEvent, debugData, err := api.SubmitCLILesson(lessonUUID, cliResults, debugSubmission)
		if debugSubmission {
			var debugPath string
			var debugWriteErr error
			defer func() {
				reportDebugFileWrite(debugPath, debugWriteErr)
			}()
			debugPath, debugWriteErr = writeSubmissionDebugFile(lessonUUID, debugData)
		}
		if err != nil {
			return err
		}
		checks.ApplySubmissionResults(data, submissionEvent.StructuredErrCLI, ch)
		finalise(submissionEvent)
	} else {
		finalise(api.LessonSubmissionEvent{})
	}
	return nil
}

func reportDebugFileWrite(path string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write submission debug output: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Submission debug output written to %s\n", path)
}

func writeSubmissionDebugFile(lessonUUID string, data api.SubmissionDebugData) (string, error) {
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	filename := fmt.Sprintf("bootdev-submit-debug-%s-%s.txt", lessonUUID, timestamp)
	status := "unavailable"
	if data.ResponseStatusCode != 0 {
		status = fmt.Sprintf("%d", data.ResponseStatusCode)
	}

	contents := fmt.Sprintf(
		"bootdev submit debug\nTimestamp: %s\nLesson UUID: %s\nEndpoint: %s\n\n=== Request JSON ===\n%s\n\n=== Response ===\nStatus Code: %s\n%s\n",
		now.Format(time.RFC3339),
		lessonUUID,
		data.Endpoint,
		data.RequestBody,
		status,
		data.ResponseBody,
	)

	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filename, nil
	}

	return absPath, nil
}
