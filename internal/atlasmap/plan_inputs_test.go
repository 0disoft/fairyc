package atlasmap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Compile and build the source snapshot outside the timed planner invocation.
func BenchmarkBuildPlanInputs(b *testing.B) {
	for _, count := range []int{64, 512} {
		for _, schema := range []string{LogicalSchema, LogicalSchemaV2, LogicalSchemaV3} {
			b.Run(fmt.Sprintf("declarations=%d/%s", count, schema), func(b *testing.B) {
				snapshot, plans := planInputsFixture(b, count, schema)
				input := planInputsFromPartitions(b, plans)
				b.ReportAllocs()
				for b.Loop() {
					got, err := buildPlan(snapshot, input, schema)
					if err != nil || len(got) != 1 || len(got[0].data.declarations) != count+1 {
						b.Fatalf("plan: packages=%d, error=%v", len(got), err)
					}
				}
			})
		}
	}
}

func TestBuildPlanInputReuseKeepsSnapshotsAndPlansDetached(t *testing.T) {
	for _, schema := range []string{LogicalSchema, LogicalSchemaV2, LogicalSchemaV3} {
		t.Run(schema, func(t *testing.T) {
			snapshot, plans := planInputsFixture(t, 4, schema)
			if schema == LogicalSchemaV3 {
				workspace, result := loadAggregateMapWorkspace(t, true)
				var err error
				snapshot, plans, err = PrepareWorkspaceV3(workspace, result, testCompilerBuild())
				if err != nil {
					t.Fatal(err)
				}
			}
			input := planInputsFromPartitions(t, plans)
			encode := Encode
			if schema == LogicalSchemaV2 {
				encode = EncodeV2
			} else if schema == LogicalSchemaV3 {
				encode = EncodeV3
			}
			original, err := encode(plans[0], testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			other, err := buildPlan(snapshot, input, schema)
			if err != nil {
				t.Fatal(err)
			}
			// Mutate nested storage directly to test planner ownership, independently
			// of the public Partition getters' own defensive copies.
			for i := range plans[0].data.declarations {
				declaration := &plans[0].data.declarations[i]
				if len(declaration.Parameters) > 0 {
					declaration.Parameters[0].Type.Name = "changed"
				}
				if len(declaration.Fields) > 0 {
					declaration.Fields[0].Type.Name = "changed"
				}
			}
			for i := range plans[0].data.specializations {
				item := &plans[0].data.specializations[i]
				item.TypeArguments[0].Name = "changed"
				if len(item.Function.Borrows) > 0 {
					item.Function.Borrows[0].Kind = "changed"
				}
				if item.Aggregate != nil {
					item.Aggregate.Fields[0].Type.Name = "changed"
				}
				if item.Cleanup != nil {
					item.Cleanup.Calls[0].Specialization = "changed"
				}
			}
			fresh, err := buildPlan(snapshot, input, schema)
			if err != nil {
				t.Fatal(err)
			}
			for _, partition := range []Partition{other[0], fresh[0]} {
				encoded, err := encode(partition, testMapLimit)
				if err != nil || !bytes.Equal(original, encoded) {
					t.Fatalf("planner output aliases the snapshot or another plan: %v", err)
				}
			}
		})
	}
}

func planInputsFromPartitions(tb testing.TB, plans []Partition) Input {
	tb.Helper()
	input := Input{CompilerBuild: testCompilerBuild(), Interfaces: map[string]moduleinterface.Hash{}, GenericInterfaces: map[string]moduleinterface.GenericHash{}}
	for _, plan := range plans {
		header := plan.Header()
		hash, err := moduleinterface.ParseHash(header.InterfaceHash)
		if err != nil {
			tb.Fatal(err)
		}
		input.Interfaces[header.PackageInstance] = hash
		if header.GenericInterfaceHash != "" {
			hash, err := moduleinterface.ParseGenericHash(header.GenericInterfaceHash)
			if err != nil {
				tb.Fatal(err)
			}
			input.GenericInterfaces[header.PackageInstance] = hash
		}
	}
	return input
}

func planInputsFixture(tb testing.TB, count int, schema string) (semanticsnapshot.Snapshot, []Partition) {
	tb.Helper()
	root := tb.TempDir()
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.plan-inputs\"\nname = \"plan-inputs\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	var declarations, body strings.Builder
	body.WriteString("fn i32 main() {")
	for i := 0; i < count; i++ {
		if schema == LogicalSchema {
			fmt.Fprintf(&declarations, "fn i32 f_%d(i32 value) { return value; }\n", i)
			fmt.Fprintf(&body, "discard f_%d(42);", i)
		} else {
			fmt.Fprintf(&declarations, "fn T f_%d<T>(T value) { let ref T view = ref value; return *view; }\n", i)
			fmt.Fprintf(&body, "discard f_%d<i32>(42);", i)
		}
	}
	body.WriteString("return 0;}")
	for name, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": "module main;\n" + declarations.String() + body.String()} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			tb.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		tb.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		tb.Fatal(err)
	}
	var result compiler.Result
	if schema == LogicalSchema {
		result, err = compiler.CheckPackages(input)
	} else {
		coordinates, coordinateErr := workspace.PackageCoordinates()
		if coordinateErr != nil {
			tb.Fatal(coordinateErr)
		}
		result, _, err = compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	}
	if err != nil || !result.Verified {
		tb.Fatalf("compile: verified=%t, error=%v", result.Verified, err)
	}
	snapshot, plans, err := prepareWorkspace(workspace, result, testCompilerBuild(), schema)
	if err != nil || len(plans) != 1 {
		tb.Fatalf("prepare: packages=%d, error=%v", len(plans), err)
	}
	return snapshot, plans
}
