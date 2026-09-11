package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLocalTestRedirectedOutput(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "bootdev.exe")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	config := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(config, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	command := `printf '\033[31mstdout-marker\033[0m\n'; printf 'stderr-marker\n' >&2`
	if runtime.GOOS == "windows" {
		command = `[Console]::Out.WriteLine([char]27 + '[31mstdout-marker' + [char]27 + '[0m'); [Console]::Error.WriteLine('stderr-marker')`
	}
	for _, tt := range []struct {
		name    string
		fail    bool
		verbose bool
	}{
		{"pass verbose", false, true},
		{"fail verbose", true, true},
		{"pass compact", false, false},
		{"fail compact", true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wantExit := 0
			if tt.fail {
				wantExit = 1
			}
			manifest := filepath.Join(t.TempDir(), "cli.yaml")
			data := fmt.Sprintf("allowedOperatingSystems: [%s]\nsteps:\n  - description: Output check\n    cliCommand:\n      command: %s\n      tests:\n        - exitCode: %d\n", runtime.GOOS, command, wantExit)
			if err := os.WriteFile(manifest, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			args := []string{"--config", config, "local-test", manifest}
			if tt.verbose {
				args = append(args, "--verbose")
			}
			cmd := exec.CommandContext(ctx, binary, args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != wantExit {
				t.Fatalf("exit status: %v; stdout: %s; stderr: %s", err, &stdout, &stderr)
			}
			output := stdout.String()
			if strings.Contains(output+stderr.String(), "\x1b") || strings.Contains(stderr.String(), "TTY") {
				t.Fatalf("terminal output leaked: stdout=%q stderr=%q", output, stderr.String())
			}
			if strings.Count(output, "Output check") != 1 {
				t.Fatalf("expected one final report: %s", output)
			}
			for _, diagnostic := range []string{"Command stdout:\n\nstdout-marker", "Command stderr:\n\nstderr-marker"} {
				if got, want := strings.Contains(output, diagnostic), tt.verbose || tt.fail; got != want {
					t.Fatalf("diagnostic %q present = %t, want %t; output: %s", diagnostic, got, want, output)
				}
			}
			if tt.fail {
				if !strings.Contains(stderr.String(), "local checks failed") {
					t.Fatalf("missing failure error: %s", &stderr)
				}
			} else if stderr.Len() != 0 || !strings.Contains(output, "All tests passed!") {
				t.Fatalf("unexpected success output: stdout=%q stderr=%q", output, stderr.String())
			}
		})
	}
}
