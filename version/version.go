package version

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/goccy/go-json"
	"golang.org/x/mod/semver"
)

const (
	modulePath         = "github.com/bootdotdev/bootdev"
	updateCheckTimeout = 10 * time.Second
)

type VersionInfo struct {
	CurrentVersion   string
	LatestVersion    string
	IsOutdated       bool
	IsUpdateRequired bool
	FailedToFetch    error
}

func FetchUpdateInfo(currentVersion string) VersionInfo {
	latest, err := getLatestVersion()
	if err != nil {
		return VersionInfo{
			CurrentVersion: currentVersion,
			FailedToFetch:  err,
		}
	}
	isUpdateRequired := isUpdateRequired(currentVersion, latest)
	isOutdated := isOutdated(currentVersion, latest)
	return VersionInfo{
		IsUpdateRequired: isUpdateRequired,
		IsOutdated:       isOutdated,
		CurrentVersion:   currentVersion,
		LatestVersion:    latest,
	}
}

func (v *VersionInfo) PromptUpdateIfAvailable() {
	if v.IsOutdated {
		fmt.Fprintln(os.Stderr, "A new version of the bootdev CLI is available!")
		fmt.Fprintln(os.Stderr, "Please run the following command to update:")
		fmt.Fprintln(os.Stderr, "  bootdev upgrade")
		fmt.Fprintln(os.Stderr, "or")
		fmt.Fprintf(os.Stderr, "  go install github.com/bootdotdev/bootdev@%s\n\n", v.LatestVersion)
	}
}

// Returns true if the current version is older than the latest.
func isOutdated(current string, latest string) bool {
	return semver.Compare(current, latest) < 0
}

// Returns true if the latest version has a higher major or minor
// number than the current version. If you don't want to force
// an update, you can increment the patch number instead.
func isUpdateRequired(current string, latest string) bool {
	latestMajorMinor := semver.MajorMinor(latest)
	currentMajorMinor := semver.MajorMinor(current)
	return semver.Compare(currentMajorMinor, latestMajorMinor) < 0
}

func getLatestVersion() (string, error) {
	return getLatestVersionWithTimeout(updateCheckTimeout)
}

func getLatestVersionWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", modulePath+"@latest")
	output, err := cmd.Output()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return "", fmt.Errorf("latest version check failed: %w", ctxErr)
	}
	if err != nil {
		return "", fmt.Errorf("failed to query latest version: %w", err)
	}

	var latest struct{ Version string }
	if err := json.Unmarshal(output, &latest); err != nil {
		return "", fmt.Errorf("failed to parse latest version: %w", err)
	}
	if latest.Version == "" {
		return "", fmt.Errorf("latest version response did not include a version")
	}
	return latest.Version, nil
}
