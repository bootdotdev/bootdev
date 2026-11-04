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
	manifest := []byte(`steps:
  - cliCommand:
      command: echo hello
`)
	if err := os.WriteFile(filepath.Join(dir, "cli.yaml"), manifest, 0o600); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	data, err := readLocalCLIData(dir)
	if err != nil {
		t.Fatalf("readLocalCLIData() error = %v", err)
	}
	if len(data.Steps) != 1 || data.Steps[0].CLICommand == nil || data.Steps[0].CLICommand.Command != "echo hello" {
		t.Fatalf("expected one command loaded from cli.yaml, got %#v", data.Steps)
	}
}

func TestLocalRunContext(t *testing.T) {
	for _, tt := range []struct {
		name      string
		readme    string
		wantTitle string
	}{
		{name: "lesson", readme: "# Sorting Chirps\n", wantTitle: "Sorting Chirps"},
		{name: "missing README"},
		{name: "README without title", readme: "No title"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.readme != "" {
				if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte(tt.readme), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			title := tt.wantTitle
			if title == "" {
				title = filepath.Base(dir)
			}
			if got := localRunContext(dir); got.LessonTitle != title {
				t.Fatalf("title = %q, want %q", got.LessonTitle, title)
			}
		})
	}
	t.Run("numbered lesson paths", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "9-documentation", "3-sorting-chirps")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Sorting Chirps\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Chdir(dir)
		for _, path := range []string{dir, filepath.Join(dir, "cli.yaml"), ".", "cli.yaml", "./cli.yaml"} {
			if got := localRunContext(path); got.LessonTitle != "9.3 - Sorting Chirps" {
				t.Errorf("path %q: unexpected title %q", path, got.LessonTitle)
			}
		}
	})
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
