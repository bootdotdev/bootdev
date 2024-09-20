package checks

import (
	"os/exec"
	"strings"

	api "github.com/bootdotdev/bootdev/client"
)

func CLICommand(
	lesson api.Lesson,
) []api.CLICommandResult {
	data := lesson.Lesson.LessonDataCLICommand.CLICommandData
	responses := make([]api.CLICommandResult, len(data.Commands))
	for i, command := range data.Commands {
		responses[i].FinalCommand = command.Command

		cmd := exec.Command("sh", "-c", "LANG=en_US.UTF-8 "+command.Command)
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
