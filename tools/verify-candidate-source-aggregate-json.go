//go:build ignore

// Compare retained aggregate JSON and definition digests to the actual Go AST.
// bun tools/run-go.ts run tools/verify-candidate-source-aggregate-json.go
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

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	authored, err := os.ReadFile("examples/lexer_candidate/tests/generic_source_aggregate_json.fy")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	matches := pattern.FindAllSubmatch(authored, -1)
	if len(matches) != 12 {
		return fmt.Errorf("expected twelve authored sources")
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
	out.WriteString("FYASTAG1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	number(len(inputs))
	definitions := 0
	var cases []map[string]any
	for i, text := range inputs {
		path := "sample.fy"
		if i == 12 {
			path = "std/vec.fy"
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, text), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			return fmt.Errorf("source %d syntax: %v", i, err)
		}
		wrapper := struct {
			Struct   *ast.Struct
			Resource *ast.Resource
		}{}
		var value any
		if len(parsed.AST.Structs) == 1 {
			wrapper.Struct = &parsed.AST.Structs[0]
			value = wrapper.Struct
		}
		if len(parsed.AST.Resources) == 1 {
			wrapper.Resource = &parsed.AST.Resources[0]
			value = wrapper.Resource
		}
		if len(parsed.AST.Enums) == 1 {
			value = &parsed.AST.Enums[0]
		}
		if value == nil {
			return fmt.Errorf("source %d aggregate selection", i)
		}
		if i == 12 && (wrapper.Resource == nil || wrapper.Resource.Name != "Vec" || len(wrapper.Resource.Fields) != 6 || wrapper.Resource.Cleanup == nil) {
			return fmt.Errorf("SDK Vec source shape changed")
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var template []byte
		var digest string
		if wrapper.Struct != nil || wrapper.Resource != nil {
			template, err = json.Marshal(wrapper)
			if err != nil {
				return err
			}
			digest = genericplan.DigestText("source-aggregate-ast-v0", string(template)).String()
			definitions++
		}
		field(text)
		field(encoded)
		field(template)
		field([]byte(digest))
		cases = append(cases, map[string]any{"case": i, "aggregateBytes": len(encoded), "templateBytes": len(template), "definition": digest})
	}
	dir := filepath.Join(".cache", "generic-source-aggregates-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "aggregates": len(inputs), "definitions": definitions, "fixtureBytes": out.Len(), "cases": cases, "sdk": map[string]any{"path": sdkPath, "sha256": fmt.Sprintf("%x", sha256.Sum256(sdk)), "bytes": len(sdk)}})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
