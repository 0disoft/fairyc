//go:build ignore

// Run from the repository root with:
// bun tools/run-go.ts run tools/verify-candidate-public-interface-extract.go
// The authored Fairy tests own the input bytes and expected values. This oracle
// uses the real Go frontend and interface implementation, not a second encoder.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type oracleCase struct {
	Case       int    `json:"case"`
	Modules    int    `json:"modules"`
	Items      int    `json:"items"`
	Bytes      int    `json:"bytes"`
	EncodedHex string `json:"encodedHex"`
	Hash       string `json:"hash"`
}

func run() error {
	test, err := os.ReadFile("examples/lexer_candidate/tests/module_public_interface_extract.fy")
	if err != nil {
		return err
	}
	literal := `("(?:\\.|[^"\\])*")`
	inputs := regexp.MustCompile(`if\(mode==([0-9]+) && i==([0-9]+)\)\{return b`+literal+`;\}`).FindAllSubmatch(test, -1)
	moduleName := regexp.MustCompile(`^module\s+([a-zA-Z_][a-zA-Z_0-9]*(?:::[a-zA-Z_][a-zA-Z_0-9]*)*)\s*;`)
	golden := regexp.MustCompile(`fn ref u8\[\] expected_hex\(\)\{return b` + literal + `;\}`).FindSubmatch(test)
	hashes := regexp.MustCompile(`fyif5-sha256-[0-9a-f]{64}`).FindAll(test, -1)
	if len(golden) != 2 || len(hashes) != 2 {
		return fmt.Errorf("authored test lacks exact encoding/hash expectations")
	}
	expectedHex, err := strconv.Unquote(string(golden[1]))
	if err != nil {
		return err
	}
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{
		PackageUID: "example.public-extraction", VersionIdentity: "1.0.0",
	})
	if err != nil {
		return err
	}
	var cases []oracleCase
	for mode := 0; mode < 3; mode++ {
		var files []semantic.File
		for _, input := range inputs {
			if string(input[1]) != strconv.Itoa(mode) {
				continue
			}
			if string(input[2]) != strconv.Itoa(len(files)) {
				return fmt.Errorf("case %d source indexes are not contiguous", mode)
			}
			text, err := strconv.Unquote(string(input[3]))
			if err != nil {
				return err
			}
			name := moduleName.FindStringSubmatch(text)
			if len(name) != 2 {
				return fmt.Errorf("case %d fixture lacks module name", mode)
			}
			path := strings.ReplaceAll(name[1], "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				return fmt.Errorf("case %d %s syntax: %v %v", mode, path, err, parsed.Tree.Diagnostics)
			}
			files = append(files, semantic.File{Path: path, Module: parsed.AST})
		}
		if len(files) != 2 {
			return fmt.Errorf("case %d must contain two source modules", mode)
		}
		typed := semantic.Analyze(files)
		if typed.FailedPhase != semantic.PhaseNone || len(typed.Diagnostics) != 0 || len(typed.Unsupported) != 0 {
			return fmt.Errorf("case %d semantics: %v %v", mode, typed.Diagnostics, typed.Unsupported)
		}
		descriptor, err := moduleinterface.Extract(typed.HIR, hir.DefaultPackage,
			map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: coordinate})
		if err != nil {
			return err
		}
		encoded, err := moduleinterface.Encode(descriptor)
		if err != nil {
			return err
		}
		hash, err := moduleinterface.Compute(descriptor)
		if err != nil {
			return err
		}
		expectedHash := string(hashes[1])
		if mode == 2 {
			expectedHash = string(hashes[0])
		}
		if hash.String() != expectedHash || mode < 2 && hex.EncodeToString(encoded) != expectedHex {
			return fmt.Errorf("case %d differs from authored Fairy expectations", mode)
		}
		items := 0
		for _, module := range descriptor.Modules {
			items += len(module.Items)
		}
		cases = append(cases, oracleCase{mode, len(descriptor.Modules), items, len(encoded), hex.EncodeToString(encoded), hash.String()})
	}
	if cases[0].Hash != cases[1].Hash || cases[0].Hash == cases[2].Hash {
		return fmt.Errorf("interface stability/mutation expectations failed")
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Schema     string       `json:"schema"`
		Passed     bool         `json:"passed"`
		Coordinate string       `json:"coordinate"`
		Cases      []oracleCase `json:"cases"`
	}{"fairyc.candidate-public-interface-extract-oracle/v1", true, coordinate.String(), cases})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
