package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const candidatePackRelationsMain = `module main;import project_pack_relations as relations;import project_pack_relation_order as ordering;import project_atlas_current as current;import project_atlas_validate as validate;import project_catalog_json as w;import project_json_tree as json;import project_atlas_wire as wire;import std::vec as v;` + atlasReadInput + `
fn Result<bytes::Buffer,u32> line(ref u8[] raw,ref mut usize cursor){let start=*cursor;while(*cursor<raw.len && raw[*cursor]!=10){*cursor+=1;}if(*cursor==raw.len){return .err(9);}let result=switch(bytes::copy(raw[start..*cursor])){case .ok(p):{p}case .err(_):{return .err(2);}};*cursor+=1;return .ok(result);}
fn Result<bytes::Buffer,u32> query(ref u8[] raw){var usize cursor=0;var usize meter=0;let q=switch(json::parse(line(raw,ref mut cursor)?,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let data=line(raw,ref mut cursor)?;let expected=line(raw,ref mut cursor)?;if(cursor!=raw.len){return .err(9);}let version=wire::integer(ref q,0,b"version");
 let ready=if(version==2){current::read_v2(data,expected,ref mut meter)?}else if(version==3){current::read_generics(data,expected,ref mut meter)?}else{current::read(data,expected,ref mut meter)?};
 let mode=wire::string(ref q,0,b"mode");if(bytes::equal(mode,b"meter")){meter=wire::integer(ref q,0,b"meter");}let before=meter;let p=current::partition(ref ready);
 var rows=relations::build(ref ready,ref mut meter)?;
 if(bytes::equal(mode,b"reverse")){var usize i=0;let n=v::len<relations::Relation>(ref rows);while(i<n/2){let a=*v::at<relations::Relation>(ref rows,i);let b=*v::at<relations::Relation>(ref rows,n-i-1);{let slot=v::at_mut<relations::Relation>(ref mut rows,i);*slot=b;}{let slot=v::at_mut<relations::Relation>(ref mut rows,n-i-1);*slot=a;}i+=1;}}
 if(bytes::equal(mode,b"duplicate") && v::len<relations::Relation>(ref rows)>0){let order=ordering::sort(p,ref rows,ref mut meter)?;let first=*v::at<relations::Relation>(ref rows,*v::at<usize>(ref order,0));if(!v::push<relations::Relation>(ref mut rows,first).is_ok){return .err(2);}}
 let order=ordering::sort(p,ref rows,ref mut meter)?;var out=bytes::empty();discard w::raw(ref mut out,b"[",ref mut meter)?;var usize i=0;
 while(i<v::len<usize>(ref order)){if(i>0){discard w::raw(ref mut out,b",",ref mut meter)?;}let r=relations::encode(p,*v::at<relations::Relation>(ref rows,*v::at<usize>(ref order,i)),ref mut meter)?;discard w::raw(ref mut out,bytes::view(ref r),ref mut meter)?;i+=1;}discard w::raw(ref mut out,b"]",ref mut meter)?;
 if(bytes::equal(mode,b"measure")){var result=bytes::empty();if(!bytes::append_usize(ref mut result,meter-before).is_ok){return .err(2);}return .ok(result);}return .ok(out);
}
@effects(console) fn i32 main(){let data=input();return switch(query(bytes::view(ref data))){case .ok(p):{if(io::write_bytes(bytes::view(ref p)).is_ok){0}else{4}}case .err(_):{7}};}`

func candidatePackRelationFixtures(t *testing.T) []candidateCatalogFixture {
	t.Helper()
	fixtures := candidatePackNodeFixtures(t)
	source := `module main;import std::alloc as alloc;
unsafe extern C fn i32 opaque(i32 n);
fn i32 foreign(i32 n){return unsafe{opaque(n)};}
fn void self(){self();}
struct Cell{i32 value;}
fn ref i32 borrow(ref Cell value){return ref value.value;}
fn Result<i32,i32> checked(bool ok){return if(ok){.ok(1)}else{.err(2)};}
fn Result<i32,i32> propagate(bool ok){let i32 value=checked(ok)?;return .ok(value);}
fn void placement(alloc::Arena owned){var alloc::Arena arena=owned;region scratch in ref mut arena{discard scratch.place(42);}}
resource Item{i32 value;cleanup{discard this.value;}}
fn i32 main(){let item=Item{.value=1};var i32 x=item.value;let ref mut i32 view=ref mut x;*view=2;discard checked(true);return foreign(x);}`
	for v, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3} {
		root := writeWorkspaceSessionFixture(t, source)
		var diagnostics bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostics, schema)
		if code != 0 {
			t.Fatalf("relation source v%d: %d %s", v+1, code, &diagnostics)
		}
		for _, p := range prepared.partitions {
			if p.Header().PackageUID != "example.session.app" {
				continue
			}
			data, err := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[v](p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			e := prepared.expectations[p.Header().PackageInstance]
			index, err := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[v](p, e)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, candidateCatalogFixture{fmt.Sprintf("relations-v%d", v+1), p.Header().PackageUID, v + 1, data, candidateAtlasCurrent(t, e), index})
		}
	}
	return fixtures
}

func TestCandidatePackRelations(t *testing.T) {
	fixtures := candidatePackRelationFixtures(t)
	candidateAtlasProbe(t, "project_pack_relation_order", candidatePackRelationsMain, func(t *testing.T, image string) {
		processes, comparisons := 0, 0
		seen := map[atlasmap.RelationKind]bool{}
		facts := map[string]bool{}
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				catalog, err := f.index.CatalogSymbols(atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 65536, MaxSelectionBytes: 4 << 20}})
				if err != nil || !catalog.Complete {
					t.Fatalf("relation oracle catalog: %v", err)
				}
				wantRelations := []atlasmap.QueryRelation{}
				for _, s := range catalog.Symbols {
					r, err := f.index.Show(s.Symbol.Full, atlasmap.RelationRequest{Direction: atlasmap.DirectionOutgoing, Relations: atlasmap.AllRelationKinds(), Budget: atlasmap.QueryBudget{MaxRecords: 65536, MaxRelations: 65536, MaxSelectionBytes: 4 << 20}})
					if err != nil {
						t.Fatal(err)
					}
					for _, relation := range r.Relations {
						seen[relation.Kind] = true
						if relation.SourceSpecialization != "" {
							facts["source-specialization"] = true
						}
						if relation.SourceBody == "cleanup" {
							facts["cleanup"] = true
						}
						if relation.Target != nil {
							if relation.Target.Symbol == "" {
								facts["unjoined"] = true
							}
							if relation.Target.Specialization != "" {
								facts["target-specialization"] = true
							}
						}
						if relation.ABI == "C" && relation.LinkName != "" {
							facts["foreign-link"] = true
						}
						if relation.Escapes {
							facts["escaping-borrow"] = true
						}
						if relation.Span == nil && relation.Detail != "" {
							facts["spanless-detail"] = true
						}
					}
					wantRelations = append(wantRelations, r.Relations...)
				}
				want, err := json.Marshal(wantRelations)
				if err != nil {
					t.Fatal(err)
				}
				comparisons += len(wantRelations)
				run := func(mode string, meter int) (int, string) {
					t.Helper()
					processes++
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
				for _, mode := range []string{"all", "reverse", "duplicate"} {
					expected := want
					if mode == "duplicate" && len(wantRelations) > 0 {
						extra := append([]atlasmap.QueryRelation{wantRelations[0]}, wantRelations...)
						expected, err = json.Marshal(extra)
						if err != nil {
							t.Fatal(err)
						}
					}
					if code, got := run(mode, 0); code != 0 || got != string(expected) {
						t.Fatalf("%s relations: %d\n got=%s\nwant=%s", mode, code, got, expected)
					}
				}
				code, text := run("measure", 0)
				cost, err := strconv.Atoi(text)
				if code != 0 || err != nil || cost <= 0 || cost >= 16<<20 {
					t.Fatalf("invalid work: %d %q %v", code, text, err)
				}
				for _, extra := range []int{-1, 0, 1} {
					code, got := run("meter", (16<<20)-cost-extra)
					if extra < 0 && (code != 7 || got != "") || extra >= 0 && (code != 0 || got != string(want)) {
						t.Fatalf("relation work boundary %d: %d %q", extra, code, got)
					}
				}
			})
		}
		for _, kind := range atlasmap.AllRelationKinds() {
			if !seen[kind] {
				t.Errorf("no independent actual-source observation of relation kind %s", kind)
			}
		}
		for _, name := range []string{"source-specialization", "cleanup", "unjoined", "target-specialization", "foreign-link", "escaping-borrow", "spanless-detail"} {
			if !facts[name] {
				t.Errorf("missing actual-source relation fact: %s", name)
			}
		}
		t.Logf("%d native relation observations; %d independently compared relations", processes, comparisons)
	})
}
