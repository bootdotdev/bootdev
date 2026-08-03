//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package checks

import "os/exec"

func configureCommandCancellation(cmd *exec.Cmd) {}

func forwardSignalsToCommand(cmd *exec.Cmd) func() {
	return func() {}
}
