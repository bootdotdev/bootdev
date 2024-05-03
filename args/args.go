package args

import (
	"fmt"
	"strings"
)

func InterpolateCommand(rawCommand string, optionalPositionalArgs []string) string {
	// replace $1, $2, etc. with the optional positional args
	for i, arg := range optionalPositionalArgs {
		rawCommand = strings.ReplaceAll(rawCommand, fmt.Sprintf("$%d", i+1), arg)
	}
	return rawCommand
}
