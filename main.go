package main

import (
	"os"
	"strings"

	_ "embed"

	"github.com/bootdotdev/bootdev/cmd"
)

//go:embed version.txt
var version string

func main() {
	err := cmd.Execute(strings.TrimSpace(version))
	if err != nil {
		os.Exit(1)
	}
}
