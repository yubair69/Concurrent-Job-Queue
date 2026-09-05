package main

import (
	"os"

	"github.com/gotask/gotask/internal/cli"
)

func main() {
	exitCode := cli.Run(os.Args)
	os.Exit(exitCode)
}
