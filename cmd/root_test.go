package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
