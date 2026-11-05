package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestAtlasOwnerQualifiedDistinctNamespaces(t *testing.T) {
	for _, ordinary := range []bool{false, true} {
		t.Run(fmt.Sprint("ordinary=", ordinary), func(t *testing.T) {
			source := `module main;struct Cell{i32 value;}struct Other{i32 value;}
fn i32 Cell::get(ref Cell this){return this.value;}
fn i32 Other::get(ref Other this){return this.value;}`
			call := ""
			if ordinary {
				source += "fn i32 get(){return 1;}"
				call = "+get()"
			}
			source += `fn i32 main(){let a=Cell{.value=20};let b=Other{.value=21};let left=ref a;let right=ref b;return left.get()+right.get()` + call + ";}"
			root := writeWorkspaceSessionFixture(t, source)
			if code, _, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess {
				t.Fatalf("ordinary owner namespaces must be valid: %d %s", code, diagnostic)
			}
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("owner namespaces lost during v3 preparation: %d %s", code, &diagnostic)
			}
			want := map[string]bool{"Cell::get": false, "Other::get": false}
			if ordinary {
				want["get"] = false
			}
			for _, partition := range prepared.partitions {
				if partition.Header().PackageUID != "example.session.app" {
					continue
				}
				for _, symbol := range partition.Symbols() {
					if _, ok := want[symbol.Declaration.Name]; ok {
						if want[symbol.Declaration.Name] {
							t.Fatal("duplicate qualified function identity")
						}
						want[symbol.Declaration.Name] = true
					}
				}
			}
			for name, seen := range want {
				if !seen {
					t.Fatal("missing distinct function identity", name)
				}
			}
			index, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128})
			if err != nil || len(index.Calls()) != len(want) {
				t.Fatalf("distinct same-leaf calls: index=%v err=%v", index, err)
			}
		})
	}
}
