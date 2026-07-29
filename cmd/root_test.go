package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
	"github.com/bootdotdev/bootdev/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func TestRefreshCredentialsPersistsRotatedCredentials(t *testing.T) {
	const (
		oldAccessToken  = "old-access-token"
		oldRefreshToken = "old-refresh-token"
		newAccessToken  = "new-access-token"
		newRefreshToken = "new-refresh-token"
	)

	originalFetchAccessToken := fetchAccessToken
	originalConfigFile := viper.ConfigFileUsed()
	originalAccessToken := viper.GetString("access_token")
	originalRefreshToken := viper.GetString("refresh_token")
	originalLastRefresh := viper.GetInt64("last_refresh")
	t.Cleanup(func() {
		fetchAccessToken = originalFetchAccessToken
		viper.Set("access_token", originalAccessToken)
		viper.Set("refresh_token", originalRefreshToken)
		viper.Set("last_refresh", originalLastRefresh)
		viper.SetConfigFile(originalConfigFile)
	})
	fetchAccessToken = func() (*api.LoginResponse, error) {
		return &api.LoginResponse{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		}, nil
	}

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("create test config: %v", err)
	}
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read test config: %v", err)
	}
	viper.Set("access_token", oldAccessToken)
	viper.Set("refresh_token", oldRefreshToken)

	if err := refreshCredentials(); err != nil {
		t.Fatalf("refreshCredentials() error = %v", err)
	}

	if got := viper.GetString("access_token"); got != newAccessToken {
		t.Fatalf("access token = %q, want %q", got, newAccessToken)
	}
	if got := viper.GetString("refresh_token"); got != newRefreshToken {
		t.Fatalf("refresh token = %q, want %q", got, newRefreshToken)
	}

	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if !strings.Contains(string(config), newAccessToken) || !strings.Contains(string(config), newRefreshToken) {
		t.Fatalf("persisted config does not contain rotated credentials:\n%s", config)
	}
}
