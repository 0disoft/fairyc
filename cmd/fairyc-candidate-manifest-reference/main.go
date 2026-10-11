// Test-only oracle for decoded manifest-v1 values. The Fairy leaf implements
// the rules; this command only serializes current Go observations as fixtures.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

type input struct {
	Name  string `json:"name"`
	Mode  uint8  `json:"mode"`
	Value []byte `json:"value"`
}

func observe(item input) bool {
	value := string(item.Value)
	switch item.Mode {
	case 1:
		return projectmanifest.ValidPackageName(value)
	case 2:
		return projectmanifest.ValidPackageUID(value)
	case 5:
		return projectmanifest.ValidPortablePathSegment(value)
	}
	quoted, _ := json.Marshal(value)
	fields := map[string]string{"uid": "\"example.app\"", "name": "\"app\"", "version": "\"1.0.0\"", "language": "\"fairyc-language/20\"", "source": "\"src\""}
	field := map[uint8]string{3: "language", 4: "version", 6: "source"}[item.Mode]
	fields[field] = string(quoted)
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\n"
	for _, key := range []string{"uid", "name", "version", "language", "source"} {
		manifest += key + " = " + fields[key] + "\n"
	}
	_, err := projectmanifest.Parse([]byte(manifest))
	return err == nil
}

func run(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("request, binary fixture and report paths are required")
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
	if request.Schema != "fairyc.candidate-manifest-values-request/v1" || len(request.Cases) == 0 || len(request.Cases) > 2048 {
		return fmt.Errorf("invalid request schema or case count")
	}
	var output bytes.Buffer
	output.WriteString("FYMFVAL1\n")
	type result struct {
		Name     string `json:"name"`
		Mode     uint8  `json:"mode"`
		Accepted bool   `json:"accepted"`
	}
	var rows []result
	seen := map[string]bool{}
	for _, item := range request.Cases {
		if item.Name == "" || seen[item.Name] || item.Mode < 1 || item.Mode > 6 || len(item.Value) > 4096 {
			return fmt.Errorf("invalid or duplicate case %q", item.Name)
		}
		seen[item.Name] = true
		accepted := observe(item)
		expected := byte(0)
		if accepted {
			expected = 1
		}
		output.Write([]byte{item.Mode, expected})
		var size [2]byte
		binary.LittleEndian.PutUint16(size[:], uint16(len(item.Value)))
		output.Write(size[:])
		output.Write(item.Value)
		rows = append(rows, result{item.Name, item.Mode, accepted})
	}
	if output.Len() > 65536 {
		return fmt.Errorf("binary fixture exceeds 65536 bytes")
	}
	report := struct {
		Schema string   `json:"schema"`
		Cases  []result `json:"cases"`
	}{"fairyc.candidate-manifest-values-reference/v1", rows}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(args[1], output.Bytes(), 0600); err != nil {
		return err
	}
	return os.WriteFile(args[2], append(encoded, '\n'), 0600)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
