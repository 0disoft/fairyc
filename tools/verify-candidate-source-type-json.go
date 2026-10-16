//go:build ignore

// Build the source-type identity fixture using actual Go frontend AST values.
// bun tools/run-go.ts run tools/verify-candidate-source-type-json.go
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

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	test, err := os.ReadFile("examples/lexer_candidate/tests/generic_source_type_json.fy")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	var inputs []string
	for _, match := range pattern.FindAllSubmatch(test, -1) {
		i, err := strconv.Atoi(string(match[1]))
		if err != nil || i != len(inputs) {
			return fmt.Errorf("noncontiguous source inputs")
		}
		value, err := strconv.Unquote(string(match[2]))
		if err != nil {
			return err
		}
		inputs = append(inputs, value)
	}
	last := regexp.MustCompile(`return b("(?:\\.|[^"\\])*");\n\}`).FindSubmatch(test)
	if len(inputs) != 5 || len(last) != 2 {
		return fmt.Errorf("expected six authored sources")
	}
	value, err := strconv.Unquote(string(last[1]))
	if err != nil {
		return err
	}
	inputs = append(inputs, value)
	var out bytes.Buffer
	out.WriteString("FYASTYP1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	number(len(inputs))
	types := 0
	for i, text := range inputs {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			return fmt.Errorf("source %d syntax: %v %v", i, err, parsed.Tree.Diagnostics)
		}
		if len(parsed.AST.Functions) != 1 {
			return fmt.Errorf("source %d lacks one function", i)
		}
		fn := parsed.AST.Functions[0]
		values := []ast.TypeReference{fn.ResultType}
		for _, p := range fn.Parameters {
			values = append(values, p.Type)
		}
		field([]byte(text))
		number(len(values))
		for _, v := range values {
			encoded, err := json.Marshal(v)
			if err != nil {
				return err
			}
			field(encoded)
			types++
		}
	}
	dir := filepath.Join(".cache", "generic-source-types-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "cases": len(inputs), "types": types, "fixtureBytes": out.Len()})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
