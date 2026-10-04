package cli

import (
	"os"
	"strings"
	"testing"
)

func TestBoolViewNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/bool_views.fy")
	if err != nil {
		t.Fatal(err)
	}
	base := `module main;fn i32 main(){var bool[2] flags=[true,false];let ref mut bool[] view=ref mut flags;`
	cases := []struct{ name, source, trap string }{
		{"matrix", string(source), ""},
		{"sdk-vector", boolViewSDKSource, ""},
	}
	for _, tc := range []struct{ name, body string }{
		{"index-end", "let bool unused=view[2];"},
		{"store-end", "view[2]=true;"},
		{"index-huge", "let bool unused=view[18446744073709551615];"},
		{"empty-index", "let bool unused=view[2..][0];"},
		{"reversed", "let ref mut bool[] child=view[2..1];"},
		{"slice-past-end", "let ref mut bool[] child=view[..3];"},
	} {
		cases = append(cases, struct{ name, source, trap string }{tc.name, base + tc.body + "return 0;}", "bounds"})
	}
	runI64ViewNativeCases(t, "bool-view", cases)
}
func TestBoolViewCheckRejections(t *testing.T) {
	for name, tc := range map[string]struct{ source, code string }{
		"numeric-store": {`fn void bad(ref mut bool[] flags){flags[0]=1;}`, "E1006"},
		"shared-store":  {`fn void bad(ref bool[] flags){for(let ref bool item in flags){*item=false;}}`, "E1008"},
		"owner-write":   {`fn void bad(){var bool[1] flags=[true];for(let ref bool item in flags){assert(*item);flags[0]=false;break;}}`, "E2101"},
		"local-escape":  {`fn ref bool bad(){let bool[1] flags=[true];for(let ref bool item in flags){return item;}return ref flags[0];}`, "E2102"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, "module main;"+tc.source+"fn i32 main(){return 0;}")
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, tc.code) {
				t.Fatalf("expected %s: %d %s", tc.code, code, diag)
			}
		})
	}
}

const boolViewSDKSource = `module main;import std::vec as vec;
fn bool identity<T>(T value){return value==value;}
fn i32 main(){
 var vec::Vec<bool> owner=vec::empty<bool>();
 assert(vec::push<bool>(ref mut owner,true).is_ok);
 assert(vec::push<bool>(ref mut owner,false).is_ok);
 let ref mut bool[] view=vec::view_mut<bool>(ref mut owner);
 for(let ref mut bool item in view){*item=!*item;}
 assert(!view[0] && view[1]);
 for(let ref bool item in view){assert(identity<bool>(*item));}
 let bool[1] data=[true];for(let bool item in data){assert(identity<bool>(item));}
 assert(vec::clear<bool>(ref mut owner)==2);return 0;}`

func TestBoolViewCache(t *testing.T) {
	checkStructVecViewCache(t, boolViewSDKSource)
}
