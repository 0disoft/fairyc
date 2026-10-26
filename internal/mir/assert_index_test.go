package mir

import (
	"fmt"
	"slices"
	"testing"
)

func TestAssertEdgesLateLookupStillRejectsInvalidEdges(t *testing.T) {
	function := branchOnlyFunction(32)
	if failures := verifyAssertEdges(&function); len(failures) != 0 {
		t.Fatalf("assert-free graph: %+v", failures)
	}
	assertion := len(function.Blocks) - 1
	success, failure := BlockID(assertion+1), BlockID(assertion+2)
	function.Blocks[assertion].Terminator = Terminator{Kind: TerminatorSwitch, SwitchKind: "assert", Entity: "late-assert", Successors: []BlockID{success, failure}}
	function.Blocks = append(function.Blocks,
		Block{ID: success, Terminator: Terminator{Kind: TerminatorReturn}},
		Block{ID: failure, Terminator: Terminator{Kind: TerminatorTrap, TrapCategories: []string{"assertion-failed"}}})
	for _, test := range []struct {
		name string
		edit func(*Function)
		bad  bool
	}{
		{"valid", func(*Function) {}, false},
		{"wrong trap kind", func(f *Function) { f.Blocks[failure].Terminator.Kind = TerminatorReturn }, true},
		{"wrong category", func(f *Function) { f.Blocks[failure].Terminator.TrapCategories = []string{"overflow"} }, true},
		{"extra category", func(f *Function) {
			f.Blocks[failure].Terminator.TrapCategories = []string{"assertion-failed", "overflow"}
		}, true},
		{"missing target", func(f *Function) { f.Blocks[assertion].Terminator.Successors = []BlockID{success, failure + 1} }, true},
		{"duplicate edge", func(f *Function) { f.Blocks[assertion].Terminator.Successors = []BlockID{success, success} }, true},
		{"missing edge", func(f *Function) { f.Blocks[assertion].Terminator.Successors = []BlockID{success} }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := function
			candidate.Blocks = slices.Clone(function.Blocks)
			test.edit(&candidate)
			failures := verifyAssertEdges(&candidate)
			if !test.bad {
				if len(failures) != 0 {
					t.Fatalf("valid late assertion: %+v", failures)
				}
			} else if len(failures) != 1 || failures[0].Code != "V1501" || failures[0].Entity != "late-assert" {
				t.Fatalf("invalid late assertion: %+v", failures)
			}
		})
	}
	// A malformed early assertion must not prevent later target validation.
	function.Blocks[0].Terminator = Terminator{Kind: TerminatorSwitch, SwitchKind: "assert", Entity: "early-assert", Successors: []BlockID{1}}
	if failures := verifyAssertEdges(&function); len(failures) != 1 || failures[0].Entity != "early-assert" {
		t.Fatalf("early malformed assertion with valid late assertion: %+v", failures)
	}
	function.Blocks[failure].Terminator.TrapCategories = []string{"overflow"}
	if failures := verifyAssertEdges(&function); len(failures) != 2 || failures[0].Entity != "early-assert" || failures[1].Entity != "late-assert" {
		t.Fatalf("late target validation was skipped after early failure: %+v", failures)
	}
}

func BenchmarkAssertFreeEdgeVerification(b *testing.B) {
	for _, branches := range []int{341, 2730} {
		function := branchOnlyFunction(branches)
		b.Run(fmt.Sprintf("blocks=%d", len(function.Blocks)), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if failures := verifyAssertEdges(&function); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}
