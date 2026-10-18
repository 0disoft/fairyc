// Test-only filesystem observations from the public reference loader.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"os"
	"path/filepath"
)

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: directory-reference <fixture-root> <report.json>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	type row struct {
		Source   string
		Accepted bool
		Reason   string
	}
	rows := []row{}
	for _, source := range []string{"Source/Nested", "source", "Source/nested", "Missing", "ordinary.txt", "k", "Link"} {
		manifest := fmt.Sprintf("schema='fairyc.manifest/v1'\n[package]\nuid='example.directory'\nname='directory'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource=%q\n", source)
		if err = os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0600); err != nil {
			return err
		}
		_, err = workspaceloader.Load(root)
		r := row{Source: source, Accepted: err == nil}
		if err != nil {
			if e, ok := err.(*workspaceloader.LoadError); ok {
				r.Reason = e.Reason
			} else {
				return err
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
