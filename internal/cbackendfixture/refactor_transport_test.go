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
	// Include source bytes and Unit metadata for both emission profiles of all
	// 52 cases. The native differential suite checks the behavior of these units.
	const wantHash = "06230ec446dbb29c730050585902a4a766f1e1d1060f946198f435684f60dd0b"
	if len(units) != 104 || len(encoded) != 466480 || fmt.Sprintf("%x", sha256.Sum256(encoded)) != wantHash {
		t.Fatalf("transport changed: units=%d bytes=%d sha256=%x", len(units), len(encoded), sha256.Sum256(encoded))
	}
}
