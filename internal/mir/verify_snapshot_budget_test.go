package mir

import (
	"github.com/0disoft/fairyc/internal/hir"
	"testing"
)

func TestVerificationJoinPointsLinearAndLoop(t *testing.T) {
	function := reverseFieldStoreChain(2048)
	for _, join := range verificationJoinPoints(&function) {
		if join {
			t.Fatal("straight-line state requires a snapshot")
		}
	}
	if failures := Verify(Program{Functions: []Function{function}}); len(failures) != 0 {
		t.Fatalf("linear graph: %+v", failures)
	}
	// A backedge to the entry joins the implicit caller edge, even though the
	// entry has only one written predecessor. The worklist must reach a fixed point.
	function = Function{Symbol: 1, Name: "test::loop", ResultType: hir.TypeVoid, Entry: 0,
		Blocks: []Block{{ID: 0, Terminator: Terminator{Kind: TerminatorGoto, Successors: []BlockID{1}, UnwindSuccessor: NoBlock}},
			{ID: 1, Terminator: Terminator{Kind: TerminatorGoto, Successors: []BlockID{0}, UnwindSuccessor: NoBlock}}}}
	joins := verificationJoinPoints(&function)
	if !joins[0] || joins[1] {
		t.Fatalf("loop join points: %v", joins)
	}
	if failures := verifyFunction(&function, nil, nil, nil, nil, nil); len(failures) != 0 {
		t.Fatalf("loop: %+v", failures)
	}
}

func TestVerificationSnapshotBudgetAndOwnership(t *testing.T) {
	state := verificationState{live: map[hir.SymbolID]bool{1: true}, flags: map[string]flagState{"drop": {}}, regions: map[hir.SymbolID]string{2: "region"}}
	for _, available := range []int{2, 3, 4} {
		remaining := available
		snapshot, ok := cloneVerificationSnapshot(state, &remaining)
		if ok != (available >= 3) {
			t.Fatalf("budget %d admitted=%t", available, ok)
		}
		if !ok {
			if remaining != available {
				t.Fatal("rejected clone consumed budget")
			}
			continue
		}
		if remaining != available-3 {
			t.Fatal("incorrect charge")
		}
		snapshot.live[1] = false
		snapshot.flags["drop"] = flagState{active: true}
		snapshot.regions[2] = "changed"
		if !state.live[1] || state.flags["drop"].active || state.regions[2] != "region" {
			t.Fatal("snapshot aliases input")
		}
	}
}

func BenchmarkVerifyLinearState(b *testing.B) {
	function := reverseFieldStoreChain(1024)
	b.ReportAllocs()
	for range b.N {
		if failures := verifyFunction(&function, nil, nil, nil, nil, nil); len(failures) != 0 {
			b.Fatal(failures)
		}
	}
}
