package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/prevrevision"
)

func main() {
	write := flag.Bool("write", false, "atomically replace every successfully planned Fairy C source")
	root := flag.String("root", ".", "repository root to scan")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "fairyc-migrate-prev accepts flags only")
		os.Exit(2)
	}
	report, err := prevrevision.MigrateTree(*root, *write)
	encoded, encodeErr := report.JSON()
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(encoded)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
