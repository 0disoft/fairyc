package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type atlasOwnerQualifiedFixture struct {
	name, source, member, uid, declaration, receiver string
}

func atlasOwnerQualifiedFixtures() []atlasOwnerQualifiedFixture {
	return []atlasOwnerQualifiedFixture{
		{"associated", `module main;struct Cell{i32 value;}fn i32 Cell::answer(){return 42;}fn i32 main(){return Cell::answer();}`, "", "example.session.app", "Cell::answer", ""},
		{"associated-parameter", `module main;struct Cell{i32 value;}fn i32 Cell::answer(i32 value){return value;}fn i32 main(){return Cell::answer(42);}`, "", "example.session.app", "Cell::answer", ""},
		{"shared", "module main;struct Cell{i32 value;}\n/// Read the cell.\nfn i32 Cell::get(ref Cell this){return this.value;}fn i32 main(){let cell=Cell{.value=42};let view=ref cell;return view.get();}", "", "example.session.app", "Cell::get", "shared"},
		{"exclusive", `module main;struct Cell{i32 value;}fn void Cell::set(ref mut Cell this,i32 value){this.value=value;}fn i32 main(){var cell=Cell{.value=0};{let view=ref mut cell;view.set(42);}return cell.value;}`, "", "example.session.app", "Cell::set", "exclusive"},
		{"value-record", `module main;struct Cell{i32 value;}fn i32 Cell::take(Cell this){return this.value;}fn i32 main(){let cell=Cell{.value=42};return Cell::take(cell);}`, "", "example.session.app", "Cell::take", "value"},
		{"value-resource", `module main;resource Token{i32 value;cleanup{}}fn i32 Token::take(Token this){return this.value;}fn i32 main(){let token=Token{.value=42};return Token::take(token);}`, "", "example.session.app", "Token::take", "value"},
		{"imported-shared", `module main;import answer as a;fn i32 main(){let cell=a::Cell{.value=42};let view=ref cell;return view.get();}`, `module answer;pub struct Cell{i32 value;}pub fn i32 Cell::get(ref Cell this){return this.value;}`, "example.session.answer", "Cell::get", "shared"},
		{"unused-associated", `module main;struct Cell{i32 value;}fn i32 Cell::answer(){return 42;}fn i32 main(){return 42;}`, "", "example.session.app", "Cell::answer", ""},
	}
}

func atlasOwnerQualifiedWorkspace(t *testing.T, fixture atlasOwnerQualifiedFixture) string {
	t.Helper()
	root := writeWorkspaceSessionFixture(t, fixture.source)
	if fixture.member != "" {
		writeWorkspaceFile(t, root, "libs/answer/src/answer.fy", fixture.member)
	}
	return root
}

func TestAtlasOwnerQualifiedSourceOracle(t *testing.T) {
	for _, fixture := range atlasOwnerQualifiedFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			root := atlasOwnerQualifiedWorkspace(t, fixture)
			var diagnostic bytes.Buffer
			prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
			if code != ExitSuccess {
				t.Fatalf("owner-qualified source oracle: %d %s", code, &diagnostic)
			}
			owners, calls := 0, 0
			for _, partition := range prepared.partitions {
				encoded, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				partition, err = atlasmap.DecodeV3(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				again, err := atlasmap.EncodeV3(partition, 4<<20)
				if err != nil || !bytes.Equal(encoded, again) {
					t.Fatalf("owner-qualified codec changed facts: %v", err)
				}
				if partition.Header().PackageUID == fixture.uid {
					index, err := atlasmap.NewQueryIndexV3(partition, prepared.expectations[partition.Header().PackageInstance])
					if err != nil {
						t.Fatal(err)
					}
					catalog, err := index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Name: fixture.declaration, Kinds: []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationFunction}, Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 64, MaxSelectionBytes: 4 << 20}})
					if err != nil || !catalog.Complete || catalog.Total != 1 || len(catalog.Symbols) != 1 || catalog.Symbols[0].Name != fixture.declaration {
						t.Fatalf("qualified exact-name catalog: %+v %v", catalog, err)
					}
				}
				for _, declaration := range partition.Declarations() {
					if declaration.Owner == nil {
						continue
					}
					owners++
					if partition.Header().PackageUID != fixture.uid || declaration.Key.Name != fixture.declaration || declaration.Receiver != fixture.receiver || declaration.Owner.Kind != semanticsnapshot.TypeNominal || declaration.Owner.PackageCoordinate == "" {
						t.Fatal("owner/receiver/declaration identity differs", declaration)
					}
					ownerName, _, ok := strings.Cut(fixture.declaration, "::")
					if !ok || declaration.Owner.Name != declaration.Key.Module+"::"+ownerName {
						t.Fatal("owner lost its exact defining module", declaration)
					}
					if fixture.receiver != "" {
						if len(declaration.Parameters) == 0 || declaration.Parameters[0].Name != "this" {
							t.Fatal("receiver parameter disappeared", declaration)
						}
					}
				}
				for _, function := range partition.Functions() {
					for _, call := range function.Calls {
						if call.Target.Name == fixture.declaration {
							calls++
							if call.TargetKind != semanticsnapshot.CallTargetResolved || call.Certainty != semanticsnapshot.CertaintyExact {
								t.Fatal("owner-qualified call became opaque", call)
							}
						}
					}
				}
			}
			wantCalls := 1
			if fixture.name == "unused-associated" {
				wantCalls = 0
			}
			if owners != 1 || calls != wantCalls {
				t.Fatalf("owner-qualified inventory: owners=%d calls=%d want calls=%d", owners, calls, wantCalls)
			}
			if _, err := atlasmap.NewWorkspaceCallIndexV3(prepared.partitions, prepared.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 128}); err != nil {
				t.Fatal("owner-qualified complete join", err)
			}
		})
	}
}
