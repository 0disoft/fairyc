package foreignstructprobe

import (
	"strings"
	"testing"
)

func TestGenerateOwnsAggregateArgumentAndResultShapes(t *testing.T) {
	t.Parallel()
	unit := Generate()
	if unit.FairySource == "" || unit.CompanionC == "" || unit.ExpectedExit != 42 {
		t.Fatalf("Generate() returned an incomplete unit: exit=%d", unit.ExpectedExit)
	}
	for _, required := range []string{
		"struct fairyc_fixture_floating_array_packet_v0",
		"fairyc_fixture_floating_array_observe_v0",
		"fairyc_fixture_floating_array_small_observe_v0",
		"struct fairyc_fixture_status_v0",
		"struct fairyc_fixture_large_status_v0",
		"fairyc_fixture_make_status_v0",
		"fairyc_fixture_make_large_status_v0",
		"fairyc_fixture_change_status_v0",
		"fairyc_fixture_change_large_v0",
		"fairyc_fixture_mixed_status_v0",
		"fairyc_fixture_exhausted_v0",
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

func TestCScalarArrayTargetCoverage(t *testing.T) {
	portable := Generate()
	posix := GeneratePOSIX()
	for _, name := range []string{"bool", "char", "double", "float", "int", "long", "longlong", "schar", "short", "size_t", "uchar", "uint", "ulong", "ulonglong", "ushort"} {
		for _, unit := range []Unit{portable, posix} {
			if !strings.Contains(unit.FairySource, "check_c_array_"+name+"(2)") || !strings.Contains(unit.CompanionC, "fairyc_c_array_"+name+"_observe") {
				t.Fatalf("missing C array oracle %s", name)
			}
		}
	}
	if strings.Contains(portable.FairySource, "c::ssize_t") || !strings.Contains(posix.FairySource, "c::ssize_t[3]") {
		t.Fatal("POSIX array provider boundary lost")
	}
	if strings.Contains(posix.CompanionC, "%!") || strings.Contains(posix.FairySource, "%!") {
		t.Fatal("invalid probe template expansion")
	}
}
