// Verify original source expressions against the actual Go frontend AST.
// bun tools/run-go.ts run tools/verify-candidate-source-expression-json.go
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
	test, err := os.ReadFile("examples/lexer_candidate/tests/generic_source_expression_json.fy")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	var out bytes.Buffer
	out.WriteString("FYASTEX1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	inputs := pattern.FindAllSubmatch(test, -1)
	if len(inputs) != 16 {
		return fmt.Errorf("expected sixteen authored inputs")
	}
	number(len(inputs))
	nodes := 0
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
		if fn.Body == nil || len(fn.Body.Statements) != 1 || fn.Body.Statements[0].Expression == nil {
			return fmt.Errorf("source %d lacks return expression", i)
		}
		expression := fn.Body.Statements[0].Expression
		encoded, err := json.Marshal(expression)
		if err != nil {
			return err
		}
		field([]byte(text))
		field(encoded)
		var walk func(*ast.Expression)
		walk = func(x *ast.Expression) {
			if x == nil {
				return
			}
			nodes++
			for _, c := range x.Children {
				walk(c)
			}
		}
		walk(expression)
		cases = append(cases, map[string]any{"case": i, "expressionBytes": len(encoded)})
	}
	dir := filepath.Join(".cache", "generic-source-expressions-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "cases": cases, "expressionNodes": nodes, "fixtureBytes": out.Len()})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
