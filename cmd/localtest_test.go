package cmd

import (
	"os"
	"path/filepath"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
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

func TestLocalTestFailureErrorIncludesStructuredContext(t *testing.T) {
	err := localTestFailureError(&api.StructuredErrCLI{
		ErrorMessage:    `expected stdout to contain "hello"`,
		FailedStepIndex: 1,
		FailedTestIndex: 2,
	})

	want := "local checks failed: step 2, test 3\nexpected stdout to contain \"hello\""
	if err == nil || err.Error() != want {
		t.Fatalf("localTestFailureError() = %v, want %q", err, want)
	}
}
