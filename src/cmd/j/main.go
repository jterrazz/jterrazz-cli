package main

import (
	"os"

	"github.com/jterrazz/jterrazz-cli/src/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
