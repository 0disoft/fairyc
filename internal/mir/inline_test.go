package mir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func inlineVerificationFixture(blocks int) *Function {
	fn := &Function{Name: "main::f", ResultType: hir.TypeI32, InlineBody: hir.InlineFunctionPrefix + strings.Repeat("a7", 32)}
	for index := range blocks {
		fn.Blocks = append(fn.Blocks, Block{ID: BlockID(index), Terminator: Terminator{Kind: TerminatorReturn}})
	}
	return fn
}

func checkInlineVerification(t *testing.T, fn *Function, functions map[hir.SymbolID]*Function, detail string) {
	t.Helper()
	before, err := json.Marshal(functions)
	if err != nil {
		t.Fatal(err)
	}
	failures := verifyInlineFunction(fn, functions)
	if detail == "" {
		if len(failures) != 0 {
			t.Fatalf("unexpected failures: %+v", failures)
		}
	} else if len(failures) != 1 || failures[0].Code != CodeFunctionIdentity || failures[0].Entity != fn.Name || failures[0].Detail != "inline body: "+detail {
		t.Fatalf("failures = %+v, want %q", failures, detail)
	}
	after, err := json.Marshal(functions)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("verification changed its input: %v", err)
	}
}

func TestInlineVerificationIdentityAndErrorOrder(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*Function)
		want string
	}{
		{"unmarked", func(fn *Function) { fn.InlineBody = ""; fn.ResultType = hir.TypeBool }, ""},
		{"short", func(fn *Function) { fn.InlineBody = fn.InlineBody[:len(fn.InlineBody)-1] }, "invalid versioned identity"},
		{"long", func(fn *Function) { fn.InlineBody += "a" }, "invalid versioned identity"},
		{"missing prefix", func(fn *Function) { fn.InlineBody = strings.Repeat("a", len(fn.InlineBody)) }, "invalid versioned identity"},
		{"uppercase", func(fn *Function) { fn.InlineBody = strings.ToUpper(fn.InlineBody) }, "invalid versioned identity"},
		{"uppercase digest", func(fn *Function) { fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("A7", 32) }, "invalid versioned identity"},
		{"nonhex", func(fn *Function) { fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("z", 64) }, "invalid versioned identity"},
		{"all lowercase hex", func(fn *Function) { fn.InlineBody = hir.InlineFunctionPrefix + strings.Repeat("0123456789abcdef", 4) }, ""},
		{"identity before signature", func(fn *Function) { fn.InlineBody = "bad"; fn.ResultType = hir.TypeBool }, "invalid versioned identity"},
		{"signature before parameter", func(fn *Function) { fn.ResultType = hir.TypeBool; fn.ParameterTypes = []hir.Type{hir.TypeBool} }, "unsupported function signature or state"},
		{"parameter before operation", func(fn *Function) {
			fn.ParameterTypes = []hir.Type{hir.TypeBool}
			fn.Blocks[0].Operations = []Operation{{}}
		}, "non-i32 parameter"},
		{"operation before budget", func(fn *Function) { fn.Blocks[0].Operations = make([]Operation, 129) }, "unsupported operation "},
		{"terminator before budget", func(fn *Function) {
			fn.Blocks[0].Operations = make([]Operation, 129)
			for i := range fn.Blocks[0].Operations {
				fn.Blocks[0].Operations[i].Kind = OperationAssign
			}
			fn.Blocks[0].Terminator.Kind = TerminatorUnreachable
		}, "calls and resource/control-flow extensions are closed"},
		{"control flow before calls", func(fn *Function) {
			fn.Blocks[0].Terminator = Terminator{Kind: TerminatorCall, Callee: 99, Successors: []BlockID{-1}}
		}, "cyclic or invalid control flow"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn := inlineVerificationFixture(1)
			test.edit(fn)
			checkInlineVerification(t, fn, map[hir.SymbolID]*Function{fn.Symbol: fn}, test.want)
		})
	}
}

func TestInlineVerificationOnlyAcceptsDetachedEmptyUnreachableJoins(t *testing.T) {
	fn := inlineVerificationFixture(2)
	fn.Blocks[1].Terminator.Kind = TerminatorUnreachable
	functions := map[hir.SymbolID]*Function{fn.Symbol: fn}
	checkInlineVerification(t, fn, functions, "")
	fn.Blocks[0].Terminator = Terminator{Kind: TerminatorGoto, Successors: []BlockID{1}}
	checkInlineVerification(t, fn, functions, "calls and resource/control-flow extensions are closed")
}

func TestInlineVerificationLimitsAndControlFlow(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit int
		edit  func(*Function, int)
		want  string
	}{
		{"parameters", 4, func(fn *Function, count int) {
			for range count {
				fn.ParameterTypes = append(fn.ParameterTypes, hir.TypeI32)
			}
		}, "unsupported function signature or state"},
		{"blocks", 256, func(fn *Function, count int) { fn.Blocks = inlineVerificationFixture(count).Blocks }, "unsupported function signature or state"},
		{"operations", 128, func(fn *Function, count int) {
			for range count {
				fn.Blocks[0].Operations = append(fn.Blocks[0].Operations, Operation{Kind: OperationAssign})
			}
		}, "body exceeds operation budget"},
	} {
		for _, count := range []int{test.limit - 1, test.limit, test.limit + 1} {
			t.Run(fmt.Sprintf("%s/%d", test.name, count), func(t *testing.T) {
				fn := inlineVerificationFixture(1)
				test.edit(fn, count)
				want := ""
				if count > test.limit {
					want = test.want
				}
				checkInlineVerification(t, fn, map[hir.SymbolID]*Function{fn.Symbol: fn}, want)
			})
		}
	}
	for _, successor := range []BlockID{-1, 0, 2} {
		fn := inlineVerificationFixture(2)
		fn.Blocks[0].Terminator.Successors = []BlockID{successor}
		checkInlineVerification(t, fn, map[hir.SymbolID]*Function{fn.Symbol: fn}, "cyclic or invalid control flow")
	}
	fn := inlineVerificationFixture(2)
	fn.Blocks[1].Terminator.Successors = []BlockID{1} // Unreachable cycles must also fail.
	checkInlineVerification(t, fn, map[hir.SymbolID]*Function{fn.Symbol: fn}, "cyclic or invalid control flow")
}

func TestInlineVerificationCallGraph(t *testing.T) {
	const graphError = "calls require an acyclic graph of at most 64 same-module pure inline functions"
	for _, count := range []int{63, 64, 65} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			functions := make(map[hir.SymbolID]*Function)
			for index := range count {
				fn := inlineVerificationFixture(1)
				fn.Symbol = hir.SymbolID(index)
				if index+1 < count {
					fn.Blocks[0].Terminator = Terminator{Kind: TerminatorCall, Callee: fn.Symbol + 1}
				}
				functions[fn.Symbol] = fn
			}
			want := ""
			if count > 64 {
				want = graphError
			}
			// A shared target is not a 65th declaration, even after the limit is reached.
			functions[0].Blocks = append(functions[0].Blocks, Block{ID: 1, Terminator: Terminator{Kind: TerminatorCall, Callee: hir.SymbolID(count - 1)}})
			checkInlineVerification(t, functions[0], functions, want)
		})
	}
	for _, test := range []struct {
		name string
		edit func(*Function)
	}{
		{"cycle", func(target *Function) { target.Blocks[0].Terminator = Terminator{Kind: TerminatorCall, Callee: 0} }},
		{"other module", func(target *Function) { target.Name = "other::f" }},
		{"unmarked", func(target *Function) { target.InlineBody = "" }},
		{"argument mismatch", func(target *Function) { target.ParameterTypes = []hir.Type{hir.TypeI32} }},
		{"non i32 result", func(target *Function) { target.ResultType = hir.TypeBool }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, target := inlineVerificationFixture(1), inlineVerificationFixture(1)
			root.Blocks[0].Terminator = Terminator{Kind: TerminatorCall, Callee: 1}
			target.Symbol = 1
			test.edit(target)
			checkInlineVerification(t, root, map[hir.SymbolID]*Function{0: root, 1: target}, graphError)
		})
	}
}

func TestInlineIdentityValidationDoesNotAllocate(t *testing.T) {
	for _, test := range []struct {
		name     string
		identity string
		valid    bool
	}{
		{"valid", hir.InlineFunctionPrefix + strings.Repeat("a7", 32), true},
		{"oversized", hir.InlineFunctionPrefix + strings.Repeat("a", 1<<20), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			allocations := testing.AllocsPerRun(10, func() {
				if validInlineIdentity(test.identity) != test.valid {
					t.Fatal("incorrect identity admission")
				}
			})
			if allocations != 0 {
				t.Fatalf("identity validation allocated %v objects", allocations)
			}
			fn := inlineVerificationFixture(1)
			fn.InlineBody = test.identity
			want := ""
			if !test.valid {
				want = "invalid versioned identity"
			}
			checkInlineVerification(t, fn, map[hir.SymbolID]*Function{0: fn}, want)
		})
	}
}

func BenchmarkInlineVerification(b *testing.B) {
	for _, test := range []struct {
		name     string
		identity string
	}{
		{"valid", hir.InlineFunctionPrefix + strings.Repeat("a7", 32)},
		{"oversized_1MiB", hir.InlineFunctionPrefix + strings.Repeat("a", 1<<20)},
	} {
		b.Run(test.name, func(b *testing.B) {
			fn := inlineVerificationFixture(1)
			fn.InlineBody = test.identity
			functions := map[hir.SymbolID]*Function{0: fn}
			b.ReportAllocs()
			for b.Loop() {
				verifyInlineFunction(fn, functions)
			}
		})
	}
}
