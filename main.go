package main

import (
	"os"

	"tkx/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Deps{
		Version: version,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}))
}
