//go:build ignore

// Verify original source function metadata against the actual Go frontend AST.
// bun tools/run-go.ts run tools/verify-candidate-source-declaration-json.go
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	test, err := os.ReadFile("examples/lexer_candidate/tests/generic_source_declaration_json.fy")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	var out bytes.Buffer
	out.WriteString("FYASTHD1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	inputs := pattern.FindAllSubmatch(test, -1)
	if len(inputs) != 11 {
		return fmt.Errorf("expected eleven authored inputs")
	}
	number(len(inputs))
	attached := 0
	var cases []map[string]any
	for i, match := range inputs {
		index, err := strconv.Atoi(string(match[1]))
		if err != nil || index != i {
			return fmt.Errorf("noncontiguous authored inputs")
		}
		text, err := strconv.Unquote(string(match[2]))
		if err != nil {
			return err
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			return fmt.Errorf("source %d syntax: %v %v", i, err, parsed.Tree.Diagnostics)
		}
		if len(parsed.AST.Functions) != 1 {
			return fmt.Errorf("source %d lacks one function", i)
		}
		fn := parsed.AST.Functions[0]
		fn.Body = nil
		encoded, err := json.Marshal(fn)
		if err != nil {
			return err
		}
		suffix := []byte(`,"Body":null}`)
		if !bytes.HasSuffix(encoded, suffix) {
			return fmt.Errorf("function JSON tail changed")
		}
		header := append(bytes.TrimSuffix(encoded, suffix), '}')
		field([]byte(text))
		field(header)
		number(fn.DocumentationSpan.StartByte)
		number(fn.DocumentationSpan.EndByte)
		if fn.DocumentationSpan.EndByte > fn.DocumentationSpan.StartByte {
			attached++
		}
		cases = append(cases, map[string]any{"case": i, "headerBytes": len(header), "documentation": fn.DocumentationSpan})
	}
	dir := filepath.Join(".cache", "generic-source-declarations-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "cases": cases, "attached": attached, "fixtureBytes": out.Len()})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
