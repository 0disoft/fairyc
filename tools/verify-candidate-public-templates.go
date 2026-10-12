//go:build ignore

// Capture original public templates and compiler-published source interfaces.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type reader struct {
	data []byte
	at   int
}

func (r *reader) number() (int, error) {
	if len(r.data)-r.at < 8 {
		return 0, fmt.Errorf("truncated number")
	}
	n := binary.BigEndian.Uint64(r.data[r.at : r.at+8])
	r.at += 8
	if n > 1<<20 {
		return 0, fmt.Errorf("fixture count exceeds bound")
	}
	return int(n), nil
}
func (r *reader) field() ([]byte, error) {
	n, err := r.number()
	if err != nil {
		return nil, err
	}
	if n > len(r.data)-r.at {
		return nil, fmt.Errorf("truncated field")
	}
	v := bytes.Clone(r.data[r.at : r.at+n])
	r.at += n
	return v, nil
}

type imported struct {
	alias  string
	target int
}
type module struct {
	digit   int
	raw     []byte
	imports []imported
}
type fixture struct {
	label   string
	modules []module
}

func readFixture(path string) ([]fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 16 || string(data[:8]) != "FYGDEP01" {
		return nil, fmt.Errorf("wrong fixture version")
	}
	r := reader{data: data, at: 8}
	count, err := r.number()
	if err != nil {
		return nil, err
	}
	var result []fixture
	for range count {
		label, err := r.field()
		if err != nil {
			return nil, err
		}
		n, err := r.number()
		if err != nil {
			return nil, err
		}
		f := fixture{label: string(label)}
		for range n {
			digit, err := r.number()
			if err != nil {
				return nil, err
			}
			raw, err := r.field()
			if err != nil {
				return nil, err
			}
			m := module{digit: digit, raw: raw}
			n, err := r.number()
			if err != nil {
				return nil, err
			}
			for range n {
				alias, err := r.field()
				if err != nil {
					return nil, err
				}
				target, err := r.number()
				if err != nil {
					return nil, err
				}
				m.imports = append(m.imports, imported{string(alias), target})
			}
			f.modules = append(f.modules, m)
		}
		n, err = r.number()
		if err != nil {
			return nil, err
		}
		for range n {
			for range 3 {
				if _, err := r.field(); err != nil {
					return nil, err
				}
			}
		}
		result = append(result, f)
	}
	if r.at != len(data) {
		return nil, fmt.Errorf("trailing fixture bytes")
	}
	if len(result) != 4 {
		return nil, fmt.Errorf("expected four original graphs")
	}
	base := result[0]
	multi := fixture{label: "MultiModule", modules: slices.Clone(base.modules)}
	multi.modules = append(multi.modules, module{digit: 1, raw: []byte("module root::child;pub fn T spare<T>(T value){return value;}pub fn i32 plain(){return 5;}")})
	result = append(result, multi)
	private := fixture{label: "NoPublic", modules: slices.Clone(base.modules)}
	private.modules[0].raw = bytes.ReplaceAll(bytes.ReplaceAll(base.modules[0].raw, []byte("pub fn T api"), []byte("fn T api")), []byte("pub fn T loop"), []byte("fn T loop"))
	result = append(result, private)
	return result, nil
}
func run(inputPath, output string) error {
	cases, err := readFixture(inputPath)
	if err != nil {
		return err
	}
	var wire bytes.Buffer
	wire.WriteString("FYGPUB01")
	number := func(n int) { _ = binary.Write(&wire, binary.BigEndian, uint64(n)) }
	field := func(b []byte) { number(len(b)); wire.Write(b) }
	number(len(cases))
	var reports []map[string]any
	for _, f := range cases {
		field([]byte(f.label))
		number(len(f.modules))
		inputs := make([]genericsource.PackageModule, len(f.modules))
		semanticInput := semantic.PackageInput{}
		coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
		var digits []int
		for i, m := range f.modules {
			if m.digit < 1 || m.digit > 3 {
				return fmt.Errorf("invalid owner digit")
			}
			header := bytes.IndexByte(m.raw, ';')
			if header < 0 {
				return fmt.Errorf("missing fixture module header")
			}
			path := strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(string(m.raw[:header]), "module")), "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, m.raw), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				return fmt.Errorf("%s syntax: %v %v", f.label, err, parsed.Tree.Diagnostics)
			}
			id := hir.PackageID(m.digit)
			coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(fmt.Sprint(m.digit), 64))
			if err != nil {
				return err
			}
			coordinates[id] = coordinate
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(m.digit), 64))
			inputs[i] = genericsource.PackageModule{Module: parsed.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}}
			semanticInput.Files = append(semanticInput.Files, semantic.File{Package: id, Path: strings.ReplaceAll(parsed.AST.Path, "::", "/") + ".fy", Module: parsed.AST})
			if !slices.Contains(digits, m.digit) {
				digits = append(digits, m.digit)
			}
		}
		seen := map[string]bool{}
		for i, m := range f.modules {
			number(m.digit)
			field(m.raw)
			number(len(m.imports))
			for _, binding := range m.imports {
				if binding.target < 0 || binding.target >= len(inputs) {
					return fmt.Errorf("bad import target")
				}
				target := inputs[binding.target]
				inputs[i].Imports[target.Module.Path] = target.Owner
				field([]byte(binding.alias))
				number(binding.target)
				key := fmt.Sprintf("%d/%s", m.digit, target.Module.Path)
				if !seen[key] {
					seen[key] = true
					semanticInput.Imports = append(semanticInput.Imports, semantic.PackageImport{FromPackage: hir.PackageID(m.digit), ImportPath: target.Module.Path, ToPackage: hir.PackageID(f.modules[binding.target].digit)})
				}
			}
		}
		sdkOwner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat("3", 64))
		prepared, err := genericsource.PrepareSDKStoragePackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{sdkOwner})
		if err != nil {
			return fmt.Errorf("%s prepare: %w", f.label, err)
		}
		analysis, err := compiler.AnalyzeSDKStoragePackages(semanticInput, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
		if err != nil {
			return fmt.Errorf("%s analyze: %w", f.label, err)
		}
		if analysis.Analysis.Semantic.FailedPhase != semantic.PhaseNone || len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 || len(analysis.Interfaces) != len(digits) {
			return fmt.Errorf("%s analysis failed: %v %v", f.label, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
		}
		slices.Sort(digits)
		number(len(digits))
		total := 0
		counts := map[int]int{}
		for _, digit := range digits {
			number(digit)
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(digit), 64))
			var templates []genericsource.PublicTemplate
			for _, p := range prepared.Templates {
				if p.Owner == owner {
					templates = append(templates, p)
				}
			}
			number(len(templates))
			for _, p := range templates {
				field([]byte(p.Module))
				field([]byte(p.Name))
				field([]byte(p.Closure.String()))
			}
			field([]byte(analysis.OrdinaryInterfaces[hir.PackageID(digit)].String()))
			field([]byte(analysis.Interfaces[hir.PackageID(digit)].String()))
			counts[digit] = len(templates)
			total += len(templates)
		}
		reports = append(reports, map[string]any{"case": f.label, "modules": len(inputs), "publicTemplates": total, "packageCounts": counts, "flowCompleted": analysis.Analysis.Semantic.CompletedThrough == semantic.PhaseFlow})
	}
	if wire.Len() > 131072 {
		return fmt.Errorf("fixture exceeds the authored reader bound")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, wire.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema": "fairyc.candidate-public-templates-oracle/v1", "passed": true, "fixture": output, "bytes": wire.Len(), "cases": reports, "source": "actual PrepareSDKStoragePackages public templates and compiler.AnalyzeSDKStoragePackages ordinary/generic interfaces after Flow"})
}
func main() {
	input := flag.String("input", "", "original Go dependency fixture")
	output := flag.String("output", "", "bounded public-template fixture output")
	flag.Parse()
	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "explicit input/output required")
		os.Exit(2)
	}
	if err := run(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
