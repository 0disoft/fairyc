package atlasmap

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestGenericLogicalPlanPreservesOriginalsAndConcreteFacts(t *testing.T) {
	for _, used := range []bool{false, true} {
		t.Run(map[bool]string{false: "unused", true: "specialized"}[used], func(t *testing.T) {
			workspace, result := loadGenericPlanWorkspace(t, used)
			if _, _, err := PrepareWorkspace(workspace, result, testCompilerBuild()); err == nil {
				t.Fatal("v1 accepted generic facts")
			}
			snapshot, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
			if err != nil || len(plans) != 1 {
				t.Fatalf("v2 plans: %d, %v", len(plans), err)
			}
			_, again, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
			if err != nil || !reflect.DeepEqual(plans, again) {
				t.Fatalf("plan changed across repeated construction: %v", err)
			}
			plan := plans[0]
			if plan.Header().LogicalSchema != LogicalSchemaV2 ||
				!reflect.DeepEqual(plan.Declarations(), snapshot.Declarations()) ||
				!reflect.DeepEqual(plan.Functions(), snapshot.Functions()) ||
				!reflect.DeepEqual(plan.Specializations(), snapshot.Specializations()) ||
				!reflect.DeepEqual(plan.Completeness(), snapshot.Completeness()) {
				t.Fatal("partition lost snapshot facts")
			}
			if len(plan.Declarations()) != 4 || len(plan.Functions()) != 1 {
				t.Fatal("source declarations were replaced by implementations")
			}
			want := 0
			if used {
				want = 4
			}
			if len(plan.Specializations()) != want || plan.Header().Counts.Specializations != want {
				t.Fatalf("specialization count: %+v", plan.Header().Counts)
			}
			relations := countRelations(plan.Functions())
			byID := map[string]semanticsnapshot.SpecializationSummary{}
			for _, item := range plan.Specializations() {
				byID[item.ID] = item
				relations += relationCount(item.Function)
				if item.Declaration != item.Function.Function || item.Declaration.PackageInstance != plan.Header().PackageInstance {
					t.Fatal("specialization lost declaration ownership")
				}
			}
			if plan.Header().Counts.Relations != relations {
				t.Fatal("specialization relations were not counted")
			}
			all := plan.Functions()
			for _, item := range plan.Specializations() {
				all = append(all, item.Function)
			}
			for _, function := range all {
				for _, call := range function.Calls {
					if call.Specialization != "" && byID[call.Specialization].Declaration != call.Target {
						t.Fatal("call lost its concrete implementation target")
					}
				}
			}
			if _, err := Encode(plan, 1<<20); err == nil {
				t.Fatal("v1 codec accepted a v2 plan")
			}
			declarations := plan.Declarations()
			for i := range declarations {
				if len(declarations[i].TypeParameters) > 0 {
					declarations[i].TypeParameters[0] = "changed"
				}
			}
			if !reflect.DeepEqual(plan.Declarations(), snapshot.Declarations()) {
				t.Fatal("declaration accessor aliases type parameter names")
			}
			items := plan.Specializations()
			for i := range items {
				items[i].TypeArguments[0].Name = "changed"
				if len(items[i].Function.Calls) > 0 {
					items[i].Function.Calls[0].Specialization = "changed"
				}
				if len(items[i].Function.Borrows) > 0 {
					items[i].Function.Borrows = nil
				}
			}
			if !reflect.DeepEqual(plan.Specializations(), snapshot.Specializations()) {
				t.Fatal("specialization accessor aliases retained facts")
			}
		})
	}
}

func TestLogicalV2RetainsOrdinaryPackagePartitions(t *testing.T) {
	workspace, result := loadQuickstart(t)
	_, legacy, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	_, current, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil || len(current) != len(legacy) {
		t.Fatalf("v2: %v", err)
	}
	for i := range current {
		if current[i].Header().LogicalSchema != LogicalSchemaV2 {
			t.Fatal("missing v2 schema")
		}
		current[i].data.header.LogicalSchema = LogicalSchema
		if !reflect.DeepEqual(current[i], legacy[i]) {
			t.Fatal("ordinary package facts changed")
		}
	}
}

func loadGenericPlanWorkspace(t *testing.T, used bool) (workspaceloader.Workspace, compiler.Result) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `schema = "fairyc.manifest/v1"
[package]
uid = "test.atlas.generic-plan"
name = "generic-plan"
version = "1.0.0"
language = "fairyc-language/16"
source = "src"
`
	source := `module main;
pub fn T id<T>(T value) { let ref T view = ref value; return *view; }
fn T pass<T>(T value) { return id<T>(value); }
fn T unused<T>(T value) { return value; }
`
	if used {
		source += "fn i32 main() { assert(pass<i32>(42) == 42); assert(pass<u32>(7) == 7); return 0; }\n"
	} else {
		source += "fn i32 main() { return 0; }\n"
	}
	for path, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": source} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compiler: %v, %+v", err, result.Semantic)
	}
	return workspace, result
}
