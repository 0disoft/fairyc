//go:build ignore

// Verify the authored Region admission cases against actual Go package Flow.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func run() error {
	test, err := os.ReadFile("examples/lexer_candidate/tests/module_unit_region_demands.fy")
	if err != nil {
		return err
	}
	parts := strings.Split(string(test), "fn ref u8[] provider(u32 mode)")
	if len(parts) != 2 {
		return fmt.Errorf("missing authored provider fixtures")
	}
	expression := regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`)
	read := func(text string) (map[int]string, error) {
		out := map[int]string{}
		for _, m := range expression.FindAllStringSubmatch(text, -1) {
			n, _ := strconv.Atoi(m[1])
			raw, err := strconv.Unquote(m[2])
			if err != nil {
				return nil, err
			}
			out[n] = raw
		}
		return out, nil
	}
	mains, err := read(parts[0])
	if err != nil {
		return err
	}
	providers, err := read(parts[1])
	if err != nil {
		return err
	}
	allocator, err := os.ReadFile("internal/sdkstdlib/sources/std/alloc.fy")
	if err != nil {
		return err
	}
	var cases []map[string]any
	for mode := 0; mode <= 4; mode++ {
		raw, exists := mains[mode]
		if !exists {
			return fmt.Errorf("missing mode %d", mode)
		}
		raws := []string{raw}
		if mode >= 3 {
			raws = append(raws, providers[mode])
		}
		if mode >= 2 {
			raws = append(raws, string(allocator))
		}
		input := semantic.PackageInput{}
		coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
		owners := map[string]hir.PackageID{}
		for _, raw := range raws {
			header := strings.IndexByte(raw, ';')
			if header < 0 {
				return fmt.Errorf("missing module header")
			}
			name := strings.TrimSpace(strings.TrimPrefix(raw[:header], "module"))
			path := strings.ReplaceAll(name, "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(raw)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				return fmt.Errorf("mode %d syntax: %v", mode, err)
			}
			id := hir.PackageID(len(input.Files) + 1)
			owners[name] = id
			input.Files = append(input.Files, semantic.File{Package: id, Path: path, Module: parsed.AST})
			coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: name, VersionIdentity: "1"})
			if err != nil {
				return err
			}
			coordinates[id] = coordinate
		}
		for _, file := range input.Files {
			for _, imported := range file.Module.Imports {
				input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: file.Package, ImportPath: imported.Path, ToPackage: owners[imported.Path]})
			}
		}
		analysis, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil {
			return fmt.Errorf("mode %d: %w", mode, err)
		}
		checked := analysis.Analysis.Semantic
		ready := checked.FailedPhase == semantic.PhaseNone && checked.CompletedThrough == semantic.PhaseFlow && len(checked.Diagnostics) == 0 && len(checked.Unsupported) == 0
		if ready != (mode != 1) {
			return fmt.Errorf("mode %d readiness %v: %v %v", mode, ready, checked.Diagnostics, checked.Unsupported)
		}
		cases = append(cases, map[string]any{"mode": mode, "flowReady": ready, "specializations": len(analysis.Plan.Specializations), "diagnostics": len(checked.Diagnostics), "unsupported": len(checked.Unsupported)})
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema": "fairyc.candidate-template-region-oracle/v1", "passed": true, "cases": cases, "source": "authored Fairy fixtures through compiler.AnalyzeGenericPackages and exact std::alloc source"})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
