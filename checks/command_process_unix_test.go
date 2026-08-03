//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package checks

import (
	"errors"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	api "github.com/bootdotdev/bootdev/client"
)

func TestRunCLICommandTimeoutKillsDescendants(t *testing.T) {
	result := runCLICommandWithLimits(
		api.CLIStepCLICommand{Command: "sleep 30 & echo $!; wait"},
		map[string]string{},
		100*time.Millisecond,
		1024,
	)
	if !strings.Contains(result.Err, "command timed out") {
		t.Fatalf("command error = %q, want timeout error", result.Err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		t.Fatalf("child PID output = %q: %v", result.Stdout, err)
	}
	childAlive := true
	t.Cleanup(func() {
		if childAlive {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			childAlive = false
			return
		}
		if err != nil {
			t.Fatalf("check child process %d: %v", pid, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("child process %d survived command cancellation", pid)
}
