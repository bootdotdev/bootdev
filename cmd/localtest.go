package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/bootdotdev/bootdev/checks"
	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func init() {
	rootCmd.AddCommand(localTestCmd)
	localTestCmd.Flags().Bool("ignore-os", false, "skip the lesson's allowed operating systems check")
	localTestCmd.Flags().Bool("run", false, "run local lesson commands without grading")
	localTestCmd.Flags().String("shell", "", "shell for lesson commands: sh or pwsh (default depends on host OS)")
	localTestCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "show detailed final output for every step")
}

var localTestCmd = &cobra.Command{
	Use:     "local-test PATH",
	Args:    cobra.ExactArgs(1),
	Hidden:  true,
	RunE:    localTestHandler,
	Example: "  bootdev local-test ./cli.yaml --ignore-os --shell pwsh",
}

func localTestHandler(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = false

	data, err := readLocalCLIData(args[0])
	if err != nil {
		return err
	}
	ignoreOS, err := cmd.Flags().GetBool("ignore-os")
	if err != nil {
		return err
	}
	if !ignoreOS {
		if err := validateAllowedOS(data); err != nil {
			return err
		}
	}
	shell, err := cmd.Flags().GetString("shell")
	if err != nil {
		return err
	}
	runOnly, err := cmd.Flags().GetBool("run")
	if err != nil {
		return err
	}

	overrideBaseURL := viper.GetString("override_base_url")
	runContext := localRunContext(args[0])
	if runOnly {
		runContext.Action = "Running lesson:"
	}
	runContext.BaseURL = data.BaseURLDefault
	if overrideBaseURL != "" {
		runContext.BaseURL = overrideBaseURL
	}

	send, finish := render.StartRenderer(!runOnly, verboseOutput, false, runContext)
	submissionEvent := api.LessonSubmissionEvent{}
	defer func() {
		finish(submissionEvent)
	}()

	cliResults, err := checks.CLIChecks(data, checks.RunOptions{OverrideBaseURL: overrideBaseURL, Shell: shell}, send)
	if err != nil {
		return err
	}
	if runOnly {
		return nil
	}
	submissionEvent = checks.LocalSubmissionEvent(data, cliResults)
	checks.ApplySubmissionResults(data, submissionEvent.StructuredErrCLI, send)

	if submissionEvent.ResultSlug != api.VerificationResultSlugSuccess {
		cmd.SilenceErrors = true
		return localTestFailureError(submissionEvent.StructuredErrCLI)
	}

	return nil
}

func localRunContext(path string) render.RunContext {
	manifestPath := filepath.Clean(path)
	if info, err := os.Stat(manifestPath); err == nil && info.IsDir() {
		manifestPath = filepath.Join(manifestPath, "cli.yaml")
	}
	lessonDirectory := filepath.Dir(manifestPath)
	context := render.RunContext{
		Action:      "Submitting lesson:",
		LessonTitle: filepath.Base(lessonDirectory),
	}
	if readme, err := os.ReadFile(filepath.Join(lessonDirectory, "readme.md")); err == nil {
		for _, line := range strings.Split(string(readme), "\n") {
			if strings.HasPrefix(line, "# ") {
				if title := strings.TrimSpace(strings.TrimPrefix(line, "# ")); title != "" {
					context.LessonTitle = title
				}
				break
			}
		}
	}
	chapterPrefix, _, chapterFound := strings.Cut(filepath.Base(filepath.Dir(lessonDirectory)), "-")
	lessonPrefix, _, lessonFound := strings.Cut(filepath.Base(lessonDirectory), "-")
	chapterNumber, chapterErr := strconv.Atoi(chapterPrefix)
	lessonNumber, lessonErr := strconv.Atoi(lessonPrefix)
	if chapterFound && lessonFound && chapterErr == nil && lessonErr == nil && chapterNumber > 0 && lessonNumber > 0 {
		context.LessonTitle = fmt.Sprintf("%d.%d - %s", chapterNumber, lessonNumber, context.LessonTitle)
	}
	return context
}

func localTestFailureError(failure *api.StructuredErrCLI) error {
	if failure == nil {
		return errors.New("local checks failed")
	}
	return fmt.Errorf(
		"local checks failed: step %d, test %d\n%s",
		failure.FailedStepIndex+1,
		failure.FailedTestIndex+1,
		failure.ErrorMessage,
	)
}

func readLocalCLIData(path string) (api.CLIData, error) {
	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return api.CLIData{}, err
	}
	if info.IsDir() {
		cleanPath = filepath.Join(cleanPath, "cli.yaml")
	}

	bytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return api.CLIData{}, err
	}

	var data api.CLIData
	if err := yaml.Unmarshal(bytes, &data); err != nil {
		return api.CLIData{}, err
	}
	if len(data.Steps) == 0 {
		return api.CLIData{}, errors.New("test manifest should include at least one step")
	}

	return data, nil
}

func validateAllowedOS(data api.CLIData) error {
	if len(data.AllowedOperatingSystems) == 0 {
		return errors.New("lesson does not specify any allowed operating systems")
	}

	if slices.Contains(data.AllowedOperatingSystems, runtime.GOOS) {
		return nil
	}

	return fmt.Errorf(
		"lesson is not supported for your operating system (%s)\ntry again with one of the following: %v",
		runtime.GOOS,
		data.AllowedOperatingSystems,
	)
}
