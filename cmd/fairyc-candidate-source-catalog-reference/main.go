// Test-only source-catalog observations from the public Go workspace loader.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: source-catalog-reference <fixture-root> <report.json>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type row struct {
		Source   string
		Accepted bool
		Reason   string
		Paths    []string
	}
	rows := []row{}
	for _, source := range []string{"Good", "Empty", "Collision", "Reparse", "IgnoredReparse"} {
		manifest := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid='example.catalog'\nname='catalog'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource=%q\n", source)
		if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0600); err != nil {
			return err
		}
		workspace, err := workspaceloader.Load(root)
		r := row{Source: source, Accepted: err == nil, Paths: []string{}}
		if err != nil {
			if e, ok := err.(*workspaceloader.LoadError); ok {
				r.Reason = e.Reason
			} else {
				return err
			}
		} else {
			for _, source := range workspace.Packages[0].Sources {
				r.Paths = append(r.Paths, source.Path)
			}
		}
		rows = append(rows, r)
	}
	out, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(args[1], append(out, '\n'), 0600)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
