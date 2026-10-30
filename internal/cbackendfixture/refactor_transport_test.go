package cbackendfixture

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/cbackend"
	"path/filepath"
	"testing"
)

func TestEmissionTransportAcrossResponsibilitySplit(t *testing.T) {
	report, err := Run(filepath.Join("..", "..", "spec", "fixtures", "c-emission", "manifest-v0.json"))
	if err != nil || len(report.Failures) != 0 {
		t.Fatalf("fixture baseline: %v %+v", err, report.Failures)
	}
	var units []cbackend.Unit
	for _, fixture := range report.Emitted {
		// Preserve the original 52-case refactor snapshot. Added cases have
		// independent source-linked drop assertions in the fixture tests.
		switch fixture.ID {
		case "pass.c-emission.resource-option-drop", "pass.c-emission.resource-result-drop-ok", "pass.c-emission.resource-result-drop-err":
			continue
		case "pass.c-emission.nested-resource-drop":
			continue
		}
		units = append(units, fixture.Unit, fixture.TraceUnit)
	}
	encoded, err := json.Marshal(units)
	if err != nil {
		t.Fatal(err)
	}
	// Include source bytes and Unit metadata for production and drop-trace profiles
	// of 52 cases. Refreshed for intentional conditional-branch braces; the
	// generated C formatting changes without changing MIR or runtime behavior.
	const wantHash = "7bd6f790248d811337150509279df30cadd6759cf213c06b924a284ee12e9c08"
	if len(units) != 104 || len(encoded) != 447846 || fmt.Sprintf("%x", sha256.Sum256(encoded)) != wantHash {
		t.Fatalf("transport changed: units=%d bytes=%d sha256=%x", len(units), len(encoded), sha256.Sum256(encoded))
	}
}
