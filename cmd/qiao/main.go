package main

import (
	"os"

	"github.com/raoooool/qiao/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
