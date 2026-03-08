package main

import (
	"os"

	"github.com/docbiner/docbiner/cmd/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
