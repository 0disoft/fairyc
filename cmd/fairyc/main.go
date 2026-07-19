package main

import (
	"os"

	"github.com/0disoft/fairyc/internal/cli"
)

func main() {
	// Source-consuming commands open this root inside cli.Run. Source-independent commands never
	// touch it, so help, explain, and target inspection do not perform an eager filesystem query.
	os.Exit(cli.Run(os.Args[1:], ".", os.Stdout, os.Stderr))
}
