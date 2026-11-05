package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const candidatePackNodeMain = `module main;import project_pack_node as node;import project_atlas_current as current;import project_atlas_validate as validate;import project_catalog_json as w;import project_json_tree as json;import project_atlas_wire as wire;` + atlasReadInput + `
fn Result<bytes::Buffer,u32> line(ref u8[] raw,ref mut usize cursor){let start=*cursor;while(*cursor<raw.len && raw[*cursor]!=10){*cursor+=1;}if(*cursor==raw.len){return .err(9);}let result=switch(bytes::copy(raw[start..*cursor])){case .ok(p):{p}case .err(_):{return .err(2);}};*cursor+=1;return .ok(result);}
fn Result<bytes::Buffer,u32> query(ref u8[] raw){var usize cursor=0;var usize meter=0;let q=switch(json::parse(line(raw,ref mut cursor)?,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let data=line(raw,ref mut cursor)?;let expected=line(raw,ref mut cursor)?;if(cursor!=raw.len){return .err(9);}let version=wire::integer(ref q,0,b"version");
 let ready=if(version==2){current::read_v2(data,expected,ref mut meter)?}else if(version==3){current::read_generics(data,expected,ref mut meter)?}else{current::read(data,expected,ref mut meter)?};
 let mode=wire::string(ref q,0,b"mode");if(bytes::equal(mode,b"invalid")){return node::encode(ref ready,validate::symbol_count(current::partition(ref ready)),ref mut meter);}if(bytes::equal(mode,b"meter")){meter=wire::integer(ref q,0,b"meter");}let before=meter;
 var out=bytes::empty();discard w::raw(ref mut out,b"[",ref mut meter)?;var usize i=0;while(i<validate::symbol_count(current::partition(ref ready))){if(i>0){discard w::raw(ref mut out,b",",ref mut meter)?;}let n=node::encode(ref ready,i,ref mut meter)?;discard w::raw(ref mut out,bytes::view(ref n),ref mut meter)?;i+=1;}discard w::raw(ref mut out,b"]",ref mut meter)?;
 if(bytes::equal(mode,b"measure")){var result=bytes::empty();if(!bytes::append_usize(ref mut result,meter-before).is_ok){return .err(2);}return .ok(result);}return .ok(out);
}
@effects(console) fn i32 main(){let data=input();return switch(query(bytes::view(ref data))){case .ok(p):{if(io::write_bytes(bytes::view(ref p)).is_ok){0}else{4}}case .err(_):{7}};}`

func candidatePackNodeFixtures(t *testing.T) []candidateCatalogFixture {
	t.Helper()
	fixtures := candidateCatalogFixtures(t)
	source := `module main;import answer as a;
@repr(C) pub struct Pair{i32[2] values;bool enabled;}
pub enum Choice{none;value{i32 value;};}
pub resource Item{i32 value;cleanup{discard this.value;}}
pub fn void signature(ref mut i32 value,ref u8[] text,u8[2] data){discard text;discard data;*value=1;}
pub fn i32 Pair::get(ref Pair this){return this.values[0];}
fn i32 main(){let r=Item{.value=a::value()};return r.value;}`
	for v, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3} {
		text := source
		if v == 2 {
			text = strings.Replace(text, "return r.value;", "let token=make<i32>(r.value);discard make<u32>(7);return token.payload.value;", 1)
			text += `pub struct Box<T>{T value;}
pub resource Token<T>{Box<T> payload;cleanup{let T copy=this.payload.value;let ref T view=ref copy;discard finish<T>(*view);}}
fn T finish<T>(T value){return value;}
fn Token<T> make<T>(T value){return Token<T>{.payload=Box<T>{.value=value}};}`
		}
		root := writeWorkspaceSessionFixture(t, text)
		var diagnostics bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostics, schema)
		if code != 0 {
			t.Fatalf("rich node source v%d: %d %s", v+1, code, &diagnostics)
		}
		for _, p := range prepared.partitions {
			data, err := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[v](p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			e := prepared.expectations[p.Header().PackageInstance]
			index, err := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[v](p, e)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, candidateCatalogFixture{fmt.Sprintf("rich-v%d-%s", v+1, p.Header().PackageUID), p.Header().PackageUID, v + 1, data, candidateAtlasCurrent(t, e), index})
		}
	}
	return fixtures
}

func TestCandidatePackNode(t *testing.T) {
	fixtures := candidatePackNodeFixtures(t)
	candidateAtlasProbe(t, "project_pack_node", candidatePackNodeMain, func(t *testing.T, image string) {
		observations, nodes := 0, 0
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				catalog, err := f.index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 65536, MaxSelectionBytes: 4 << 20}})
				if err != nil || !catalog.Complete {
					t.Fatalf("independent node catalog: %+v %v", catalog, err)
				}
				var wantNodes []atlasmap.QueryNode
				for _, entry := range catalog.Symbols {
					subject, err := f.index.ResolveAttributionSubject(entry.Symbol.Full)
					if err != nil {
						t.Fatal(err)
					}
					wantNodes = append(wantNodes, subject.Subject)
				}
				want, err := json.Marshal(wantNodes)
				if err != nil {
					t.Fatal(err)
				}
				nodes += len(wantNodes)
				run := func(mode string, meter int) (int, string) {
					t.Helper()
					observations++
					control, err := json.Marshal(map[string]any{"version": f.version, "mode": mode, "meter": meter})
					if err != nil {
						t.Fatal(err)
					}
					packet := append(control, '\n')
					packet = append(packet, f.data...)
					packet = append(packet, '\n')
					packet = append(packet, f.expected...)
					packet = append(packet, '\n')
					return candidateAtlasProcess(t, image, packet)
				}
				if code, got := run("all", 0); code != 0 || got != string(want) {
					t.Fatalf("node projection: code=%d\n got=%s\nwant=%s", code, got, want)
				}
				if code, got := run("invalid", 0); code != 7 || got != "" {
					t.Fatalf("out-of-range node published: %d %q", code, got)
				}
				code, text := run("measure", 0)
				cost, err := strconv.Atoi(text)
				if code != 0 || err != nil || cost <= 0 || cost >= 16<<20 {
					t.Fatalf("invalid projection cost: %d %q %v", code, text, err)
				}
				for _, extra := range []int{-1, 0, 1} {
					code, got := run("meter", (16<<20)-cost-extra)
					if extra < 0 && (code != 7 || got != "") || extra >= 0 && (code != 0 || got != string(want)) {
						t.Fatalf("projection work boundary %d: %d %q", extra, code, got)
					}
				}
			})
		}
		t.Logf("%d native node observations; %d independently compared nodes", observations, nodes)
	})
}
