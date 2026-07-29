package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bootdotdev/bootdev/version"
	"github.com/spf13/cobra"
)

func TestSecureConfigFileRestrictsExistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix file permission semantics")
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("access_token: secret\n"), 0o644); err != nil {
		t.Fatalf("create config file: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("set initial config permissions: %v", err)
	}

	if err := secureConfigFile(path); err != nil {
		t.Fatalf("secureConfigFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config permissions = %o, want 600", got)
	}
}

func TestSecureConfigFileAllowsEmptyPath(t *testing.T) {
	if err := secureConfigFile(""); err != nil {
		t.Fatalf("secureConfigFile() error = %v", err)
	}
}

func TestExecuteSkipsVersionLookupForHelp(t *testing.T) {
	originalFetch := fetchUpdateInfo
	originalOut := rootCmd.OutOrStdout()
	originalErr := rootCmd.ErrOrStderr()
	t.Cleanup(func() {
		fetchUpdateInfo = originalFetch
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(originalOut)
		rootCmd.SetErr(originalErr)
	})

	called := false
	fetchUpdateInfo = func(currentVersion string) version.VersionInfo {
		called = true
		return version.VersionInfo{CurrentVersion: currentVersion}
	}
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)

	if err := Execute("v1.2.3"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called {
		t.Fatal("help unexpectedly triggered a version lookup")
	}
}

func TestLoadVersionInfoPopulatesCommandContext(t *testing.T) {
	originalFetch := fetchUpdateInfo
	t.Cleanup(func() {
		fetchUpdateInfo = originalFetch
	})

	fetchUpdateInfo = func(currentVersion string) version.VersionInfo {
		return version.VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  "v1.2.4",
			IsOutdated:     true,
		}
	}

	info := version.VersionInfo{}
	cmd := &cobra.Command{Version: "v1.2.3"}
	cmd.SetContext(version.WithContext(context.Background(), &info))

	loadVersionInfo(cmd, nil)

	if info.CurrentVersion != "v1.2.3" || info.LatestVersion != "v1.2.4" || !info.IsOutdated {
		t.Fatalf("version context was not populated: %#v", info)
	}
}
