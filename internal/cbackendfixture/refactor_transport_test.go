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
	// Include source bytes and Unit metadata for production and drop-trace profiles
	// of 52 cases. Refreshed at 03008309 for intentional trap-location emission;
	// the original pre-split snapshot at b49cfbfb predates that runtime payload.
	const wantHash = "bf5799da6db22b0fb2cfb25b2b86fb90e503852d017253ee65fb9655ab7043b6"
	if len(units) != 104 || len(encoded) != 446318 || fmt.Sprintf("%x", sha256.Sum256(encoded)) != wantHash {
		t.Fatalf("transport changed: units=%d bytes=%d sha256=%x", len(units), len(encoded), sha256.Sum256(encoded))
	}
}
