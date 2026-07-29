package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func preserveLogoutConfig(t *testing.T) {
	t.Helper()

	originalConfigFile := viper.ConfigFileUsed()
	originalAPIURL := viper.GetString("api_url")
	originalAccessToken := viper.GetString("access_token")
	originalRefreshToken := viper.GetString("refresh_token")
	originalLastRefresh := viper.GetInt64("last_refresh")
	t.Cleanup(func() {
		viper.Set("api_url", originalAPIURL)
		viper.Set("access_token", originalAccessToken)
		viper.Set("refresh_token", originalRefreshToken)
		viper.Set("last_refresh", originalLastRefresh)
		viper.SetConfigFile(originalConfigFile)
	})
}

func TestLogoutClearsLocalCredentialsWhenServerLogoutFails(t *testing.T) {
	preserveLogoutConfig(t)

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("create test config: %v", err)
	}
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read test config: %v", err)
	}
	viper.Set("api_url", "://invalid-api-url")
	viper.Set("access_token", "access-token")
	viper.Set("refresh_token", "refresh-token")
	viper.Set("last_refresh", int64(123))

	if err := logout(); err != nil {
		t.Fatalf("logout() error = %v", err)
	}

	stored := viper.New()
	stored.SetConfigFile(configPath)
	if err := stored.ReadInConfig(); err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if got := stored.GetString("access_token"); got != "" {
		t.Fatalf("persisted access token = %q, want empty", got)
	}
	if got := stored.GetString("refresh_token"); got != "" {
		t.Fatalf("persisted refresh token = %q, want empty", got)
	}
	if got := stored.GetInt64("last_refresh"); got != 0 {
		t.Fatalf("persisted last refresh = %d, want 0", got)
	}
}

func TestLogoutReturnsConfigWriteError(t *testing.T) {
	preserveLogoutConfig(t)

	configPath := filepath.Join(t.TempDir(), "missing", "config.yaml")
	viper.SetConfigFile(configPath)
	viper.Set("access_token", "access-token")
	viper.Set("refresh_token", "")
	viper.Set("last_refresh", int64(123))

	err := logout()
	if err == nil {
		t.Fatal("logout() unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "failed to clear stored credentials") {
		t.Fatalf("logout() error = %q, want config write error", err)
	}
}

func TestLogoutCommandDoesNotRequireAuthentication(t *testing.T) {
	if logoutCmd.PreRun != nil {
		t.Fatal("logout command unexpectedly requires authentication")
	}
}
