package version

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGetLatestVersionHasOverallTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX executable script")
	}

	dir := t.TempDir()
	fakeGo := filepath.Join(dir, "go")
	if err := os.WriteFile(fakeGo, []byte("#!/bin/sh\nexec /bin/sleep 5\n"), 0o755); err != nil {
		t.Fatalf("create fake go command: %v", err)
	}
	t.Setenv("PATH", dir)

	start := time.Now()
	_, err := getLatestVersionWithTimeout(25 * time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("version lookup exceeded timeout allowance: %s", elapsed)
	}
}

func TestLatestVersionCommandIsIsolated(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=vendor")
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "go.work"))
	t.Setenv("GOPROXY", "https://proxy.example.com")

	cmd := latestVersionCommand(context.Background())
	if cmd.Dir != os.TempDir() {
		t.Fatalf("command directory = %q, want %q", cmd.Dir, os.TempDir())
	}
	if args := strings.Join(cmd.Args, " "); !strings.Contains(args, " -mod=mod ") {
		t.Fatalf("command arguments %q do not force module mode", args)
	}
	if got := lastEnvValue(cmd.Env, "GOWORK"); got != "off" {
		t.Fatalf("GOWORK = %q, want off", got)
	}
	if got := lastEnvValue(cmd.Env, "GOFLAGS"); got != "" {
		t.Fatalf("GOFLAGS = %q, want empty", got)
	}
	if got := lastEnvValue(cmd.Env, "GOPROXY"); got != "https://proxy.example.com" {
		t.Fatalf("GOPROXY = %q, want inherited value", got)
	}
}

func lastEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range slices.Backward(env) {
		if after, ok := strings.CutPrefix(entry, prefix); ok {
			return after
		}
	}
	return ""
}

func TestGetLatestVersionIncludesGoDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX executable script")
	}

	dir := t.TempDir()
	fakeGo := filepath.Join(dir, "go")
	if err := os.WriteFile(fakeGo, []byte("#!/bin/sh\necho 'go: module lookup disabled by GOPROXY=off' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("create fake go command: %v", err)
	}
	t.Setenv("PATH", dir)

	_, err := getLatestVersionWithTimeout(time.Second)
	if err == nil {
		t.Fatal("expected latest version lookup to fail")
	}
	if !strings.Contains(err.Error(), "module lookup disabled by GOPROXY=off") {
		t.Fatalf("error %q does not include go command diagnostic", err)
	}
}
