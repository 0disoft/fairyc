//go:build ignore

// Compare complete original module AST and nominal definitions to the Go owner.
// bun tools/run-go.ts run tools/verify-candidate-source-module-json.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	authored, err := os.ReadFile("examples/lexer_candidate/tests/generic_source_module_json.fy")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	matches := pattern.FindAllSubmatch(authored, -1)
	if len(matches) != 17 {
		return fmt.Errorf("expected seventeen authored sources")
	}
	var inputs [][]byte
	for i, match := range matches {
		index, err := strconv.Atoi(string(match[1]))
		if err != nil || index != i {
			return fmt.Errorf("noncontiguous inputs")
		}
		text, err := strconv.Unquote(string(match[2]))
		if err != nil {
			return err
		}
		inputs = append(inputs, []byte(text))
	}
	const sdkPath = "internal/sdkstdlib/sources/optional/std/vec.fy"
	sdk, err := os.ReadFile(sdkPath)
	if err != nil {
		return err
	}
	inputs = append(inputs, sdk)
	var out bytes.Buffer
	out.WriteString("FYASTMD1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	number(len(inputs))
	var cases []map[string]any
	definitions := 0
	for i, text := range inputs {
		path := "sample.fy"
		if i == 17 {
			path = "std/vec.fy"
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, text), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			return fmt.Errorf("source %d syntax: %v", i, err)
		}
		encoded, err := json.Marshal(parsed.AST)
		if err != nil {
			return err
		}
		field(text)
		field(encoded)
		var names []string
		for _, structure := range parsed.AST.Structs {
			if len(structure.TypeParameters) == 0 {
				names = append(names, structure.Name)
			}
		}
		for _, resource := range parsed.AST.Resources {
			if len(resource.TypeParameters) == 0 {
				names = append(names, resource.Name)
			}
		}
		number(len(names))
		for _, name := range names {
			digest := genericplan.DigestChunks("source-nominal-module-v0", encoded, []byte("\n"), []byte(name))
			field([]byte(name))
			field([]byte(digest.String()))
			definitions++
		}
		cases = append(cases, map[string]any{"case": i, "moduleBytes": len(encoded), "nominalDefinitions": len(names)})
	}
	dir := filepath.Join(".cache", "generic-source-modules-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "sources": len(inputs), "definitions": definitions, "fixtureBytes": out.Len(), "cases": cases, "sdk": map[string]any{"path": sdkPath, "sha256": fmt.Sprintf("%x", sha256.Sum256(sdk)), "bytes": len(sdk)}})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
