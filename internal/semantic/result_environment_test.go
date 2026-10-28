package semantic

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestResultEnvironmentCloneKeepsBranchUpdatesIsolated(t *testing.T) {
	owner := &symbol{}
	evidence := make([]hir.Span, 32, 64)
	for index := range evidence {
		evidence[index] = hir.Span{Source: "main.fy", StartByte: index, EndByte: index + 1}
	}
	want := slices.Clone(evidence)
	spare := evidence[:cap(evidence)]
	sentinel := hir.Span{Source: "spare-capacity"}
	spare[len(evidence)] = sentinel
	baseline := resultEnvironment{owner: {pending: true, producers: evidence}}
	handled := cloneResultEnvironment(baseline)
	retained := cloneResultEnvironment(baseline)
	handled[owner] = resultObligation{}
	delete(handled, owner)
	if !baseline[owner].pending || !retained[owner].pending || !slices.Equal(baseline[owner].producers, want) {
		t.Fatal("handling/deleting one branch changed another branch")
	}
	later := hir.Span{Source: "main.fy", StartByte: 100, EndByte: 101}
	reassigned := cloneResultEnvironment(baseline)
	reassigned[owner] = pendingResult(later)
	var a analyzer
	joined := make(resultEnvironment)
	a.mergeResultBranches(joined, baseline, []resultBranch{
		{environment: retained, reachable: true},
		{environment: reassigned, reachable: true},
		{environment: resultEnvironment{owner: {}}, reachable: true},
	})
	wantJoined := append(slices.Clone(want), later)
	if !joined[owner].pending || !slices.Equal(joined[owner].producers, wantJoined) {
		t.Fatalf("joined obligation = %+v", joined[owner])
	}
	// A join must own its new evidence. Even append with spare input capacity
	// must not overwrite a sibling's retained producer list.
	joined[owner].producers[0] = later
	if !slices.Equal(baseline[owner].producers, want) || !slices.Equal(retained[owner].producers, want) ||
		!slices.Equal(reassigned[owner].producers, []hir.Span{later}) || spare[len(evidence)] != sentinel {
		t.Fatal("merging/rewriting evidence changed an incoming branch")
	}
}

func TestResultEnvironmentCloneEmptyIsWritable(t *testing.T) {
	for _, input := range []resultEnvironment{nil, {}} {
		cloned := cloneResultEnvironment(input)
		cloned[&symbol{}] = resultObligation{pending: true}
		if len(input) != 0 || len(cloned) != 1 {
			t.Fatal("empty environment was not independently writable")
		}
	}
}

func BenchmarkCloneResultEnvironmentEvidence(b *testing.B) {
	for _, count := range []int{8, 512} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			input := make(resultEnvironment)
			for local := range 64 {
				producers := make([]hir.Span, count)
				for index := range producers {
					producers[index] = hir.Span{Source: "main.fy", StartByte: local*count + index}
				}
				input[&symbol{}] = resultObligation{pending: true, producers: producers}
			}
			b.ReportAllocs()
			for b.Loop() {
				if cloned := cloneResultEnvironment(input); len(cloned) != len(input) {
					b.Fatalf("cloned locals = %d, want %d", len(cloned), len(input))
				}
			}
		})
	}
}
