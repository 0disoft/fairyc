//go:build ignore

// Capture actual SDK Vec names and public signature types from the Go frontend.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func run(sdkPath, output string) error {
	authored, err := os.ReadFile("examples/lexer_candidate/tests/module_public_vector_types.fy")
	if err != nil {
		return err
	}
	literal := `("(?:\\.|[^"\\])*")`
	texts := []string{}
	for _, name := range []string{"root", "provider"} {
		m := regexp.MustCompile(`fn ref u8\[\] ` + name + `\(\)\{return b` + literal + `;\}`).FindSubmatch(authored)
		if len(m) != 2 {
			return fmt.Errorf("missing authored %s source", name)
		}
		text, e := strconv.Unquote(string(m[1]))
		if e != nil {
			return e
		}
		texts = append(texts, text)
	}
	sdk, err := os.ReadFile(sdkPath)
	if err != nil {
		return err
	}
	texts = append(texts, string(sdk))
	input := semantic.PackageInput{}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
	modules := []genericsource.PackageModule{}
	for i, text := range texts {
		id := hir.PackageID(i + 1)
		coord, e := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(fmt.Sprint(i+1), 64))
		if e != nil {
			return e
		}
		coordinates[id] = coord
		path := []string{"root.fy", "helper.fy", "std/vec.fy"}[i]
		parsed, e := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
		if e != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			return fmt.Errorf("syntax %s: %v %v", path, e, parsed.Tree.Diagnostics)
		}
		input.Files = append(input.Files, semantic.File{Package: id, Path: path, Module: parsed.AST})
		modules = append(modules, genericsource.PackageModule{Module: parsed.AST, Owner: genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(i+1), 64)), Imports: map[string]genericplan.PackageID{}})
	}
	input.Imports = []semantic.PackageImport{{FromPackage: 1, ImportPath: "helper", ToPackage: 2}, {FromPackage: 1, ImportPath: "std::vec", ToPackage: 3}}
	modules[0].Imports["helper"] = modules[1].Owner
	modules[0].Imports["std::vec"] = modules[2].Owner
	prepared, err := genericsource.PrepareSDKStoragePackages(modules, genericplan.DefaultPlanLimits(), []genericplan.PackageID{modules[2].Owner})
	if err != nil {
		return err
	}
	analysis, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
	if err != nil {
		return err
	}
	checked := analysis.Analysis.Semantic
	if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
		return fmt.Errorf("semantic rejection: %v %v", checked.Diagnostics, checked.Unsupported)
	}
	vectors := map[genericplan.DeclarationID]bool{}
	for _, d := range prepared.Declarations {
		if d.Kind == genericsource.OriginResource && d.Owner == modules[2].Owner && d.Module == "std::vec" && d.Name == "Vec" {
			vectors[d.ID] = true
		}
	}
	names := []string{}
	for _, o := range prepared.Origins {
		if vectors[o.Declaration] {
			names = append(names, "std::vec::"+o.GeneratedName)
		}
	}
	slices.Sort(names)
	if len(names) != 6 || len(slices.Compact(slices.Clone(names))) != 6 {
		return fmt.Errorf("expected six unique vector types, got %d", len(names))
	}
	descriptor, err := moduleinterface.Extract(checked.HIR, 1, coordinates)
	if err != nil {
		return err
	}
	parameters := []string{}
	for _, m := range descriptor.Modules {
		for _, item := range m.Items {
			if m.Path == "root" && item.Name == "expose" {
				parameters = append(parameters, item.Parameters...)
			}
		}
	}
	if len(parameters) != 4 {
		return fmt.Errorf("expected four public vector parameters")
	}
	var wire bytes.Buffer
	wire.WriteString("FYGVTYPE")
	number := func(n int) { _ = binary.Write(&wire, binary.BigEndian, uint64(n)) }
	field := func(raw []byte) { number(len(raw)); wire.Write(raw) }
	field(sdk)
	number(len(names))
	for _, n := range names {
		field([]byte(n))
	}
	number(len(parameters))
	for _, p := range parameters {
		field([]byte(p))
	}
	if wire.Len() > 1<<20 {
		return fmt.Errorf("fixture exceeds 1MiB")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, wire.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema": "fairyc.candidate-public-vector-types-oracle/v1", "passed": true, "vectors": names, "parameters": parameters, "flowCompleted": true, "fixture": output, "bytes": wire.Len()})
}
func main() {
	sdk := flag.String("sdk", "", "original authenticated SDK Vec source")
	output := flag.String("output", "", "bounded fixture output")
	flag.Parse()
	if *sdk == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "explicit sdk/output required")
		os.Exit(2)
	}
	if e := run(*sdk, *output); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
