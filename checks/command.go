package checks

import (
	"os/exec"
	"strings"

	"github.com/bootdotdev/bootdev/args"
	api "github.com/bootdotdev/bootdev/client"
)

func CLICommand(
	lesson api.Lesson,
	optionalPositionalArgs []string,
) []api.CLICommandResult {
	data := lesson.Lesson.LessonDataCLICommand.CLICommandData
	responses := make([]api.CLICommandResult, len(data.Commands))
	for i, command := range data.Commands {
		finalCommand := args.InterpolateCommand(command.Command, optionalPositionalArgs)
		cmd := exec.Command("sh", "-c", finalCommand)
		b, err := cmd.Output()
		if ee, ok := err.(*exec.ExitError); ok {
			responses[i].ExitCode = ee.ExitCode()
		} else if err != nil {
			responses[i].ExitCode = -2
		}
		responses[i].Stdout = strings.TrimRight(string(b), " \n\t\r")
	}
	return responses
}
