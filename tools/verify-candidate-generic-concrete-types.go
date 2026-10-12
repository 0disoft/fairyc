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

func TestCandidateConcreteTypeOracle(t *testing.T) {
	repo, output := os.Getenv("FAIRYC_CONCRETE_TYPE_REPO"), os.Getenv("FAIRYC_CONCRETE_TYPE_OUTPUT")
	if repo == "" || output == "" {
		t.Fatal("explicit fixture paths required")
	}
	authored, err := os.ReadFile(filepath.Join(repo, "examples/lexer_candidate/tests/generic_concrete_types.fy"))
	if err != nil {
		t.Fatal(err)
	}
	values := map[int][]byte{}
	for _, match := range regexp.MustCompile(`if\(mode==([0-9]+)\)\{return b("(?:\\.|[^"\\])*");\}`).FindAllStringSubmatch(string(authored), -1) {
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
	wire.WriteString("FYGTYPE1")
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
		var actual *genericplan.Catalog
		_, plan, err := expandAll(expanders, genericplan.DefaultPlanLimits(), func(c *genericplan.Catalog) error { actual = c; return nil })
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
		if !slices.Equal(observerIDs, packageIDs) {
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
		field([]byte(expanders[0].nominalContext))
		ids := make([]genericplan.TypeID, 0, 13)
		for _, name := range scalarTypes {
			ids = append(ids, genericplan.TypeID(name))
		}
		for _, input := range inputs {
			names := []string{}
			for _, s := range input.Module.Structs {
				if len(s.TypeParameters) == 0 {
					names = append(names, s.Name)
				}
			}
			for _, r := range input.Module.Resources {
				if len(r.TypeParameters) == 0 {
					names = append(names, r.Name)
				}
			}
			for _, name := range names {
				ids = append(ids, genericplan.TypeID("source_nominal_v0::"+string(input.Owner)+"::"+input.Module.Path+"::"+name))
			}
		}
		slices.Sort(ids)
		number(len(ids))
		for _, id := range ids {
			row, ok := actual.Type(id)
			if !ok {
				t.Fatalf("missing actual descriptor %s", id)
			}
			field([]byte(row.ID))
			field([]byte(row.Owner))
			field([]byte(row.Definition.String()))
			number(len(row.Abilities))
			for _, ability := range row.Abilities {
				field([]byte(ability))
			}
		}
		leaf := genericplan.DeclarationID(string(inputs[tc.provider].Owner) + "::helper::id")
		found := false
		for _, s := range plan.Specializations {
			if s.Declaration == leaf && len(s.Bindings) == 1 && s.Bindings[0].Type == "i32" {
				if len(s.Dependencies) != 0 || len(s.Constraints) != 0 {
					t.Fatal("leaf requires dependencies")
				}
				field([]byte(s.Declaration))
				field([]byte(s.ID))
				found = true
				break
			}
		}
		if !found {
			t.Fatal("missing actual leaf specialization")
		}
		reports = append(reports, map[string]any{"case": tc.name, "modules": len(inputs), "types": len(ids), "nominals": len(ids) - 13, "specializations": len(plan.Specializations), "leafComparedToPackageAdapter": true})
	}
	if wire.Len() > 65536 {
		t.Fatal("fixture exceeds selected reader")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "oracle.bin"), wire.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	report := map[string]any{"schema": "fairyc.candidate-concrete-type-oracle/v1", "passed": true, "bytes": wire.Len(), "cases": reports, "source": "actual expandAll Catalog.Type and leaf Plan.Specializations; plan IDs crosschecked against PrepareSDKStoragePackages"}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "oracle.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d source graphs, %d actual descriptor bytes", len(cases), wire.Len())
}
