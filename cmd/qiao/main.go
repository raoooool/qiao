package main

import (
	"os"

	"github.com/raoooool/qiao/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
