//go:build ignore

// Observe original-source locations from the actual Go generic source planner.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/source"
)

type binding struct {
	alias  string
	target int
}
type module struct {
	digit   int
	raw     string
	imports []binding
}
type fixture struct {
	label   string
	modules []module
	limits  genericplan.PlanLimits
}

func owner(digit int) genericplan.PackageID {
	return genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(digit), 64))
}
func run(output string) error {
	defaults := genericplan.DefaultPlanLimits()
	cycle := module{2, "module helper;pub fn T cycle<T>(T x){return secret<T>(x);}fn T secret<T>(T x){return cycle<T>(x);}", nil}
	z := module{1, "module zcaller;import helper as z;fn void later(){discard z::cycle<i32>(1);}", []binding{{"z", 1}}}
	a := module{1, "module acaller;import helper as z;import helper as a;fn void use(){discard a::cycle<i32>(1);discard z::cycle<i32>(2);}", []binding{{"z", 1}, {"a", 1}}}
	per := defaults
	per.MaxSpecializationsPerDeclaration = 1
	depth := defaults
	depth.MaxExpansionDepth = 1
	total := defaults
	total.MaxSpecializations = 1
	sdk, err := os.ReadFile("internal/sdkstdlib/sources/optional/std/vec.fy")
	if err != nil {
		return err
	}
	cases := []fixture{
		{"Direct", []module{{1, "module root;fn T cycle<T>(T x){return cycle<T>(x);}fn void use(){discard cycle<i32>(1);}", nil}}, defaults},
		{"Aliases", []module{z, cycle, a}, defaults},
		{"Reordered", []module{{2, cycle.raw, nil}, {1, a.raw, []binding{{"z", 0}, {"a", 0}}}, {1, z.raw, []binding{{"z", 0}}}}, defaults},
		{"OwnerOrder", []module{{2, a.raw, []binding{{"z", 1}, {"a", 1}}}, {3, cycle.raw, nil}, {1, z.raw, []binding{{"z", 1}}}}, defaults},
		{"PerDeclaration", []module{{1, "module root;fn T id<T>(T x){return x;}fn void use(){discard id<i32>(1);discard id<u8>(1);}", nil}}, per},
		{"Depth", []module{{1, "module root;import helper as h;fn T outer<T>(T x){return h::id<T>(x);}fn void use(){discard outer<i32>(1);}", []binding{{"h", 1}}}, {2, "module helper;pub fn T id<T>(T x){return x;}", nil}}, depth},
		{"Vec", []module{{1, "module root;import std::vec as vec;struct Leaf{i32 n;}fn vec::Vec<Leaf> make(){return vec::empty<Leaf>();}", []binding{{"vec", 1}}}, {3, string(sdk), nil}}, total},
	}
	var wire bytes.Buffer
	number := func(n int) { _ = binary.Write(&wire, binary.BigEndian, uint64(n)) }
	field := func(raw []byte) { number(len(raw)); wire.Write(raw) }
	wire.WriteString("FYGRLOC1")
	number(len(cases))
	var reports []map[string]any
	for _, c := range cases {
		inputs := make([]genericsource.PackageModule, len(c.modules))
		for i, m := range c.modules {
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(strings.SplitN(m.raw, ";", 2)[0], "module")), "::", "/")+".fy", []byte(m.raw)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				return fmt.Errorf("%s syntax: %v %v", c.label, err, parsed.Tree.Diagnostics)
			}
			inputs[i] = genericsource.PackageModule{Module: parsed.AST, Owner: owner(m.digit), Imports: map[string]genericplan.PackageID{}}
		}
		for i, m := range c.modules {
			for _, b := range m.imports {
				target := inputs[b.target]
				inputs[i].Imports[target.Module.Path] = target.Owner
			}
		}
		var sdkOwners []genericplan.PackageID
		for _, m := range inputs {
			if m.Module.Path == "std::vec" {
				sdkOwners = append(sdkOwners, m.Owner)
			}
		}
		var err error
		if len(sdkOwners) == 0 {
			_, err = genericsource.PreparePackages(inputs, c.limits)
		} else {
			_, err = genericsource.PrepareSDKStoragePackages(inputs, c.limits, sdkOwners)
		}
		var located *genericsource.SourceError
		if !errors.As(err, &located) || located.Kind != genericsource.FailureExpansion {
			return fmt.Errorf("%s missing located expansion failure: %v", c.label, err)
		}
		code := genericplan.ErrorCode(err)
		numeric := 2006
		if code == genericplan.CodeSpecializationCycle {
			numeric = 2005
		} else if code != genericplan.CodePlanningBudgetExceeded {
			return fmt.Errorf("%s unexpected code: %v", c.label, err)
		}
		found := -1
		for i, m := range inputs {
			if m.Owner == located.Owner && m.Module.Path == located.Module {
				if found != -1 {
					return fmt.Errorf("ambiguous source")
				}
				found = i
			}
		}
		if found < 0 || !located.Span.Valid(len(c.modules[found].raw)) {
			return fmt.Errorf("%s invalid location", c.label)
		}
		field([]byte(c.label))
		number(c.limits.MaxSpecializations)
		number(c.limits.MaxSpecializationsPerDeclaration)
		number(c.limits.MaxExpansionDepth)
		number(len(c.modules))
		for _, m := range c.modules {
			number(m.digit)
			field([]byte(m.raw))
			number(len(m.imports))
			for _, b := range m.imports {
				field([]byte(b.alias))
				number(b.target)
			}
		}
		number(numeric)
		number(found)
		number(located.Span.StartByte)
		number(located.Span.EndByte)
		reports = append(reports, map[string]any{"case": c.label, "code": code, "module": located.Module, "owner": located.Owner, "moduleIndex": found, "start": located.Span.StartByte, "end": located.Span.EndByte, "excerpt": c.modules[found].raw[located.Span.StartByte:located.Span.EndByte]})
	}
	if wire.Len() > 131072 {
		return fmt.Errorf("fixture too large")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, wire.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"schema": "fairyc.candidate-source-location-oracle/v1", "passed": true, "cases": reports, "fixture": output, "bytes": wire.Len(), "source": "actual PreparePackages/PrepareSDKStoragePackages located RequestError failures"})
}
func main() {
	output := flag.String("output", "", "binary oracle fixture output")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "explicit output required")
		os.Exit(2)
	}
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
