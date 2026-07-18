package main

import (
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/cli"
)

func main() {
	sourceRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fairyc: cannot determine the current source root")
		os.Exit(cli.ExitInternalFailure)
	}
	os.Exit(cli.Run(os.Args[1:], sourceRoot, os.Stdout, os.Stderr))
}
