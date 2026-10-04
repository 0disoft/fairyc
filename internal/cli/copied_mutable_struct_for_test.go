package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const copiedMutableStructForBase = `module main;import std::vec as vec;struct Token{i64 value;}fn i32 main(){var vec::Vec<Token> owner=vec::empty<Token>();assert(vec::push<Token>(ref mut owner,Token{.value=9223372036854775807}).is_ok);let ref mut Token[] view=vec::view_mut<Token>(ref mut owner);ACTION return 0;}`

func TestCopiedMutableStructForNative(t *testing.T) {
	source, err := os.ReadFile("../compiler/testdata/copied_mutable_struct_for.fy")
	if err != nil {
		t.Fatal(err)
	}
	runI64ViewNativeCases(t, "copied-mutable-struct-for", []struct{ name, source, trap string }{
		{"stored-matrix", string(source), ""},
		{"singleton", strings.Replace(copiedMutableStructForBase, "ACTION", `var usize count=0;for(let Token item in view){view[0].value=9;assert(item.value==9223372036854775807);count+=1;}assert(count==1);`, 1), ""},
		{"checked-overflow", strings.Replace(copiedMutableStructForBase, "ACTION", `for(let Token item in view){view[0].value=item.value+1;}`, 1), "integer-overflow"},
	})
}
func TestCopiedMutableStructForCheckRejections(t *testing.T) {
	for _, tc := range []struct{ name, action, code string }{
		{"owner-clear-break", `for(let Token item in view){discard vec::clear<Token>(ref mut owner);break;}`, "E2101"},
		{"owner-reserve-noop", `for(let Token item in view){discard vec::reserve<Token>(ref mut owner,0);continue;}`, "E2101"},
		{"owner-move", `for(let Token item in view){let vec::Vec<Token> moved=owner;assert(item.value>0);}`, "E2105"},
		{"item-store", `for(let Token item in view){item.value=9;}`, "E1008"},
		{"shared-live-parent", `for(let Token item in view){let ref Token alias=ref view[0];view[0].value=9;assert(alias.value>0);}`, "E2101"},
		{"copy-escape", ``, "E2102"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(copiedMutableStructForBase, "ACTION", tc.action, 1)
			if tc.name == "copy-escape" {
				source = `module main;struct Token{i64 value;}fn Option<ref Token> bad(ref mut Token[] view){for(let Token item in view){return .some(ref item);}return .none;}fn i32 main(){return 0;}`
			}
			root := t.TempDir()
			writeVecI64ViewWorkspace(t, root, source)
			code, _, diag := runForTest([]string{"check"}, root)
			if code != ExitSourceDiagnostics || !strings.Contains(diag, tc.code) {
				t.Fatalf("expected %s: %d %s", tc.code, code, diag)
			}
		})
	}
}

func TestCopiedMutableStructExampleCheck(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join("..", "..", "examples", "copied_mutable_struct_quickstart")
	manifest, err := os.ReadFile(filepath.Join(example, "fairy.toml"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", string(manifest))
	for _, name := range []string{"main.fy", "records.fy"} {
		source, err := os.ReadFile(filepath.Join(example, "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, filepath.Join("src", name), string(source))
	}
	code, _, diag := runForTest([]string{"check"}, root)
	if code != 0 {
		t.Fatalf("array and struct-view example: %d %s", code, diag)
	}
}
