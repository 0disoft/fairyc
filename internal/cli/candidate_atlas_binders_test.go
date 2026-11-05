package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const atlasBinderSource = `module main;
struct Cell<T>{T value;}
struct Nest<T>{Cell<T> value;}
resource Token<T>{T value;cleanup{}}
pub fn T identity<T>(T value){return value;}
fn U choose<T,U>(T first,U second){discard first;return second;}
unsafe fn void shapes<T>(ref T one,ref mut T[] many,ptr T raw,T[2] pair,Option<T> maybe,Result<T,i32> result){}
fn i32 main(){let cell=Cell<i32>{.value=identity<i32>(3)};let token=Token<i32>{.value=4};return choose<bool,i32>(true,cell.value+token.value);}
`

func TestCandidateAtlasBinders(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasBinderSource)
	var diagnostic bytes.Buffer
	prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 {
		t.Fatalf("generic declaration oracle: %d %s", code, &diagnostic)
	}
	type fixture struct {
		name, want string
		data       []byte
		accepted   bool
	}
	var cases []fixture
	var seedHeader, seedDeclaration, seedDocument []byte
	var cellDeclaration []byte
	var seedPartition atlasmap.Partition
	for _, partition := range prepared.partitions {
		data, err := atlasmap.EncodeV3(partition, 4<<20)
		if err != nil {
			t.Fatal(err)
		}
		var document struct {
			Map struct {
				Header       json.RawMessage
				Declarations []json.RawMessage
			}
		}
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		for _, declaration := range document.Map.Declarations {
			var row struct {
				Key struct{ Module, Kind, Name string }
			}
			if err := json.Unmarshal(declaration, &row); err != nil {
				t.Fatal(err)
			}
			var full string
			for _, symbol := range partition.Symbols() {
				if symbol.Declaration.Name == row.Key.Name && symbol.Declaration.Module == row.Key.Module && string(symbol.Declaration.Kind) == row.Key.Kind {
					full = symbol.Full
				}
			}
			if full == "" {
				t.Fatal("declaration has no Go identity", row.Key.Name)
			}
			cases = append(cases, fixture{row.Key.Name, full, atlasBinderPacket(t, document.Map.Header, declaration), true})
			if bytes.Contains(declaration, []byte(`"type_parameters":["T"]`)) {
				alpha := bytes.Replace(declaration, []byte(`"type_parameters":["T"]`), []byte(`"type_parameters":["Renamed"]`), 1)
				changed := atlasRechecksum(t, bytes.Replace(data, declaration, alpha, 1))
				if _, err := atlasmap.DecodeV3(changed, 4<<20); err != nil {
					t.Fatalf("Go alpha-rename oracle: %v", err)
				}
				cases = append(cases, fixture{row.Key.Name + "-alpha", full, atlasBinderPacket(t, document.Map.Header, alpha), true})
			}
			if row.Key.Name == "identity" {
				seedHeader, seedDeclaration, seedDocument = document.Map.Header, declaration, data
				seedPartition = partition
			}
			if row.Key.Name == "Cell" {
				cellDeclaration = declaration
			}
		}
	}
	if len(seedDeclaration) == 0 || len(cellDeclaration) == 0 {
		t.Fatal("missing generic source declarations")
	}
	for _, declaration := range seedPartition.Declarations() {
		if declaration.Key.Name != "identity" {
			continue
		}
		instances, coordinates := map[string]string{}, map[string]string{}
		for _, ref := range seedPartition.Header().PackageReferences {
			instances[ref.Instance], coordinates[ref.Coordinate] = ref.UID, ref.UID
		}
		original := declaration.Type
		oldType, _ := json.Marshal(candidateTypeFromGo(original))
		for _, depth := range []int{63, 64} {
			declaration.Type = original
			for range depth {
				declaration.Type = semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeOption, Arguments: []semanticsnapshot.TypeRef{declaration.Type}}
			}
			identities, err := symbolidentity.FromDeclarations([]semanticsnapshot.Declaration{declaration}, instances, coordinates)
			if err != nil || len(identities) != 1 {
				t.Fatalf("depth identity: %v", err)
			}
			newType, _ := json.Marshal(candidateTypeFromGo(declaration.Type))
			changed := bytes.Replace(seedDeclaration, append([]byte(`"type":`), oldType...), append([]byte(`"type":`), newType...), 1)
			if bytes.Equal(changed, seedDeclaration) {
				t.Fatal("depth mutation absent")
			}
			full := identities[0].ID.Full.String()
			mutated := bytes.Replace(seedDocument, seedDeclaration, changed, 1)
			for _, symbol := range seedPartition.Symbols() {
				if symbol.Declaration == declaration.Key {
					mutated = bytes.ReplaceAll(mutated, []byte(symbol.Full), []byte(full))
					mutated = bytes.ReplaceAll(mutated, []byte(symbol.Visible), []byte(identities[0].ID.Visible.String()))
				}
			}
			_, err = atlasmap.DecodeV3(atlasRechecksum(t, atlasBinderSymbolOrder(t, mutated)), 4<<20)
			accepted := depth == 63
			if (err == nil) != accepted {
				t.Fatalf("Go depth %d acceptance: %v", depth, err)
			}
			cases = append(cases, fixture{fmt.Sprintf("type-depth-%d", depth), full, atlasBinderPacket(t, seedHeader, changed), accepted})
		}
	}
	for _, change := range []struct{ name, before, after string }{
		{"duplicate-binder", `["T"]`, `["T","T"]`},
		{"invalid-binder", `["T"]`, `["not-valid"]`},
		{"empty-binder", `["T"]`, `[""]`},
		{"binder-type", `["T"]`, `[1]`},
		{"binder-list-type", `["T"]`, `"T"`},
		{"unbound-index", `"kind":"parameter","name":"0"`, `"kind":"parameter","name":"1"`},
		{"noncanonical-index", `"kind":"parameter","name":"0"`, `"kind":"parameter","name":"00"`},
		{"negative-index", `"kind":"parameter","name":"0"`, `"kind":"parameter","name":"-1"`},
		{"overflow-index", `"kind":"parameter","name":"0"`, `"kind":"parameter","name":"18446744073709551616"`},
		{"mutable-parameter", `"package_coordinate":"","mutable":false`, `"package_coordinate":"","mutable":true`},
		{"length-parameter", `"length":0`, `"length":1`},
		{"foreign-template", `"foreign":false`, `"foreign":true`},
		{"enum-template", `"kind":"function"`, `"kind":"enum"`},
		{"unknown-member", `"type_parameters":`, `"untrusted":0,"type_parameters":`},
	} {
		changed := bytes.Replace(seedDeclaration, []byte(change.before), []byte(change.after), 1)
		if bytes.Equal(changed, seedDeclaration) {
			t.Fatal("mutation did not change declaration", change.name)
		}
		mutated := atlasRechecksum(t, bytes.Replace(seedDocument, seedDeclaration, changed, 1))
		if _, err := atlasmap.DecodeV3(mutated, 4<<20); err == nil {
			t.Fatal("Go accepted invalid declaration", change.name)
		}
		cases = append(cases, fixture{change.name, "", atlasBinderPacket(t, seedHeader, changed), false})
	}
	for _, change := range []struct{ name, before, after string }{
		{"wrong-self-name", `"name":"main::Cell"`, `"name":"main::Other"`},
		{"wrong-self-binder", `"kind":"parameter","name":"0"`, `"kind":"builtin","name":"i32"`},
	} {
		changed := bytes.Replace(cellDeclaration, []byte(change.before), []byte(change.after), 1)
		if bytes.Equal(changed, cellDeclaration) {
			t.Fatal("aggregate mutation did not change declaration", change.name)
		}
		mutated := atlasRechecksum(t, bytes.Replace(seedDocument, cellDeclaration, changed, 1))
		if _, err := atlasmap.DecodeV3(mutated, 4<<20); err == nil {
			t.Fatal("Go accepted invalid aggregate", change.name)
		}
		cases = append(cases, fixture{change.name, "", atlasBinderPacket(t, seedHeader, changed), false})
	}

	main := `module main;import project_atlas_symbols as symbols;import project_atlas_wire as wire;import project_json_tree as json;` + atlasReadInput + `
 @effects(console) fn i32 main(){var usize meter=0;let t=switch(json::parse(input(),ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 if(process::arg_count()>1){meter=16777216;}let result=symbols::derive(ref t,wire::field(ref t,0,b"header"),wire::field(ref t,0,b"declaration"),ref mut meter);
 return switch(result){case .ok(p):{if(io::write_bytes(bytes::view(ref p)).is_ok){0}else{4}}case .err(code):{discard io::write_u32(code);7}};}`
	candidateAtlasProbe(t, "project_atlas_symbols", main, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, c.data)
				if c.accepted {
					if code != 0 || out != c.want {
						t.Fatalf("identity code=%d got=%q want=%q", code, out, c.want)
					}
				} else if code != 7 || out != "7" {
					t.Fatalf("invalid declaration code=%d output=%q", code, out)
				}
			})
		}
		if code, out := candidateAtlasProcess(t, image, cases[0].data, "budget"); code != 7 || out != "3" {
			t.Fatalf("exhausted work: %d %q", code, out)
		}
		t.Logf("%d declaration/identity cases plus exhausted-work rejection", len(cases))
	})
}

func atlasBinderPacket(t *testing.T, header, declaration []byte) []byte {
	t.Helper()
	if !json.Valid(header) || !json.Valid(declaration) {
		t.Fatal("invalid declaration packet")
	}
	return []byte(fmt.Sprintf(`{"header":%s,"declaration":%s}`, header, declaration))
}

func atlasBinderSymbolOrder(t *testing.T, document []byte) []byte {
	t.Helper()
	var envelope struct {
		Map struct{ Symbols json.RawMessage }
	}
	if err := json.Unmarshal(document, &envelope); err != nil {
		t.Fatal(err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(envelope.Map.Symbols, &raw); err != nil {
		t.Fatal(err)
	}
	type row struct {
		full string
		raw  json.RawMessage
	}
	rows := make([]row, len(raw))
	for i, data := range raw {
		var value struct{ Full string }
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatal(err)
		}
		rows[i] = row{value.Full, data}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].full < rows[j].full })
	for i, r := range rows {
		raw[i] = r.raw
	}
	ordered, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Replace(document, envelope.Map.Symbols, ordered, 1)
}
