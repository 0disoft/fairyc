package windowsunwindprobe

import (
	"slices"
	"strings"
	"testing"
)

func TestGenerateReturnsDeterministicClosedSources(t *testing.T) {
	t.Parallel()
	unit := Generate()
	repeated := Generate()
	if !slices.Equal(unit.Cases, repeated.Cases) || unit.CatcherSource != repeated.CatcherSource ||
		unit.FrameSource != repeated.FrameSource || unit.RaiserSource != repeated.RaiserSource {
		t.Fatal("repeated Windows unwind probe generation drifted")
	}
	if !slices.Equal(unit.Cases, []string{"abnormal-finally", "exception-filter-payload", "virtual-unwind-walk"}) {
		t.Fatalf("Windows unwind cases = %q", unit.Cases)
	}
	for _, required := range []string{
		"RtlLookupFunctionEntry",
		"RtlVirtualUnwind",
		"GetExceptionInformation()",
		"__except",
		"AbnormalTermination()",
		"RaiseException",
		"EXCEPTION_NONCONTINUABLE",
	} {
		if !strings.Contains(unit.CatcherSource, required) && !strings.Contains(unit.FrameSource, required) && !strings.Contains(unit.RaiserSource, required) {
			t.Errorf("Windows unwind probe is missing %q", required)
		}
	}
	for _, caseID := range unit.Cases {
		if strings.Contains(unit.CatcherSource, caseID) || strings.Contains(unit.FrameSource, caseID) || strings.Contains(unit.RaiserSource, caseID) {
			t.Errorf("Windows unwind probe source leaks case ID %q", caseID)
		}
	}
	cloned := CloneCases(unit)
	cloned[0] = "mutated"
	if unit.Cases[0] == "mutated" {
		t.Fatal("CloneCases() leaked caller mutation")
	}
}
