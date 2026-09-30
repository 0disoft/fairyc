package main

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/languageserver"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("fairyc-lsp 0.1.0")
			return
		case "--help", "-h":
			fmt.Println("fairyc-lsp: local stdio Language Server Protocol preview (diagnostics, definition, completion)")
			return
		default:
			fmt.Fprintln(os.Stderr, "fairyc-lsp accepts no server arguments")
			os.Exit(2)
		}
	}
	os.Exit(languageserver.Serve(os.Stdin, os.Stdout, os.Stderr))
}
