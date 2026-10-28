package semantic

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

// Preserve the original merge as an oracle, including its diagnostic evidence
// ordering and nil normalization for unusual or not-yet-materialized values.
func referenceMergeBorrowValues(values []borrowValueState) borrowValueState {
	result := borrowValueState{known: len(values) != 0}
	for _, value := range values {
		result.known = result.known && value.known
		result.origins = append(result.origins, value.origins...)
		result.loans = append(result.loans, value.loans...)
		result.pending = append(result.pending, value.pending...)
	}
	result.origins = compactOrigins(result.origins)
	result.loans = uniqueInts(result.loans)
	slices.SortFunc(result.pending, func(left, right pendingBorrowLoan) int {
		if value := compareSpan(left.borrow, right.borrow); value != 0 {
			return value
		}
		return strings.Compare(left.origin.name, right.origin.name)
	})
	result.pending = slices.CompactFunc(result.pending, func(left, right pendingBorrowLoan) bool {
		return left.subview == right.subview && left.owner == right.owner && compareSpan(left.borrow, right.borrow) == 0 && left.kind == right.kind && sameFlowOrigin(left.origin, right.origin)
	})
	return result
}

func TestBorrowJoinPreservesEvidenceAndIsolation(t *testing.T) {
	random := rand.New(rand.NewPCG(42, 17))
	owner := &symbol{}
	for trial := range 300 {
		value := borrowValueState{known: trial%7 != 0}
		for index := range 32 {
			declaration := hir.Span{Source: "main.fy", StartByte: index}
			if trial%5 == 0 {
				declaration.StartByte = random.IntN(8) // duplicates and declaration ties
			}
			origin := flowOrigin{name: []string{"Param(a)", "Param(b)"}[index%2], declaration: declaration, allocation: &hir.Span{StartByte: index + 100}}
			value.origins = append(value.origins, origin)
			value.loans = append(value.loans, index)
			if trial%3 == 0 {
				value.pending = append(value.pending, pendingBorrowLoan{owner: owner, borrow: declaration, origin: origin, kind: "shared"})
			}
		}
		if trial%11 == 0 {
			slices.Reverse(value.origins)
			slices.Reverse(value.loans)
			value.loans = append(value.loans, value.loans[0])
		}
		values := make([]borrowValueState, 2+trial%4)
		for index := range values {
			values[index] = value // the normal immutable fork path
			if trial%13 == 0 && index == 1 {
				values[index].known = !value.known
			}
			if trial%17 == 0 && index == 1 {
				values[index] = cloneBorrowValue(values[index])
				// Equal origin identity must not lose different allocation evidence.
				values[index].origins[0].allocation = &hir.Span{StartByte: 999}
				values[index].loans = append(values[index].loans, 1000)
			}
		}
		before := make([]borrowValueState, len(values))
		for index, value := range values {
			before[index] = cloneBorrowValue(value)
		}
		want := referenceMergeBorrowValues(values)
		got := mergeBorrowValues(values)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("trial %d merge differs: got %+v, want %+v", trial, got, want)
		}
		// Every output list must remain independently writable, on both paths.
		got.origins[0] = flowOrigin{name: "changed"}
		got.loans[0] = -1
		if len(got.pending) != 0 {
			got.pending[0].kind = "changed"
		}
		if !reflect.DeepEqual(values, before) {
			t.Fatalf("trial %d merge changed an input", trial)
		}
	}
}

func TestBorrowJoinEmptyAndMissingBranches(t *testing.T) {
	for _, values := range [][]borrowValueState{
		nil, {{}}, {{known: true}, {known: true}},
		{{known: true}, {known: false}},
		{{known: true, origins: []flowOrigin{}, loans: []int{}, pending: []pendingBorrowLoan{}}, {known: true}},
	} {
		if got, want := mergeBorrowValues(values), referenceMergeBorrowValues(values); !reflect.DeepEqual(got, want) {
			t.Fatalf("empty merge = %+v, want %+v", got, want)
		}
	}
	binding := &symbol{}
	base := borrowEnvironment{binding: {known: true, origins: []flowOrigin{{name: "Param(a)"}}}}
	for _, branches := range [][]borrowEnvironment{nil, {base, {}}, {{}, base}} {
		joined := mergeBorrowEnvironments(base, branches)
		if joined[binding].known {
			t.Fatal("missing branches must leave the origin unknown")
		}
	}
}

func BenchmarkMergeUnchangedBorrowBranches(b *testing.B) {
	input := make(borrowEnvironment)
	for local := range 64 {
		state := borrowValueState{known: true}
		for index := range 256 {
			state.origins = append(state.origins, flowOrigin{name: "Local", declaration: hir.Span{Source: "main.fy", StartByte: local*256 + index}})
			state.loans = append(state.loans, local*256+index)
		}
		input[&symbol{}] = state
	}
	branches := []borrowEnvironment{cloneBorrowEnvironment(input), cloneBorrowEnvironment(input)}
	b.ReportAllocs()
	for b.Loop() {
		if merged := mergeBorrowEnvironments(input, branches); len(merged) != len(input) {
			b.Fatal("incomplete branch join")
		}
	}
}
