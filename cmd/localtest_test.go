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
			for name, contents := range map[string]string{"cli.yaml": "steps: []", "readme.md": tt.readme} {
				if contents != "" {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, path := range []string{dir, filepath.Join(dir, "cli.yaml")} {
				got := localRunContext(path)
				title := tt.wantTitle
				if title == "" {
					title = filepath.Base(dir)
				}
				if got.LessonTitle != title {
					t.Fatalf("unexpected context: %+v", got)
				}
			}
		})
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

func TestLocalRunContextUsesCourseLessonNumber(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "9-documentation", "3-sorting-chirp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Sorting Chirps\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := localRunContext(dir); got.LessonTitle != "9.3 - Sorting Chirps" {
		t.Fatalf("unexpected title: %s", got.LessonTitle)
	}
}
