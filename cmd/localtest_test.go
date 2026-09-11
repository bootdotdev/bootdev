package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReadLocalCLIDataAcceptsLessonDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := []byte(`allowedOperatingSystems:
  - linux
  - darwin
baseURLDefault: http://localhost:3000
steps:
  - description: Prints a greeting
    cliCommand:
      command: echo hello
      tests:
        - exitCode: 0
        - stdoutContainsAll:
            - hello
`)
	if err := os.WriteFile(filepath.Join(dir, "cli.yaml"), manifest, 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	data, err := readLocalCLIData(dir)
	if err != nil {
		t.Fatalf("readLocalCLIData() error = %v", err)
	}
	if data.BaseURLDefault != "http://localhost:3000" {
		t.Fatalf("BaseURLDefault = %q, want localhost default", data.BaseURLDefault)
	}
	if len(data.Steps) != 1 || data.Steps[0].CLICommand == nil {
		t.Fatalf("expected one CLI command step, got %#v", data.Steps)
	}
	if data.Steps[0].Description != "Prints a greeting" {
		t.Fatalf("Description = %q, want manifest description", data.Steps[0].Description)
	}
	if len(data.Steps[0].CLICommand.Tests[1].StdoutContainsAll) != 1 {
		t.Fatalf("expected stdoutContainsAll test to load")
	}
}

func TestLocalTestShellDoesNotBypassOSValidation(t *testing.T) {
	for _, allowedOS := range []string{"[]", "[unsupported-os]"} {
		t.Run(allowedOS, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cli.yaml")
			manifest := "allowedOperatingSystems: " + allowedOS + "\nsteps:\n  - cliCommand:\n      command: echo hello\n"
			if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			command := &cobra.Command{}
			command.Flags().Bool("ignore-os", false, "")
			command.Flags().String("shell", "pwsh", "")
			err := localTestHandler(command, []string{path})
			if err == nil || !strings.Contains(err.Error(), "operating system") {
				t.Fatalf("error = %v, want OS validation error", err)
			}
		})
	}
}
