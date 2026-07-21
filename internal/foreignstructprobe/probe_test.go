package foreignstructprobe

import (
	"strings"
	"testing"
)

func TestGenerateOwnsBothAggregateResultShapes(t *testing.T) {
	t.Parallel()
	unit := Generate()
	if unit.FairySource == "" || unit.CompanionC == "" || unit.ExpectedExit != 42 {
		t.Fatalf("Generate() returned an incomplete unit: exit=%d", unit.ExpectedExit)
	}
	for _, required := range []string{
		"struct fairyc_fixture_status_v0",
		"struct fairyc_fixture_large_status_v0",
		"fairyc_fixture_make_status_v0",
		"fairyc_fixture_make_large_status_v0",
	} {
		if !strings.Contains(unit.FairySource, required) || !strings.Contains(unit.CompanionC, required) {
			t.Errorf("probe sources do not both contain %q", required)
		}
	}
	if len(unit.RequiredCFragments) == 0 {
		t.Fatal("Generate() returned no generated-C expectations")
	}
}

func TestGenerateReturnsIndependentExpectations(t *testing.T) {
	t.Parallel()
	first := Generate()
	second := Generate()
	original := second.RequiredCFragments[0]
	first.RequiredCFragments[0] = "mutated"
	if second.RequiredCFragments[0] != original {
		t.Fatal("Generate() shared mutable expectation storage")
	}
}
