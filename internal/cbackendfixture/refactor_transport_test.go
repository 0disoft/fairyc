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
		units = append(units, fixture.Unit, fixture.TraceUnit)
	}
	encoded, err := json.Marshal(units)
	if err != nil {
		t.Fatal(err)
	}
	// Measured before the responsibility split at b49cfbfb. Include both source
	// bytes and Unit metadata for production and drop-trace profiles of 52 cases.
	const wantHash = "42a758b79df7b3f0139a3ed9a7617ebe21b1b11e1b1bbd6bc845ea2819270c8a"
	if len(units) != 104 || len(encoded) != 392816 || fmt.Sprintf("%x", sha256.Sum256(encoded)) != wantHash {
		t.Fatalf("transport changed: units=%d bytes=%d sha256=%x", len(units), len(encoded), sha256.Sum256(encoded))
	}
}
