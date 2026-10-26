package packageartifact

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestBatchExtractionMatchesIndependentPackages(t *testing.T) {
	_, original, identities, coordinates := checkedWorkspaceProgram(t)
	for _, tc := range []struct {
		name string
		run  func(mir.Program, map[hir.PackageID]packagegraph.InstanceID, map[hir.PackageID]packagegraph.CoordinateID) ([]Artifact, error)
	}{
		{"ordinary", ExtractAll},
		{"storage transfers", ExtractAllStorageTransfers},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := mir.Clone(original)
			for operation := range 2 {
				artifacts, err := tc.run(program, identities, coordinates)
				if err != nil {
					t.Fatal(err)
				}
				for _, artifact := range artifacts {
					independent, err := Extract(program, artifact.PackageID, identities, coordinates)
					if err != nil || !reflect.DeepEqual(artifact, independent) {
						t.Fatalf("operation %d, package %d: batch differs from independent extraction: %v", operation, artifact.PackageID, err)
					}
					for _, record := range artifact.Functions {
						for _, function := range program.Functions {
							if record.Body.Symbol == function.Symbol && !reflect.DeepEqual(record.Body, cloneFunction(function)) {
								t.Fatal("extraction reused a body from an earlier operation")
							}
						}
					}
				}
				for fi := range program.Functions {
					for bi := range program.Functions[fi].Blocks {
						value := &program.Functions[fi].Blocks[bi].Terminator.ReturnValue
						if value.Kind == mir.OperandI32 {
							value.I32++
						}
					}
				}
			}
		})
	}
}

func TestBatchExtractionPreservesMissingOwnerDiagnostic(t *testing.T) {
	program, identities, coordinates := extractionContextFixture(2, 1)
	delete(identities, 100)
	delete(coordinates, 100)
	_, err := ExtractAll(program, identities, coordinates)
	if err == nil || !strings.Contains(err.Error(), "package 100 has no PackageInstanceId") {
		t.Fatalf("metadata indexing changed the first owner diagnostic: %v", err)
	}
}
