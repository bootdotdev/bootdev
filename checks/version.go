package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"

	"golang.org/x/mod/semver"
)

const repoOwner = "bootdotdev"
const repoName = "bootdev"

type GHTag struct {
	Name string `json:"name"`
}

func PromptUpdateIfNecessary(version string) error {
	outdated, err := isOutdated(version)
	if err != nil {
		return err
	}
	if outdated {
		fmt.Println("A new version of the bootdev CLI is available!")
		fmt.Println("Please run the following command to update:")
		fmt.Printf("  go install github.com/%s/%s@latest\n", repoOwner, repoName)
		os.Exit(1)
	}
	return nil
}

func isOutdated(currentVersion string) (bool, error) {
	latestVersion, err := getLatestVersion()
	if err != nil {
		return false, err
	}
	if semver.Compare(currentVersion, latestVersion) == -1 {
		return true, nil
	}
	return false, nil
}

func getLatestVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", repoOwner, repoName))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tags []GHTag
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return "", err
	}

	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare(tags[j].Name, tags[i].Name) == -1
	})

	if len(tags) == 0 {
		return "", errors.New("no tags found")
	}

	return tags[0].Name, nil
}
