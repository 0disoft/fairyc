package contractstatenative

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func TestVersionedSourceStateCEmissionFixtures(t *testing.T) {
	data, err := os.ReadFile("../../spec/fixtures/contract-state-native/manifest-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Schema string `json:"schema"`
		Cases  []struct {
			ID     string `json:"id"`
			Source string `json:"source"`
			Exit   string `json:"exit"`
			Status int    `json:"status"`
		} `json:"cases"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "fairyc.contract-state-native-fixtures/v0" || len(manifest.Cases) == 0 {
		t.Fatal("invalid fixture manifest")
	}
	seen := map[string]bool{}
	for _, fixture := range manifest.Cases {
		if seen[fixture.ID] || fixture.ID == "" || !filepath.IsLocal(fixture.Source) {
			t.Fatal("invalid fixture identity or path")
		}
		seen[fixture.ID] = true
		t.Run(fixture.ID, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("../..", fixture.Source))
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("transaction.fy", data), source.DefaultLimits())
			if err != nil || parsed.AST == nil {
				t.Fatalf("parse fixture: %v", err)
			}
			compiled, err := compiler.CheckExperimentalState(semantic.File{Path: "transaction.fy", Module: parsed.AST})
			if err != nil || !compiled.Verified {
				t.Fatalf("compile fixture: %v", err)
			}
			owner := compiled.StateTransactions[0]
			if string(owner.Program.Exit) != fixture.Exit {
				t.Fatal("fixture exit mismatch")
			}
			coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: owner.Program.Layout.ContractUID, VersionIdentity: "experimental-state-source/1"})
			if err != nil {
				t.Fatal(err)
			}
			handoff, err := PrepareCompiled(compiled, owner.Package, owner.Function, coordinate)
			if err != nil {
				t.Fatal(err)
			}
			unit, err := Emit(handoff, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(unit.Source, fmt.Sprintf("return %d;\n}", fixture.Status)) || !strings.Contains(unit.Source, "fycs_native_cell_write_v0(&transaction,") || !strings.Contains(unit.Source, "fycs_native_cell_read_v0(&transaction,") {
				t.Fatal("fixture C operations or exit absent")
			}
			repeated, err := Emit(handoff, target.X8664WindowsMSVC)
			if err != nil || repeated != unit {
				t.Fatal("non-deterministic fixture emission")
			}
		})
	}
}
