package checks

import (
	"os/exec"

	api "github.com/bootdotdev/bootdev/client"
)

func CLICommand(
	assignment api.Assignment,
) []api.CLICommandResult {
	data := assignment.Assignment.AssignmentDataCLICommand.CLICommandData
	responses := make([]api.CLICommandResult, len(data.Commands))
	for i, command := range data.Commands {
		cmd := exec.Command("sh", "-c", command.Command)
		b, err := cmd.Output()
		if ee, ok := err.(*exec.ExitError); ok {
			responses[i].ExitCode = ee.ExitCode()
		} else if err != nil {
			responses[i].ExitCode = -2
		}
		responses[i].Stdout = string(b)
	}
	return responses
}
