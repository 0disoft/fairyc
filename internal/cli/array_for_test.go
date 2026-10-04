package cli

import (
	"os"
	"strings"
	"testing"
)

func TestArrayForNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/array_for.fy")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, trap string }{{"stored-matrix", string(source), ""}}
	for _, tc := range []struct{ typ, max string }{{"u8", "255"}, {"i32", "2147483647"}, {"i64", "9223372036854775807"}, {"u32", "4294967295"}, {"usize", "18446744073709551615"}} {
		cases = append(cases, struct{ name, source, trap string }{tc.typ + "-overflow", "module main;fn i32 main(){var " + tc.typ + "[1] data=[" + tc.max + "];for(let ref mut " + tc.typ + " item in data){*item=*item+1;}return 0;}", "integer-overflow"})
	}
	cases = append(cases, struct{ name, source, trap string }{"body-trap", `module main;fn i32 main(){var i64[1] data=[7];for(let ref mut i64 item in data){*item={assert(false);9};}return 0;}`, "assertion-failed"})
	runI64ViewNativeCases(t, "array-for", cases)
}

func TestArrayForCheckRejections(t *testing.T) {
	for name, source := range map[string]string{
		"copied-write-before-break":         `fn void bad(){var i64[1] data=[1];for(let i64 item in data){data[0]=item;break;}}`,
		"mutable-owner-after-last-item-use": `fn void bad(){var i64[1] data=[1];for(let ref mut i64 item in data){*item=9;data[0]=1;}}`,
		"retained-previous-item":            `fn void bad(){var i64[2] data=[1,2];var Option<ref mut i64> saved=.none;for(let ref mut i64 item in data){switch(saved){case .some(old):{assert(*old>0);}case .none:{}};saved=.some(item);}switch(saved){case .some(old):{assert(*old>0);}case .none:{}};}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;"+source+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, "E2101") {
				t.Fatalf("expected E2101: %d %s", code, diag)
			}
		})
	}
}
