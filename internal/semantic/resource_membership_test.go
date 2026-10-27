package semantic

import (
	"context"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestResourceMembershipSharedTaggedGraph(t *testing.T) {
	resource := hir.ParseType("main::ResourceMembership")
	// Semantic classification owns registered identities, even before their info
	// is populated; MIR independently requires a non-nil executable definition.
	a := &analyzer{resourceTypes: map[hir.Type]*resourceInfo{resource: nil}}
	plain := hir.TypeI32
	for range 80 {
		plain = hir.Option(hir.Result(plain, plain))
	}
	for _, test := range []struct {
		name string
		typ  hir.Type
		want bool
	}{
		{"registered", resource, true},
		{"negative-shared", plain, false},
		{"failure-resource", hir.Result(plain, hir.Option(resource)), true},
		{"success-resource", hir.Result(resource, plain), true},
		{"borrow", hir.SharedBorrow(resource), false},
		{"pointer", hir.RawPointer(resource, true), false},
		{"array", hir.FixedArray(resource, 1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := a.isResource(test.typ); got != test.want {
				t.Fatalf("got %t, want %t", got, test.want)
			}
		})
	}
	wrapped := hir.Option(resource)
	delete(a.resourceTypes, resource)
	if a.isResource(wrapped) {
		t.Fatal("classification reused removed identity")
	}
	a.resourceTypes[resource] = nil
	if !a.isResource(wrapped) {
		t.Fatal("classification missed re-registered identity")
	}
}

func BenchmarkResourceMembershipFlow(b *testing.B) {
	for _, depth := range []int{0, 12, 16} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			value := hir.TypeI32
			for range depth {
				value = hir.Result(value, value)
			}
			function := &ast.Function{Parameters: make([]ast.Parameter, 32), Body: &ast.Block{}}
			a := &analyzer{ctx: context.Background(),
				resourceTypes: map[hir.Type]*resourceInfo{hir.ParseType("main::Other"): {}},
				parameter:     make(map[*ast.Parameter]*symbol),
				functions:     map[*ast.Function]*symbol{function: {typ: hir.TypeVoid, effectsValid: true}},
			}
			for i := range function.Parameters {
				a.parameter[&function.Parameters[i]] = &symbol{typ: value}
			}
			module := &moduleScope{file: File{Path: "main.fy"}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				a.resourceFlowFunction(module, function)
				if len(a.diagnostics) != 0 || len(a.resourceStates) != 0 {
					b.Fatal("copy parameters acquired resource cleanup obligations")
				}
			}
		})
	}
}
