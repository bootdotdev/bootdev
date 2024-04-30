package main

import (
	"os"

	"github.com/bootdotdev/bootdev/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
