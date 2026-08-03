//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package checks

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

func configureCommandCancellation(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killCommandProcessGroup(cmd) }
}

func forwardSignalsToCommand(cmd *exec.Cmd) func() {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	var forwardOnce sync.Once
	forward := func(received os.Signal) {
		forwardOnce.Do(func() {
			_ = killCommandProcessGroup(cmd)
			signal.Reset(received)
			if unixSignal, ok := received.(syscall.Signal); ok {
				_ = syscall.Kill(os.Getpid(), unixSignal)
			}
		})
	}

	go func() {
		select {
		case received := <-signals:
			forward(received)
		case <-done:
		}
	}()

	return func() {
		signal.Stop(signals)
		select {
		case received := <-signals:
			forward(received)
		default:
		}
		close(done)
	}
}

func killCommandProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}

	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}
