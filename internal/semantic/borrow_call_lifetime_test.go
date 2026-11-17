package semantic

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/0disoft/fairyc/internal/ast"
)

func temporaryBorrowCall(t testing.TB, origins int) (*borrowFlowAnalyzer, *ast.Expression, borrowEnvironment) {
	t.Helper()
	f, _ := scalarLoopFlow(t, "module main;fn void sink(ref i32 a,ref i32 b,ref i32 c,ref i32 d){}fn i32 f(ref i32 p,ref i32 q,ref i32 r,ref i32 s){sink(p,q,r,s);return 0;}")
	environment := f.parameterEnvironment()
	for binding, value := range environment {
		origin := value.origins[0]
		value.origins = make([]flowOrigin, origins)
		for index := range value.origins {
			value.origins[index] = origin
			value.origins[index].declaration.StartByte += index
			value.origins[index].declaration.EndByte += index
		}
		environment[binding] = value
	}
	return f, f.function.Body.Statements[0].Expression, environment
}

// Count only references in retired slots, deduplicating evidence shared by the
// two stacks. The retained backing arrays themselves remain reusable capacity.
func retiredCallOrigins(f *borrowFlowAnalyzer) (slots int, bytes uintptr) {
	seen := make(map[*flowOrigin]bool)
	add := func(value borrowValueState) {
		if len(value.origins) == 0 {
			return
		}
		slots++
		first := &value.origins[0]
		if !seen[first] {
			seen[first] = true
			bytes += uintptr(cap(value.origins)) * unsafe.Sizeof(flowOrigin{})
		}
	}
	for _, value := range f.callBorrows[len(f.callBorrows):cap(f.callBorrows)] {
		add(value)
	}
	for _, access := range f.callAccesses[len(f.callAccesses):cap(f.callAccesses)] {
		add(access.value)
	}
	return slots, bytes
}

func assertRetiredCallSlotsEmpty(t *testing.T, f *borrowFlowAnalyzer) {
	t.Helper()
	for _, value := range f.callBorrows[len(f.callBorrows):cap(f.callBorrows)] {
		if !reflect.DeepEqual(value, borrowValueState{}) {
			t.Fatal("retired borrow still retains evidence")
		}
	}
	for _, access := range f.callAccesses[len(f.callAccesses):cap(f.callAccesses)] {
		if !reflect.DeepEqual(access, borrowCallAccess{}) {
			t.Fatal("retired access still retains evidence")
		}
	}
}

func TestBorrowCallReleasesRetiredEvidence(t *testing.T) {
	for _, limit := range []int{defaultBorrowWorkLimit, 3} {
		t.Run(fmt.Sprintf("budget_%d", limit), func(t *testing.T) {
			f, call, environment := temporaryBorrowCall(t, 1)
			f.analyzer.borrowWork = nil
			f.analyzer.borrowWorkLimit = limit
			_, reachable := f.evaluateExpression(call, environment)
			if reachable != (limit == defaultBorrowWorkLimit) || len(f.callBorrows) != 0 || len(f.callAccesses) != 0 {
				t.Fatalf("call did not restore its logical state: reachable=%t borrows=%d accesses=%d", reachable, len(f.callBorrows), len(f.callAccesses))
			}
			assertRetiredCallSlotsEmpty(t, f)
		})
	}
}

func TestBorrowCallPreservesOuterScopeAndReturnedEvidence(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn ref i32 identity(ref i32 p){return p;}fn ref i32 f(ref i32 p){return identity(p);}")
	f.analyzer.inferBorrowReturnSummaries()
	environment := f.parameterEnvironment()
	var outer borrowValueState
	for _, value := range environment {
		outer = value
	}
	expectedOuter := cloneBorrowValue(outer)
	f.callBorrows = make([]borrowValueState, 1, 8)
	f.callBorrows[0] = outer
	f.callAccesses = make([]borrowCallAccess, 1, 8)
	f.callAccesses[0] = borrowCallAccess{value: outer}
	value, reachable := f.evaluateExpression(f.function.Body.Statements[0].Expression, environment)
	if !reachable || !value.known || len(value.origins) != 1 || value.origins[0].name != "Param(0)" {
		t.Fatalf("call cleanup changed its returned proof: %+v", value)
	}
	if len(f.callBorrows) != 1 || len(f.callAccesses) != 1 || cap(f.callBorrows) != 8 || cap(f.callAccesses) != 8 ||
		!reflect.DeepEqual(f.callBorrows[0], expectedOuter) || !reflect.DeepEqual(f.callAccesses[0], borrowCallAccess{value: expectedOuter}) {
		t.Fatal("call cleanup changed its outer scope or reusable capacity")
	}
	assertRetiredCallSlotsEmpty(t, f)
}

func TestNestedBorrowCallClearsPeakTail(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn i32 number(ref i32 p){return *p;}fn void sink(ref i32 p,i32 a,i32 b,i32 c){}fn i32 f(ref i32 p,ref i32 q,ref i32 r,ref i32 s){sink(p,number(q),number(r),number(s));return 0;}")
	// The final inner call reaches a deeper slot than the outer call's final
	// length. Clearing only the outer tail cannot release that slot's evidence.
	if _, reachable := f.evaluateExpression(f.function.Body.Statements[0].Expression, f.parameterEnvironment()); !reachable || len(f.callBorrows) != 0 || len(f.callAccesses) != 0 {
		t.Fatal("nested call failed to restore its logical stacks")
	}
	assertRetiredCallSlotsEmpty(t, f)
}

func BenchmarkBorrowCallRetiredEvidence(b *testing.B) {
	for _, origins := range []int{1, 4096} {
		b.Run(fmt.Sprintf("origins_%d", origins), func(b *testing.B) {
			f, call, environment := temporaryBorrowCall(b, origins)
			budget := f.analyzer.borrowBudget(f.function)
			b.ReportAllocs()
			for b.Loop() {
				budget.remaining = defaultBorrowWorkLimit
				if _, reachable := f.evaluateExpression(call, environment); !reachable {
					b.Fatal("call failed")
				}
			}
			slots, bytes := retiredCallOrigins(f)
			b.ReportMetric(float64(slots), "retired-slots")
			b.ReportMetric(float64(bytes), "retained-origin-B")
		})
	}
}
