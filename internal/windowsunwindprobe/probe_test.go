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
		unit.FrameSource != repeated.FrameSource || unit.RaiserSource != repeated.RaiserSource ||
		unit.FrameSymbol != repeated.FrameSymbol || unit.FrameObject != repeated.FrameObject ||
		unit.LowHandlerSymbol != repeated.LowHandlerSymbol || unit.OptimizedHandlerSymbol != repeated.OptimizedHandlerSymbol {
		t.Fatal("repeated Windows unwind probe generation drifted")
	}
	if unit.FrameSymbol != FrameSymbol || unit.FrameObject != FrameObject ||
		unit.LowHandlerSymbol != LowHandlerSymbol || unit.OptimizedHandlerSymbol != OptimizedHandlerSymbol ||
		!strings.Contains(unit.FrameSource, "void "+unit.FrameSymbol+"(") {
		t.Fatalf("Windows unwind handler identities = %+v", unit)
	}
	for profile, want := range map[string]string{"low": LowHandlerSymbol, "optimized": OptimizedHandlerSymbol} {
		if got, ok := unit.HandlerSymbolForProfile(profile); !ok || got != want {
			t.Fatalf("HandlerSymbolForProfile(%q) = %q, %t; want %q, true", profile, got, ok, want)
		}
	}
	if got, ok := unit.HandlerSymbolForProfile("other"); ok || got != "" {
		t.Fatalf("HandlerSymbolForProfile(other) = %q, %t", got, ok)
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
