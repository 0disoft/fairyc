//go:build ignore

// Compare exact source function JSON and the real Go template definition digest.
// bun tools/run-go.ts run tools/verify-candidate-source-function-definition.go
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
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	var out bytes.Buffer
	out.WriteString("FYASTDG1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(value []byte) { number(len(value)); out.Write(value) }
	number(38)
	count := 0
	var variants []string
	pattern := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	for _, entry := range []struct {
		name  string
		count int
	}{{"generic_source_function_json", 16}, {"generic_source_view_loop_json", 16}, {"generic_source_function_definition", 6}} {
		file, err := os.ReadFile("examples/lexer_candidate/tests/" + entry.name + ".fy")
		if err != nil {
			return err
		}
		inputs := pattern.FindAllSubmatch(file, -1)
		if len(inputs) != entry.count {
			return fmt.Errorf("%s fixture count", entry.name)
		}
		for i, match := range inputs {
			index, err := strconv.Atoi(string(match[1]))
			if err != nil || index != i {
				return fmt.Errorf("%s noncontiguous input", entry.name)
			}
			text, err := strconv.Unquote(string(match[2]))
			if err != nil {
				return err
			}
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() || len(parsed.AST.Functions) != 1 {
				return fmt.Errorf("%s source %d syntax: %v", entry.name, i, err)
			}
			encoded, err := json.Marshal(parsed.AST.Functions[0])
			if err != nil {
				return err
			}
			digest := genericplan.DigestText("source-function-ast-v0", string(encoded)).String()
			field([]byte(text))
			field(encoded)
			field([]byte(digest))
			count++
			if entry.name == "generic_source_function_definition" {
				variants = append(variants, digest)
			}
		}
	}
	if count != 38 || len(variants) != 6 {
		return fmt.Errorf("definition count")
	}
	for _, hash := range variants[1:] {
		if hash == variants[0] {
			return fmt.Errorf("mutation did not invalidate definition")
		}
	}
	dir := filepath.Join(".cache", "generic-source-function-definitions-20261013")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.bin"), out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "definitions": count, "mutationDigests": variants, "fixtureBytes": out.Len()})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
