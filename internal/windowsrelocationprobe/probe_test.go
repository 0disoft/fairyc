package windowsrelocationprobe

import (
	"strings"
	"testing"
)

func TestGenerateReturnsClosedDeterministicREL32Probe(t *testing.T) {
	t.Parallel()
	first := Generate()
	second := Generate()
	if first != second {
		t.Fatalf("Generate() is not deterministic")
	}
	if first.SourceSymbol != SourceSymbol || first.TargetSymbol != TargetSymbol ||
		first.VariantObject != VariantObject || first.VariantSource != VariantSource || first.VariantTarget != VariantTarget ||
		first.ReceiptSection != ReceiptSection || first.ReceiptMagic != ReceiptMagic ||
		first.SourceLine != SourceLine || first.ExpectedExit != ExpectedExit {
		t.Fatalf("Generate() identity = %+v", first)
	}
	lines := strings.Split(first.CallerSource, "\n")
	if first.SourceLine == 0 || int(first.SourceLine) > len(lines) || !strings.Contains(lines[first.SourceLine-1], "int "+first.SourceSymbol+"(") {
		t.Fatalf("Generate() source-line contract = %d in %q", first.SourceLine, first.CallerSource)
	}
	for _, fragment := range []string{
		"__declspec(noinline) int fy_rel32_source",
		"fy_rel32_target(value)",
		"#pragma section(\".fyreloc\", read)",
		"__declspec(allocate(\".fyreloc\"))",
		"fy_rel32_source(40) != 42",
		"fy_vsrc() == 42",
	} {
		if !strings.Contains(first.CallerSource, fragment) {
			t.Fatalf("caller source is missing %q", fragment)
		}
	}
	if !strings.Contains(first.TargetSource, "__declspec(noinline) int fy_rel32_target") {
		t.Fatalf("target source lost the cross-object target")
	}
	if !strings.Contains(first.TargetSource, "__declspec(noinline) int fy_vdst") {
		t.Fatal("target source lost the REL32 variant target")
	}
}
