// Run with bun tools/run-go.ts run tools/verify-candidate-generic-interface.go.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"os"
	"regexp"
	"strconv"
)

func run() error {
	source, e := os.ReadFile("examples/lexer_candidate/tests/module_generic_interface.fy")
	if e != nil {
		return e
	}
	literal := `("(?:\\.|[^"\\])*")`
	bodyPattern := regexp.MustCompile(`if\(i==([0-9]+)\)\{return b` + literal + `;\}`)
	var closures []genericplan.Digest
	bodySource := regexp.MustCompile(`fn ref u8\[\] body\(usize i\)\{[^\n]*`).Find(source)
	for _, m := range bodyPattern.FindAllSubmatch(bodySource, -1) {
		i, e := strconv.Atoi(string(m[1]))
		if e != nil || i != len(closures) {
			return fmt.Errorf("noncontiguous body fixture")
		}
		body, e := strconv.Unquote(string(m[2]))
		if e != nil {
			return e
		}
		closures = append(closures, genericplan.DigestText("test-template", body))
	}
	if len(closures) != 4 {
		return fmt.Errorf("expected four authored template definitions")
	}
	rows := []map[string]any{}
	for mode := 0; mode < 5; mode++ {
		d := moduleinterface.Descriptor{Modules: []moduleinterface.Module{{Path: "alpha"}, {Path: "util", Items: []moduleinterface.Item{{Kind: "function", Name: "plain", ResultType: "i32", MustUse: mode == 2}}}}}
		templates := []moduleinterface.GenericTemplate{{Module: "util", Name: "b", Closure: closures[0]}, {Module: "util", Name: "a", Closure: closures[1]}, {Module: "alpha", Name: "c", Closure: closures[2]}}
		if mode == 1 {
			templates[0], templates[2] = templates[2], templates[0]
		}
		if mode == 3 {
			templates[1].Closure = closures[3]
		}
		if mode == 4 {
			templates = nil
		}
		id, e := moduleinterface.ComputeGeneric(d, templates)
		if e != nil {
			return e
		}
		row := map[string]any{"mode": mode, "id": id.String()}
		if mode == 0 {
			base, e := moduleinterface.Compute(d)
			if e != nil {
				return e
			}
			var out bytes.Buffer
			field := func(s string) { binary.Write(&out, binary.BigEndian, uint64(len(s))); out.WriteString(s) }
			field(moduleinterface.GenericHashEncoding)
			field(moduleinterface.HashEncoding)
			out.Write(base[:])
			binary.Write(&out, binary.BigEndian, uint64(3))
			for _, t := range []moduleinterface.GenericTemplate{templates[2], templates[1], templates[0]} {
				field(t.Module)
				field(t.Name)
				out.Write(t.Closure[:])
			}
			row["encodedHex"] = hex.EncodeToString(out.Bytes())
		}
		if len(os.Args) == 1 {
			pattern := `if\(mode==` + strconv.Itoa(mode) + `\)\{return b` + literal + `;\}`
			matches := regexp.MustCompile(pattern).FindAllSubmatch(source, -1)
			found := false
			for _, m := range matches {
				value, _ := strconv.Unquote(string(m[1]))
				if value == id.String() {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("missing/mismatched golden hash %d", mode)
			}
			if mode == 0 {
				m := regexp.MustCompile(`fn ref u8\[\] golden_hex\(\)\{return b` + literal + `;\}`).FindSubmatch(source)
				if len(m) != 2 {
					return fmt.Errorf("missing golden bytes")
				}
				value, e := strconv.Unquote(string(m[1]))
				if e != nil || value != row["encodedHex"] {
					return fmt.Errorf("golden bytes mismatch")
				}
			}
		} else if len(os.Args) != 2 || os.Args[1] != "--goldens" {
			return fmt.Errorf("usage: oracle [--goldens]")
		}
		rows = append(rows, row)
	}
	definitions := []string{}
	for _, d := range closures {
		definitions = append(definitions, d.String())
		if len(os.Args) == 1 && !bytes.Contains(source, []byte(d.String())) {
			return fmt.Errorf("definition golden missing")
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"closures": definitions, "cases": rows, "passed": true})
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
