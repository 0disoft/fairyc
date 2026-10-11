// Test-only document oracle. Candidate parsing remains implemented in Fairy.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/BurntSushi/toml"
)

type input struct {
	Name   string `json:"name"`
	Source []byte `json:"source"`
}
type observation struct {
	Name             string `json:"name"`
	Accepted         bool   `json:"accepted"`
	ManifestAccepted bool   `json:"manifestAccepted"`
	ManifestReason   string `json:"manifestReason,omitempty"`
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("request and report paths are required")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return fmt.Errorf("request exceeds 2 MiB")
	}
	var request struct {
		Schema string  `json:"schema"`
		Cases  []input `json:"cases"`
	}
	if err = json.Unmarshal(data, &request); err != nil {
		return err
	}
	if request.Schema != "fairyc.candidate-toml-documents-request/v1" || len(request.Cases) == 0 || len(request.Cases) > 2048 {
		return fmt.Errorf("invalid schema or case count")
	}
	seen := map[string]bool{}
	rows := make([]observation, 0, len(request.Cases))
	for _, item := range request.Cases {
		if item.Name == "" || seen[item.Name] || len(item.Source) > 1<<20 {
			return fmt.Errorf("invalid or duplicate case %q", item.Name)
		}
		seen[item.Name] = true
		row := observation{Name: item.Name}
		if utf8.Valid(item.Source) {
			var value map[string]any
			_, parseErr := toml.Decode(string(item.Source), &value)
			row.Accepted = parseErr == nil
		}
		_, manifestErr := projectmanifest.Parse(item.Source)
		row.ManifestAccepted = manifestErr == nil
		if manifestErr != nil {
			if validation, ok := manifestErr.(*projectmanifest.ValidationError); ok {
				row.ManifestReason = validation.Reason
			} else {
				return manifestErr
			}
		}
		rows = append(rows, row)
	}
	report := struct {
		Schema string        `json:"schema"`
		Cases  []observation `json:"cases"`
	}{"fairyc.candidate-toml-documents-reference/v1", rows}
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(args[1], append(output, '\n'), 0600)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
