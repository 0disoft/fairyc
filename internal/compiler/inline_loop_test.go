package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestInlineLoopsExecuteAndRestore(t *testing.T) {
	const source = `module main;
@inline pub fn i32 sum(i32 n) {
    var i32 sum = 0;
    for (var i32 i = 0; i < n; i = i + 1) {
        if (i == 2) { continue; }
        if (i == 6) { break; }
        sum = sum + i;
    }
    var i32 tail = 0;
    while (tail < 3 && sum < 50) { tail = tail + 1; }
    return sum + tail;
}
fn i32 main() { assert(sum(0) == 3); return sum(8) + 26; }`
	result := checkInlineCalls(t, source)
	marker := inlineMutationMarker(result)
	if !strings.HasPrefix(marker, hir.InlineLoopPrefix) {
		t.Fatalf("missing loop-specific marker: %s", marker)
	}
	if renamed := inlineMutationMarker(checkInlineCalls(t, strings.ReplaceAll(source, "tail", "count"))); renamed != marker {
		t.Fatal("a loop-local rename changed its identity")
	}
	if updated := inlineMutationMarker(checkInlineCalls(t, strings.Replace(source, "i == 6", "i == 7", 1))); updated == marker {
		t.Fatal("changing the break condition did not invalidate the identity")
	}
	classes, err := inlinebody.HIR(result.Semantic.HIR, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: {1}})
	if err != nil || classes[hir.DefaultPackage] == (inlinebody.Hash{}) {
		t.Fatalf("loop identity missing from package export class: %v", err)
	}
	wire, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := mircodec.Decode(wire, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := inlinebody.VerifyProgram(result.Semantic.HIR, restored); err != nil {
		t.Fatal(err)
	}
	value, err := interpreter.Execute(restored, "main::main")
	if err != nil || value.Kind != interpreter.OutcomeReturn || value.Value != 42 {
		t.Fatalf("restored loops: %+v %v", value, err)
	}
	runExpandedLanguageNative(t, result, "main::main", "")
}
