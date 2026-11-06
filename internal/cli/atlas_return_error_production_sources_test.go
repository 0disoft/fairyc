package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

type atlasReturnErrorProductionFixture struct {
	name, source, answer string
	minimum, sites       int
}

func atlasReturnErrorProductionFixtures() []atlasReturnErrorProductionFixture {
	const prefix = "module main;fn Result<i32,i32> read(){return .ok(7);}fn i32 main(){return 0;}"
	return []atlasReturnErrorProductionFixture{
		{"ordinary", prefix + "fn Result<i32,i32> exercise(){let a=read()?;let b=(read())?;return .ok(a+b);}", "", 1, 2},
		{"operand-order", prefix + "fn i32 take(i32 a,i32 b){return a+b;}fn Result<i32,i32> before(){return .ok(take(read()?,{assert(false);0}));}fn Result<i32,i32> after(){return .ok(take({assert(false);0},read()?));}", "", 1, 1},
		{"generic", prefix + "pub fn Result<T,i32> wrapped<T>(T value){return .ok(value);}pub fn Result<T,i32> forward<T>(T value){let n=wrapped<T>(value)?;return .ok(n);}fn void exercise(){discard forward<i32>(7);discard forward<u8>(8);}", "", 2, 2},
		{"cleanup-owner", prefix + "resource Token<T>{T value;cleanup{discard read();}}fn Result<i32,i32> exercise(){let token=Token<i32>{.value=1};let n=read()?;return .ok(n);}", "", 3, 1},
		{"branches-and-loop", prefix + "fn Result<i32,i32> exercise(bool flag){if(flag && read()?==7){discard read()?;}var usize n=0;while(n<2){discard read()?;n+=1;}return .ok(read()?);}", "", 1, 4},
		{"dependency", "module main;import answer as a;fn Result<i32,i32> exercise(){return .ok(a::read()?);}fn i32 main(){return 0;}", "module answer;fn Result<i32,i32> leaf(){return .ok(7);}pub fn Result<i32,i32> read(){let n=leaf()?;return .ok(n);}pub fn i32 value(){return 42;}", 1, 2},
	}
}

func atlasReturnErrorProductionWorkspace(t *testing.T, f atlasReturnErrorProductionFixture) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, f.source)
	if f.answer != "" {
		writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", f.answer)
	}
	return root
}

func TestAtlasReturnErrorProductionSources(t *testing.T) {
	for _, f := range atlasReturnErrorProductionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			root := atlasReturnErrorProductionWorkspace(t, f)
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
					count := 0
					for _, p := range prepared.partitions {
						for _, function := range p.Functions() {
							count += len(function.ReturnErrorSites)
						}
						for _, s := range p.Specializations() {
							if s.Aggregate == nil {
								count += len(s.Function.ReturnErrorSites)
							}
							if s.Cleanup != nil {
								count += len(s.Cleanup.ReturnErrorSites)
							}
						}
					}
					if count != f.sites {
						t.Fatalf("independent original question inventory: %d want %d", count, f.sites)
					}
				})
			}
		})
	}
}
