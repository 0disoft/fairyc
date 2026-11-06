package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const atlasAllocationOrdinarySource = `module main;import std::alloc as alloc;
fn void exercise(Region r){discard r.place(7);discard r.alloc_zeroed_bytes(0);discard r.copy_bytes(b"A\0\xff");}
fn void lexical(alloc::Arena owned){var alloc::Arena arena=owned;region r in ref mut arena{discard r.place(8);}}
fn i32 main(){return 0;}`

const atlasAllocationCleanupSource = `module main;import std::alloc as alloc;
resource Wrap<T>{T marker;cleanup{
 var alloc::Arena arena=switch(alloc::create()){case .ok(a):{a}case .err(_):{assert(false);return;}};
 region r in ref mut arena{discard r.place(7);discard r.alloc_zeroed_bytes(1);discard r.copy_bytes(b"x");}
}}
fn i32 main(){let value=Wrap<i32>{.marker=1};return 0;}`

type atlasAllocationProductionFixture struct {
	name, source   string
	minimum, sites int
}

func atlasAllocationProductionFixtures() []atlasAllocationProductionFixture {
	const prefix = "module main;import std::alloc as alloc;"
	const entry = "fn i32 main(){return 0;}"
	return []atlasAllocationProductionFixture{
		{"ordinary", atlasAllocationOrdinarySource, 1, 4},
		{"operand-order", prefix + `
fn void take(Result<ref i32,alloc::AllocError> p,i32 n){discard p;}
fn void reverse(i32 n,Result<ref i32,alloc::AllocError> p){discard p;}
fn void before(Region r){take(r.place(7),{assert(false);0});}
fn void after(Region r){reverse({assert(false);0},r.place(8));}
fn void trapped(Region r){discard r.copy_bytes({assert(false);b"x"});}
fn void after_statement(Region r){assert(false);discard r.alloc_zeroed_bytes(1);}` + entry, 1, 1},
		{"generic", prefix + "fn void put<T>(Region r,T n){discard r.place(n);}fn void exercise(Region r){put<i32>(r,7);put<u8>(r,8);}" + entry, 2, 2},
		{"cleanup", atlasAllocationCleanupSource, 3, 3},
		{"loop-and-result", prefix + `fn Result<i32,alloc::AllocError> exercise(Region r){var i32 n=0;while(n<3){discard r.alloc_zeroed_bytes(1);n+=1;if(n<2){continue;}break;}let value=r.place(7)?;return .ok(*value);}` + entry, 1, 2},
	}
}

func TestAtlasAllocationProductionSources(t *testing.T) {
	for _, f := range atlasAllocationProductionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, f.source)
			for v, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3} {
				t.Run(fmt.Sprintf("v%d", v+1), func(t *testing.T) {
					var diagnostic bytes.Buffer
					prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, schema)
					if (code == ExitSuccess) != (v+1 >= f.minimum) {
						t.Fatalf("independent source-family admission: %d %s", code, &diagnostic)
					}
					if v+1 < f.minimum {
						return
					}
					root, ok := selectAtlasPartition(prepared.partitions, "example.session.app")
					if !ok {
						t.Fatal("missing root partition")
					}
					count := 0
					for _, function := range root.Functions() {
						count += len(function.AllocationSites)
					}
					for _, s := range root.Specializations() {
						if s.Aggregate == nil {
							count += len(s.Function.AllocationSites)
						}
						if s.Cleanup != nil {
							count += len(s.Cleanup.AllocationSites)
						}
					}
					if count != f.sites {
						t.Fatalf("independent original allocation inventory: %d want %d", count, f.sites)
					}
				})
			}
		})
	}
}
