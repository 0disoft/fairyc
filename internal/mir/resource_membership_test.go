package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func resourceMembershipProgram(depth int) Program {
	resource := hir.ParseType("main::MembershipResource")
	plain := hir.TypeI32
	for range depth {
		plain = hir.Result(plain, plain)
	}
	return Program{
		ResourceTypes: []ResourceType{{Type: resource, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol}},
		Functions: []Function{{Symbol: 1, Name: "main::membership", ResultType: hir.TypeVoid, Entry: 0,
			Resources: []ResourceLocal{{Symbol: 2, Type: hir.Result(plain, resource)}},
			Blocks:    []Block{{ID: 0, Terminator: Terminator{Kind: TerminatorReturn, UnwindSuccessor: NoBlock}}},
		}},
	}
}

func TestContainsDefinedResourceTaggedGraph(t *testing.T) {
	resource := hir.ParseType("main::MembershipResource")
	missing := hir.ParseType("main::MissingMembershipResource")
	definitions := map[hir.Type]*ResourceType{resource: {Type: resource}, missing: nil}
	plain := hir.TypeI32
	for range 80 {
		plain = hir.Option(hir.Result(plain, plain))
	}
	for _, test := range []struct {
		name string
		typ  hir.Type
		want bool
	}{
		{"direct", resource, true},
		{"nil-definition", missing, false},
		{"shared-negative", plain, false},
		{"resource-after-shared-negative", hir.Result(plain, hir.Option(resource)), true},
		{"success-resource", hir.Result(resource, plain), true},
		{"borrow-is-not-ownership", hir.SharedBorrow(resource), false},
		{"pointer-is-not-ownership", hir.RawPointer(resource, true), false},
		{"array-is-not-a-tagged-wrapper", hir.FixedArray(resource, 1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := containsDefinedResource(test.typ, definitions); got != test.want {
				t.Fatalf("got %t, want %t", got, test.want)
			}
		})
	}
	wrapped := hir.Option(missing)
	for _, defined := range []bool{false, true, false} {
		definitions[missing] = nil
		if defined {
			definitions[missing] = &ResourceType{Type: missing}
		}
		if containsDefinedResource(wrapped, definitions) != defined {
			t.Fatal("search reused stale resource definitions")
		}
	}
}

func TestVerifySharedResourceMembership(t *testing.T) {
	if failures := Verify(resourceMembershipProgram(80)); len(failures) != 0 {
		t.Fatalf("resource after shared non-resource branch rejected: %d failures", len(failures))
	}
	program := resourceMembershipProgram(0)
	program.Functions[0].Resources[0].Type = hir.Option(hir.TypeI32)
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeAggregateUse {
		t.Fatalf("resource-free local accepted or diagnostic changed: %+v", failures)
	}
}

func BenchmarkVerifySharedResourceMembership(b *testing.B) {
	for _, depth := range []int{0, 16, 20} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			program := resourceMembershipProgram(depth)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if failures := Verify(program); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}
