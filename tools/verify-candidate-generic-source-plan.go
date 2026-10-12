//go:build ignore

// Overlaid into genericsource to observe the real immutable source planner catalog.
package genericsource

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func TestCandidateSourcePlanOracle(t *testing.T) {
	repo, output := os.Getenv("FAIRYC_SOURCE_PLAN_REPO"), os.Getenv("FAIRYC_SOURCE_PLAN_OUTPUT")
	if repo == "" || output == "" {
		t.Fatal("explicit fixture paths required")
	}
	authored, err := os.ReadFile(filepath.Join(repo, "examples/lexer_candidate/tests/generic_source_plan.fy"))
	if err != nil {
		t.Fatal(err)
	}
	values := map[int][]byte{}
	start := bytes.Index(authored, []byte("fn ref u8[] source(u32 mode){"))
	if start < 0 {
		t.Fatal("missing authored source function")
	}
	end := bytes.Index(authored[start:], []byte("\nreturn b\"\";\n}"))
	if end < 0 {
		t.Fatal("missing source function end")
	}
	sourceFixtures := authored[start : start+end]
	for _, match := range regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`).FindAllStringSubmatch(string(sourceFixtures), -1) {
		id, _ := strconv.Atoi(match[1])
		value, err := strconv.Unquote(match[2])
		if err != nil {
			t.Fatal(err)
		}
		values[id] = []byte(value)
	}
	for i := 0; i < 3; i++ {
		if values[i] == nil {
			t.Fatalf("missing authored source %d", i)
		}
	}
	vec, err := os.ReadFile(filepath.Join(repo, "internal/sdkstdlib/sources/optional/std/vec.fy"))
	if err != nil {
		t.Fatal(err)
	}
	type fixture struct {
		name     string
		raw      [][]byte
		digits   []int
		provider int
	}
	cases := []fixture{
		{"Base", [][]byte{values[0], values[1], vec}, []int{1, 2, 3}, 1},
		{"PrivateEdit", [][]byte{values[0], values[2], vec}, []int{1, 2, 3}, 1},
		{"Reordered", [][]byte{vec, values[0], values[1]}, []int{3, 1, 2}, 2},
		{"TwinProvider", [][]byte{values[0], values[1], vec, values[1]}, []int{1, 2, 3, 4}, 3},
	}
	var wire bytes.Buffer
	wire.WriteString("FYGPLAN1")
	number := func(n int) {
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(value []byte) { number(len(value)); wire.Write(value) }
	number(len(cases))
	var reports []map[string]any
	for _, tc := range cases {
		inputs := make([]PackageModule, len(tc.raw))
		expanders := make([]*expander, len(tc.raw))
		targets := make([][]int, len(tc.raw))
		sdkIndex := -1
		for i, raw := range tc.raw {
			header := bytes.IndexByte(raw, ';')
			if header < 0 {
				t.Fatal("missing header")
			}
			path := strings.TrimSpace(strings.TrimPrefix(string(raw[:header]), "module"))
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(strings.ReplaceAll(path, "::", "/")+".fy", raw), source.DefaultLimits())
			if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
				t.Fatalf("%s parse: %v", tc.name, err)
			}
			owner := genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(tc.digits[i]), 64))
			inputs[i] = PackageModule{Module: parsed.AST, Owner: owner, Imports: map[string]genericplan.PackageID{}}
			if path == "std::vec" {
				sdkIndex = i
			}
		}
		if sdkIndex < 0 {
			t.Fatal("missing SDK")
		}
		for i, input := range inputs {
			allowed := i == sdkIndex
			expanders[i] = &expander{module: input.Module, owner: input.Owner, templates: map[string]ast.Function{}, names: map[string]bool{}, imports: map[string]*expander{}, allowPublic: true, allowNominal: true, packageNominal: true, allowStorage: allowed, allowStorageTransfers: allowed, scalarTemplatesOnly: !allowed}
		}
		for i, input := range inputs {
			for _, imported := range input.Module.Imports {
				target := tc.provider
				if imported.Path == "std::vec" {
					target = sdkIndex
				}
				if inputs[target].Module.Path != imported.Path {
					t.Fatal("wrong import target")
				}
				targets[i] = append(targets[i], target)
				inputs[i].Imports[imported.Path] = inputs[target].Owner
				expanders[i].imports[imported.Alias] = expanders[target]
			}
			if err := expanders[i].activateSDKVectorLoops(); err != nil {
				t.Fatal(err)
			}
		}
		if err := bindNominalContext(expanders); err != nil {
			t.Fatal(err)
		}

		_, plan, err := expandAll(expanders, genericplan.DefaultPlanLimits())
		if err != nil {
			t.Fatalf("%s expand: %v", tc.name, err)
		}
		prepared, err := PrepareSDKStoragePackages(inputs, genericplan.DefaultPlanLimits(), []genericplan.PackageID{inputs[sdkIndex].Owner})
		if err != nil {
			t.Fatalf("%s real package adapter: %v", tc.name, err)
		}
		observerIDs, packageIDs := []string{}, []string{}
		for _, s := range plan.Specializations {
			observerIDs = append(observerIDs, string(s.ID))
		}
		for _, s := range prepared.Plan.Specializations {
			packageIDs = append(packageIDs, string(s.ID))
		}
		slices.Sort(observerIDs)
		slices.Sort(packageIDs)
		if !slices.Equal(observerIDs, packageIDs) || !slices.Equal(plan.Roots, prepared.Plan.Roots) {
			t.Fatal("observer plan differs from real package adapter")
		}
		field([]byte(tc.name))
		number(len(inputs))
		for i, input := range inputs {
			number(tc.digits[i])
			field(tc.raw[i])
			number(len(input.Module.Imports))
			for j, imported := range input.Module.Imports {
				field([]byte(imported.Alias))
				number(targets[i][j])
			}
		}
		number(len(plan.Roots))
		for _, id := range plan.Roots {
			field([]byte(id))
		}
		number(len(plan.Specializations))
		for _, spec := range plan.Specializations {
			field([]byte(spec.ID))
			field([]byte(spec.Declaration))
			args := []genericplan.TypeID{}
			for _, binding := range spec.Bindings {
				args = append(args, binding.Type)
			}
			field([]byte(requestKey(spec.Declaration, args)))
			number(len(spec.Bindings))
			for _, binding := range spec.Bindings {
				field([]byte(binding.Parameter))
				field([]byte(binding.Type))
			}
			if len(spec.Constraints) != 0 {
				t.Fatal("unexpected source capability constraints")
			}
			number(len(spec.Dependencies))
			for _, dependency := range spec.Dependencies {
				field([]byte(dependency))
			}
		}

		reports = append(reports, map[string]any{"case": tc.name, "modules": len(inputs), "roots": len(plan.Roots), "specializations": len(plan.Specializations), "rootsComparedToPackageAdapter": true})

	}
	failures := []map[string]any{}
	bad := []string{
		"module root;import helper as a;fn T cycle<T>(T value){return cycle<T>(value);}fn void use(){discard cycle<i32>(1);}",
		"module root;import helper as a;struct Token{i32 n;}fn T id<T>(T value){return value;}fn void use(){discard id<Token>(Token{.n=1});}",
		"module root;import helper as a;fn T outer<T>(T value){return a::id<T>(value);}fn void use(){discard outer<i32>(1);discard outer<i64>(2);}",
	}
	for mode := 0; mode < 6; mode++ {
		at := mode
		if at > 2 {
			at = 2
		}
		raws := []string{bad[at], "module helper;pub fn T id<T>(T value){return value;}fn T secret<T>(T value){return value;}", string(vec)}
		modules := make([]PackageModule, 3)
		for i, raw := range raws {
			path := []string{"root", "helper", "std::vec"}[i]
			result, err := frontend.AnalyzeSyntax(source.NewFile(strings.ReplaceAll(path, "::", "/")+".fy", []byte(raw)), source.DefaultLimits())
			if err != nil || result.AST == nil || !result.Tree.ErrorFree() {
				t.Fatalf("bad rejection fixture %d/%d: %v", mode, i, err)
			}
			modules[i] = PackageModule{Module: result.AST, Owner: genericplan.PackageID("package_coordinate_v1_" + strings.Repeat(fmt.Sprint(i+1), 64)), Imports: map[string]genericplan.PackageID{}}
		}
		modules[0].Imports["helper"] = modules[1].Owner
		limits := genericplan.DefaultPlanLimits()
		if mode == 2 {
			limits.MaxSpecializations = 3
		}
		if mode == 3 {
			limits.MaxSpecializationsPerDeclaration = 1
		}
		if mode == 4 {
			limits.MaxExpansionDepth = 1
		}
		if mode == 5 {
			limits.MaxSpecializations = 0
		}
		_, err := PrepareSDKStoragePackages(modules, limits, []genericplan.PackageID{modules[2].Owner})
		expected := genericplan.CodePlanningBudgetExceeded
		if mode == 0 {
			expected = genericplan.CodeSpecializationCycle
		}
		if mode == 1 {
			expected = genericplan.CodeAbilityUnsatisfied
		}
		if genericplan.ErrorCode(err) != expected {
			t.Fatalf("mode %d: %v", mode, err)
		}
		failures = append(failures, map[string]any{"mode": mode, "code": expected})
	}
	if wire.Len() > 1048576 {
		t.Fatal("fixture exceeds selected reader")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "oracle.bin"), wire.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"schema": "fairyc.candidate-source-plan-oracle/v1", "passed": true, "failureCategories": failures, "bytes": wire.Len(), "cases": reports, "source": "actual source Plan.Roots and ordered Specializations with bindings and dependencies; crosschecked against PrepareSDKStoragePackages"}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "oracle.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d source graphs, %d actual root bytes", len(cases), wire.Len())
}
