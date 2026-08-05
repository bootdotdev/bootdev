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
	verboseOutput   bool
)

func init() {
	rootCmd.AddCommand(submitCmd)
	submitCmd.Flags().BoolVar(&debugSubmission, "debug", false, "log submission request/response debug output")
	submitCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "show detailed final output for every step")
}

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:    "submit [UUID]",
	Args:   cobra.MaximumNArgs(1),
	Short:  "Submit a lesson. Submits your next lesson when no UUID is given",
	PreRun: requireUpdatedAndAuth,
	RunE:   submissionHandler,
}

func submissionHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	isSubmit := cmd.Name() == "submit" || forceSubmit

	var lesson *api.Lesson
	var lessonUUID string
	if len(args) > 0 {
		lessonUUID = args[0]
		fetchedLesson, err := api.FetchLesson(lessonUUID)
		if err != nil {
			return err
		}
		action := "Running"
		if isSubmit {
			action = "Submitting"
		}
		fmt.Printf(
			"%s lesson:\n%d.%d - %s\n",
			action,
			fetchedLesson.ChapterNumber,
			fetchedLesson.LessonNumber,
			fetchedLesson.Lesson.Title,
		)
		lesson = fetchedLesson
	} else {
		nextLesson, err := api.FetchNextCLILesson()
		if err != nil {
			return err
		}
		lessonLabel := fmt.Sprintf("%d.%d - %s", nextLesson.ChapterNumber, nextLesson.LessonNumber, nextLesson.LessonTitle)
		if isSubmit {
			if !promptToContinue(
				"Submitting next lesson:",
				lessonLabel,
				"Continue?",
			) {
				return errors.New("submission canceled")
			}
		} else {
			fmt.Printf("Running next lesson:\n%s\n", lessonLabel)
		}
		lessonUUID = nextLesson.LessonUUID
		fetchedLesson, err := api.FetchLesson(lessonUUID)
		if err != nil {
			return err
		}
		lesson = fetchedLesson
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

	send, finish := render.StartRenderer(isSubmit, verboseOutput)
	finalEvent := api.LessonSubmissionEvent{}
	var debugPath string
	var debugWriteErr error
	defer func() {
		finish(finalEvent)
		if debugSubmission && isSubmit {
			reportDebugFileWrite(debugPath, debugWriteErr)
		}
	}()

	cliResults := checks.CLIChecks(data, overrideBaseURL, send)

	if isSubmit {
		submissionEvent, debugData, err := api.SubmitCLILesson(lessonUUID, cliResults, debugSubmission)
		if debugSubmission {
			debugPath, debugWriteErr = writeSubmissionDebugFile(lessonUUID, debugData)
		}
		if err != nil {
			return err
		}
		finalEvent = submissionEvent
		if err := applySubmissionEvent(data, submissionEvent, send); err != nil {
			return err
		}
	}
	return nil
}

func applySubmissionEvent(data api.CLIData, event api.LessonSubmissionEvent, send func(tea.Msg)) error {
	if event.ResultSlug == api.VerificationResultSlugSystemError {
		return errors.New("lesson verification failed due to a system error; please try again")
	}
	checks.ApplySubmissionResults(data, event.StructuredErrCLI, send)
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
