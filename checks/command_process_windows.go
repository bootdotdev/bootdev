//go:build windows

package checks

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
)

func configureCommandCancellation(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}

		treeKill := exec.Command(
			"taskkill.exe",
			"/PID", strconv.Itoa(cmd.Process.Pid),
			"/T",
			"/F",
		)
		if err := treeKill.Run(); err == nil {
			return nil
		}

		err := cmd.Process.Kill()
		if errors.Is(err, os.ErrProcessDone) {
			return os.ErrProcessDone
		}
		return err
	}
}

func forwardSignalsToCommand(cmd *exec.Cmd) func() {
	return func() {}
}
