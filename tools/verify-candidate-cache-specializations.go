//go:build ignore

// Publish actual Go specialization membership for authored source graphs.
// Fairy must derive these IDs only from its complete checked IR owner.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
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

func run(sdkPath, output string) error {
	authored, err := os.ReadFile("examples/lexer_candidate/tests/project_cache_specializations.fy")
	if err != nil {
		return err
	}
	literal := `("(?:\\.|[^"\\])*")`
	rootLine := regexp.MustCompile(`(?m)^fn ref u8\[\] root\(usize mode\)\{[^\r\n]+`).Find(authored)
	if rootLine == nil {
		return fmt.Errorf("missing authored root")
	}
	read := func(data []byte, pattern string) (string, error) {
		m := regexp.MustCompile(pattern).FindSubmatch(data)
		if len(m) != 2 {
			return "", fmt.Errorf("missing authored %s", pattern)
		}
		return strconv.Unquote(string(m[1]))
	}
	all := regexp.MustCompile(`return b`+literal+`;`).FindAllSubmatch(rootLine, -1)
	if len(all) != 4 {
		return fmt.Errorf("expected four root alternatives")
	}
	fallback, err := strconv.Unquote(string(all[len(all)-1][1]))
	if err != nil {
		return err
	}
	helper, err := read(authored, `fn ref u8\[\] provider\(\)\{return b`+literal+`;\}`)
	if err != nil {
		return err
	}
	sdk, err := os.ReadFile(sdkPath)
	if err != nil {
		return err
	}
	var wire bytes.Buffer
	wire.WriteString("FYCSPC01")
	number := func(n int) { binary.Write(&wire, binary.BigEndian, uint64(n)) }
	field := func(data []byte) { number(len(data)); wire.Write(data) }
	number(5)
	var observations []map[string]any
	for mode := 0; mode < 5; mode++ {
		root := fallback
		if mode == 0 || mode == 1 || mode == 3 {
			root, err = read(rootLine, `if\(mode==`+strconv.Itoa(mode)+`\)\{return b`+literal+`;\}`)
			if err != nil {
				return err
			}
		}
		texts := []string{root, helper}
		owners := []int{1, 2}
		if mode >= 2 {
			texts = append(texts, string(sdk))
			owners = append(owners, 3)
		}
		if mode == 4 {
			slices.Reverse(texts)
			slices.Reverse(owners)
		}
		coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
		input := semantic.PackageInput{}
		providers := map[string]int{}
		for i, text := range texts {
			m := regexp.MustCompile(`^module\s+([a-zA-Z0-9_:]+);`).FindStringSubmatch(text)
			if len(m) != 2 {
				return fmt.Errorf("missing module path")
			}
			path := strings.ReplaceAll(m[1], "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				return fmt.Errorf("syntax case %d %s: %v", mode, path, err)
			}
			id := hir.PackageID(owners[i])
			input.Files = append(input.Files, semantic.File{Package: id, Path: path, Module: parsed.AST})
			providers[m[1]] = i
			coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(strconv.Itoa(owners[i]), 64))
			if err != nil {
				return err
			}
			coordinates[id] = coordinate
		}
		label := fmt.Sprintf("case-%d", mode)
		field([]byte(label))
		number(len(texts))
		for i, file := range input.Files {
			number(owners[i])
			field([]byte(texts[i]))
			number(len(file.Module.Imports))
			for _, imported := range file.Module.Imports {
				target, found := providers[imported.Path]
				if !found {
					return fmt.Errorf("missing provider")
				}
				field([]byte(imported.Alias))
				number(target)
				binding := semantic.PackageImport{FromPackage: file.Package, ImportPath: imported.Path, ToPackage: input.Files[target].Package}
				if !slices.Contains(input.Imports, binding) {
					input.Imports = append(input.Imports, binding)
				}
			}
		}
		var analysis compiler.GenericAnalysis
		if mode >= 2 {
			analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
		} else {
			analysis, err = compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		}
		if err != nil {
			return err
		}
		checked := analysis.Analysis.Semantic
		if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
			return fmt.Errorf("semantic rejection case %d: %v %v", mode, checked.Diagnostics, checked.Unsupported)
		}
		count := 2
		if mode >= 2 {
			count = 3
		}
		number(count)
		for owner := 1; owner <= count; owner++ {
			ids := analysis.Specializations[hir.PackageID(owner)]
			if !slices.IsSorted(ids) || len(ids) != len(slices.Compact(slices.Clone(ids))) {
				return fmt.Errorf("noncanonical Go IDs")
			}
			number(owner)
			number(len(ids))
			for _, id := range ids {
				field([]byte(id))
			}
			observations = append(observations, map[string]any{"case": mode, "owner": owner, "count": len(ids), "ids": ids})
		}
	}
	if wire.Len() > 1<<20 {
		return fmt.Errorf("fixture exceeds 1MiB")
	}
	if err := os.WriteFile(output, wire.Bytes(), 0600); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(observations)
}
func main() {
	sdk := flag.String("sdk", "", "exact original SDK Vec source")
	output := flag.String("output", "", "bounded binary fixture")
	flag.Parse()
	if *sdk == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "explicit sdk and output required")
		os.Exit(2)
	}
	if err := run(*sdk, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
