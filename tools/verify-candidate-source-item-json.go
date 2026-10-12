// Compare retained top-level item JSON to the actual Go syntax-only AST.
// bun tools/run-go.ts run tools/verify-candidate-source-item-json.go
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
	authored, err := os.ReadFile("examples/lexer_candidate/tests/generic_source_item_json.fy")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	matches := pattern.FindAllSubmatch(authored, -1)
	if len(matches) != 13 {
		return fmt.Errorf("expected thirteen authored sources")
	}
	var out bytes.Buffer
	out.WriteString("FYASTIT1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	number(len(matches))
	var cases []map[string]any
	for i, match := range matches {
		index, err := strconv.Atoi(string(match[1]))
		if err != nil || index != i {
			return fmt.Errorf("noncontiguous inputs")
		}
		text, err := strconv.Unquote(string(match[2]))
		if err != nil {
			return err
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			return fmt.Errorf("source %d syntax: %v", i, err)
		}
		var value any
		var kind string
		module := parsed.AST
		if len(module.Imports) == 1 {
			value, kind = module.Imports[0], "import"
		}
		if len(module.Constants) == 1 {
			value, kind = module.Constants[0], "constant"
		}
		if len(module.StaticAssertions) == 1 {
			value, kind = module.StaticAssertions[0], "static-assert"
		}
		if len(module.States) == 1 {
			value, kind = module.States[0], "state"
		}
		if len(module.ForeignFunctions) == 1 {
			value, kind = module.ForeignFunctions[0], "foreign-function"
		}
		if value == nil {
			return fmt.Errorf("source %d item selection", i)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		field([]byte(text))
		field(encoded)
		cases = append(cases, map[string]any{"case": i, "kind": kind, "itemBytes": len(encoded)})
	}
	dir := filepath.Join(".cache", "generic-source-items-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "items": len(matches), "fixtureBytes": out.Len(), "cases": cases})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
