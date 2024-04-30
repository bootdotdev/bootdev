package main

import (
	"os"

	_ "embed"

	"github.com/bootdotdev/bootdev/cmd"
)

//go:embed VERSION
var version string

func main() {
	err := cmd.Execute(version)
	if err != nil {
		os.Exit(1)
	}
}
