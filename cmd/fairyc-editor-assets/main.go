package main

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/editors"
	"os"
	"path/filepath"
)

func main() {
	assets, err := editors.Assets(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for name, data := range assets {
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(name, data, 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
