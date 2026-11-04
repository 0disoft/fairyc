package mir

import (
	"strings"
	"testing"
)

func TestDominatorTableSizeBoundaries(t *testing.T) {
	// Find the exact admitted block boundary without allocating the matrix.
	last := 0
	for {
		if _, _, ok := dominatorTableSize(last + 1); !ok {
			break
		}
		last++
	}
	for _, count := range []int{0, 1, 63, 64, 65, last - 1, last, last + 1, 100_000, int(^uint(0) >> 1), -1} {
		words, cells, ok := dominatorTableSize(count)
		if want := count >= 0 && count <= last; ok != want {
			t.Fatalf("count %d: admitted=%t want=%t", count, ok, want)
		}
		if ok && (cells != count*words || cells > maxDominatorBytes/8) {
			t.Fatalf("count %d: invalid allocation %d words", count, cells)
		}
	}
}

func TestTrapCheckDominanceBudgetFailsClosed(t *testing.T) {
	function := distinctArithmeticChecks(1)
	for count := len(function.Blocks); ; count++ {
		function.Blocks = append(function.Blocks, Block{ID: BlockID(count)})
		if _, _, ok := dominatorTableSize(len(function.Blocks)); !ok {
			break
		}
	}
	failures := verifyTrapCheckEdges(&function)
	if len(failures) != 1 || failures[0].Code != "V1501" || !strings.Contains(failures[0].Detail, "64 MiB verifier budget") {
		t.Fatalf("unchecked MIR must be rejected: %+v", failures)
	}
	// The cap only applies when a trapping operation needs dominance analysis.
	for i := range function.Blocks {
		function.Blocks[i].Operations = nil
	}
	if failures := verifyTrapCheckEdges(&function); len(failures) != 0 {
		t.Fatalf("branch-only function allocated dominance: %+v", failures)
	}
}
