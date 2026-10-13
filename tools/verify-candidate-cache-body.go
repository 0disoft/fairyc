// Observe the existing Go HIR state-body hash for authored ordinary source.
// Oracle preimages are accepted only if they hash to that actual compiler value.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	if len(os.Args) != 3 || os.Args[1] != "--output" {
		return fmt.Errorf("usage: verify-candidate-cache-body.go --output <fixture>")
	}
	authored, err := os.ReadFile("examples/lexer_candidate/tests/project_cache_body_identity.fy")
	if err != nil {
		return err
	}
	literal := `("(?:\\.|[^"\\])*")`
	read := func(pattern string) (string, error) {
		m := regexp.MustCompile(pattern).FindSubmatch(authored)
		if len(m) != 2 {
			return "", fmt.Errorf("missing authored input %s", pattern)
		}
		return strconv.Unquote(string(m[1]))
	}
	fallback, err := read(`return b` + literal + `;\}\r?\n`)
	if err != nil {
		return err
	}
	other, err := read(`fn ref u8\[\] other\(\)\{return b` + literal + `;\}`)
	if err != nil {
		return err
	}
	var wire bytes.Buffer
	binary.Write(&wire, binary.BigEndian, uint32(8))
	var observations []map[string]any
	for mode := 0; mode < 8; mode++ {
		raw := fallback
		if mode == 0 || mode == 1 || mode == 3 || mode == 4 {
			raw, err = read(`if\(mode==` + strconv.Itoa(mode) + `\)\{return b` + literal + `;\}`)
			if err != nil {
				return err
			}
		}
		coord, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(map[bool]string{false: "1", true: "2"}[mode == 5], 64))
		if err != nil {
			return err
		}
		otherCoord, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat("3", 64))
		if err != nil {
			return err
		}
		coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: coord, 2: otherCoord}
		input := semantic.PackageInput{}
		texts := []string{raw}
		owners := []hir.PackageID{1}
		if mode == 6 {
			texts = append(texts, other)
			owners = append(owners, 2)
		}
		if mode == 7 {
			texts = []string{other, raw}
			owners = []hir.PackageID{2, 1}
		}
		for i, text := range texts {
			module := regexp.MustCompile(`^module\s+([a-zA-Z0-9_:]+);`).FindStringSubmatch(text)
			if len(module) != 2 {
				return fmt.Errorf("missing module path")
			}
			path := strings.ReplaceAll(module[1], "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				return fmt.Errorf("syntax case %d: %v; diagnostics=%v; source=%q", mode, err, parsed.Tree.Diagnostics, text)
			}
			input.Files = append(input.Files, semantic.File{Package: owners[i], Path: path, Module: parsed.AST})
		}
		analyzed, err := compiler.AnalyzePackages(input)
		if err != nil {
			return err
		}
		checked := analyzed.Semantic
		if checked.CompletedThrough != semantic.PhaseFlow || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 || len(checked.HIR.StateTransactions) != 0 {
			return fmt.Errorf("ordinary analysis case %d did not complete without transactions", mode)
		}
		hashes, err := contractstatemir.ComputeHIRStateBodyHashes(checked.HIR, []hir.PackageID{1}, coordinates)
		if err != nil {
			return err
		}
		layouts, err := contractstatemir.LowerHIRLayouts(checked.HIR)
		if err != nil {
			return err
		}
		var encoded bytes.Buffer
		encoded.WriteString("fairyc.contract-state-package-body")
		encoded.Write([]byte{0, 1})
		encoded.Write(coord[:])
		found := false
		for _, layout := range layouts {
			if layout.Package == 1 {
				if found {
					return fmt.Errorf("duplicate layout")
				}
				encoded.WriteByte(1)
				encoded.Write(layout.Layout.Hash[:])
				found = true
			}
		}
		if !found {
			encoded.WriteByte(0)
		}
		binary.Write(&encoded, binary.BigEndian, uint64(0))
		if sha256.Sum256(encoded.Bytes()) != [32]byte(hashes[1]) {
			return fmt.Errorf("preimage mismatch case %d", mode)
		}
		binary.Write(&wire, binary.BigEndian, uint32(encoded.Len()))
		wire.Write(encoded.Bytes())
		wire.WriteString(hashes[1].String())
		observations = append(observations, map[string]any{"case": mode, "bytes": encoded.Len(), "hash": hashes[1].String(), "layout": found, "transactions": len(checked.HIR.StateTransactions)})
	}
	if err := os.WriteFile(os.Args[2], wire.Bytes(), 0600); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(observations)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
