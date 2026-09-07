package atlasmap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestGenericSourceInterfacesBindMapAndReader(t *testing.T) {
	for _, used := range []bool{false, true} {
		workspace, result := loadGenericPlanWorkspace(t, used)
		snapshot, plans, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		expectations, err := CurrentWorkspaceExpectationsV2(workspace, result, testCompilerBuild())
		if err != nil {
			t.Fatal(err)
		}
		facts := result.GenericProvenance.PackageInterfaces()
		for _, plan := range plans {
			header := plan.Header()
			if header.GenericInterfaceAlgorithm != moduleinterface.GenericHashEncoding || header.GenericInterfaceHash != facts[0].Generic.String() || header.InterfaceHash != facts[0].Ordinary.String() {
				t.Fatal("map header does not retain original source interfaces")
			}
			encoded, err := EncodeV2(plan, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeV2(encoded, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			expectation := expectations[header.PackageInstance]
			if err := compatibilityError(decoded, expectation); err != nil {
				t.Fatal(err)
			}
			expectation.GenericInterface[0] ^= 1
			if err := compatibilityError(decoded, expectation); !errors.Is(err, ErrIncompatibleMap) {
				t.Fatalf("changed generic interface: %v", err)
			}
			expectation.GenericInterface = moduleinterface.GenericHash{}
			if err := compatibilityError(decoded, expectation); !errors.Is(err, ErrIncompatibleMap) {
				t.Fatalf("missing generic interface: %v", err)
			}
			for _, mutation := range []func(*Partition){
				func(p *Partition) {
					p.data.header.GenericInterfaceHash = ""
					p.data.header.GenericInterfaceAlgorithm = ""
				},
				func(p *Partition) { p.data.header.GenericInterfaceAlgorithm = "unknown" },
				func(p *Partition) { p.data.header.GenericInterfaceHash = (moduleinterface.GenericHash{}).String() },
			} {
				changed := plan
				mutation(&changed)
				if _, err := EncodeV2(changed, testMapLimit); err == nil {
					t.Fatal("invalid generic interface header accepted")
				}
			}
		}
		ordinary := map[string]moduleinterface.Hash{plans[0].Header().PackageInstance: facts[0].Ordinary}
		if _, err := BuildPlanV2(snapshot, Input{CompilerBuild: testCompilerBuild(), Interfaces: ordinary}); err == nil {
			t.Fatal("planner accepted missing generic interfaces")
		}
		result.GenericProvenance = compiler.GenericProvenance{}
		if _, _, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild()); err == nil {
			t.Fatal("planner accepted missing source provenance")
		}
		if _, err := CurrentWorkspaceExpectationsV2(workspace, result, testCompilerBuild()); err == nil {
			t.Fatal("reader accepted missing source provenance")
		}
	}
}

func TestGenericInterfacesRejectCoordinateAndPackageMismatch(t *testing.T) {
	workspace, result := loadGenericPlanWorkspace(t, true)
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	fact := result.GenericProvenance.PackageInterfaces()[0]
	coordinates[fact.Package] = packagegraph.CoordinateID{1}
	if _, _, err := workspaceInterfaceFacts(result, []hir.PackageID{fact.Package}, coordinates, LogicalSchemaV2); err == nil {
		t.Fatal("mismatched coordinate accepted")
	}
	coordinates[fact.Package] = fact.Coordinate
	if _, _, err := workspaceInterfaceFacts(result, []hir.PackageID{fact.Package + 1}, coordinates, LogicalSchemaV2); err == nil {
		t.Fatal("mismatched package set accepted")
	}
}

func TestGenericInterfaceHashParsing(t *testing.T) {
	value := moduleinterface.GenericHash{1, 2, 3}
	if parsed, err := moduleinterface.ParseGenericHash(value.String()); err != nil || parsed != value {
		t.Fatalf("round trip: %v", err)
	}
	for _, value := range []string{"", "fygif0-sha256-xyz", "fygif1-sha256-0000", "fygif0-sha256-" + string(make([]byte, 64))} {
		if _, err := moduleinterface.ParseGenericHash(value); err == nil {
			t.Fatal("malformed generic hash accepted")
		}
	}
}

func TestUnusedExportedTemplateChangesMapSourceInterface(t *testing.T) {
	workspace, result := loadGenericPlanWorkspace(t, false)
	_, before, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace.Root, "src", "main.fy")
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(text), "return *view;", "let T copy = *view; return copy;", 1)
	if changed == string(text) {
		t.Fatal("fixture mutation did not change the template")
	}
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err = workspaceloader.Load(workspace.Root)
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
	result, _, err = compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v", err)
	}
	_, after, err := PrepareWorkspaceV2(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	if before[0].Header().GenericInterfaceHash == after[0].Header().GenericInterfaceHash || before[0].Header().InterfaceHash != after[0].Header().InterfaceHash {
		t.Fatal("unused exported template change did not affect only the generic source interface")
	}
}
