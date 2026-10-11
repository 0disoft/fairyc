// Test-only document oracle. Candidate parsing remains implemented in Fairy.
package main

import (
	"bytes"
	"encoding/binary"
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
	Name             string                    `json:"name"`
	Accepted         bool                      `json:"accepted"`
	ManifestAccepted bool                      `json:"manifestAccepted"`
	ManifestReason   string                    `json:"manifestReason,omitempty"`
	Manifest         *projectmanifest.Manifest `json:"manifest,omitempty"`
}

func wire(output *bytes.Buffer, source []byte, row observation) error {
	if len(source) > 65535 {
		return fmt.Errorf("binary source exceeds uint16")
	}
	if err := binary.Write(output, binary.LittleEndian, uint16(len(source))); err != nil {
		return err
	}
	accepted := byte(0)
	if row.ManifestAccepted {
		accepted = 1
	}
	output.WriteByte(accepted)
	reasons := map[string]byte{"": 0, "parse-error": 1, "unknown-field": 2, "missing-field": 3, "invalid-package-uid": 4, "invalid-name": 5, "invalid-version": 6, "invalid-language-revision": 7, "invalid-path": 8, "not-sorted-unique": 9, "unsupported-schema": 10, "input-too-large": 11, "invalid-utf8": 12}
	reason, ok := reasons[row.ManifestReason]
	if !ok {
		return fmt.Errorf("unknown reason %q", row.ManifestReason)
	}
	output.WriteByte(reason)
	output.Write(source)
	if !row.ManifestAccepted {
		return nil
	}
	m := row.Manifest
	stringValue := func(value string) error {
		if len(value) > 65535 {
			return fmt.Errorf("binary string exceeds uint16")
		}
		if err := binary.Write(output, binary.LittleEndian, uint16(len(value))); err != nil {
			return err
		}
		output.WriteString(value)
		return nil
	}
	list := func(values []string) error {
		if len(values) > 65535 {
			return fmt.Errorf("binary list exceeds uint16")
		}
		if err := binary.Write(output, binary.LittleEndian, uint16(len(values))); err != nil {
			return err
		}
		for _, value := range values {
			if err := stringValue(value); err != nil {
				return err
			}
		}
		return nil
	}
	for _, value := range []string{m.Package.UID, m.Package.Name, m.Package.Version, m.Package.Language, m.Package.Source} {
		if err := stringValue(value); err != nil {
			return err
		}
	}
	present := byte(0)
	if m.Workspace.Present {
		present = 1
	}
	output.WriteByte(present)
	if err := list(m.Workspace.Members); err != nil {
		return err
	}
	if len(m.Dependencies) > 65535 {
		return fmt.Errorf("binary dependency count exceeds uint16")
	}
	if err := binary.Write(output, binary.LittleEndian, uint16(len(m.Dependencies))); err != nil {
		return err
	}
	for _, dep := range m.Dependencies {
		for _, value := range []string{dep.Binding, dep.Path, dep.Version} {
			if err := stringValue(value); err != nil {
				return err
			}
		}
		if err := list(dep.Features); err != nil {
			return err
		}
	}
	return nil
}

func run(args []string) error {
	if len(args) != 2 && len(args) != 3 {
		return fmt.Errorf("request, report and optional binary fixture paths are required")
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
	var outputFixture bytes.Buffer
	outputFixture.WriteString("FYMNF001\n")
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
		manifest, manifestErr := projectmanifest.Parse(item.Source)
		row.ManifestAccepted = manifestErr == nil
		if row.ManifestAccepted {
			row.Manifest = &manifest
		}
		if manifestErr != nil {
			if validation, ok := manifestErr.(*projectmanifest.ValidationError); ok {
				row.ManifestReason = validation.Reason
			} else {
				return manifestErr
			}
		}
		rows = append(rows, row)
		if len(args) == 3 {
			if err := wire(&outputFixture, item.Source, row); err != nil {
				return err
			}
			if outputFixture.Len() > 65536 {
				return fmt.Errorf("binary fixture exceeds 64 KiB")
			}
		}
	}
	report := struct {
		Schema string        `json:"schema"`
		Cases  []observation `json:"cases"`
	}{"fairyc.candidate-toml-documents-reference/v1", rows}
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(args[1], append(output, '\n'), 0600); err != nil {
		return err
	}
	if len(args) == 3 {
		return os.WriteFile(args[2], outputFixture.Bytes(), 0600)
	}
	return nil
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
