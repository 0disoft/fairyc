package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// The graph lane is compiled from source independently by Go, then extracted
// through the existing production identity APIs. No oracle admission is widened.
func stateGraphInterfaceOracle(t *testing.T, text string, graph bool) string {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("ledger.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	file := semantic.File{Package: hir.DefaultPackage, Path: "ledger.fy", Module: parsed.AST}
	var checked semantic.Result
	if graph {
		result, e := compiler.CheckExperimentalStateGraph(file)
		if e != nil || !result.Verified || len(result.StateTransactions) != 1 {
			t.Fatalf("Go graph compiler: %v %+v", e, result)
		}
		checked = result.Semantic
	} else {
		checked, err = semantic.AnalyzePackages(semantic.PackageInput{Files: []semantic.File{file}})
	}
	if err != nil || checked.CompletedThrough != semantic.PhaseFlow || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
		t.Fatalf("source analysis: %v %+v", err, checked)
	}
	coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := moduleinterface.Extract(checked.HIR, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := moduleinterface.Encode(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := moduleinterface.Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	generic, err := moduleinterface.ComputeGeneric(descriptor, nil)
	if err != nil {
		t.Fatal(err)
	}
	// ComputeGeneric publishes a hash, so reconstruct its documented bytes and
	// cross-check them with that API rather than treating a local digest as truth.
	var sourceBytes bytes.Buffer
	for _, s := range []string{moduleinterface.GenericHashEncoding, moduleinterface.HashEncoding} {
		if err := binary.Write(&sourceBytes, binary.BigEndian, uint64(len(s))); err != nil {
			t.Fatal(err)
		}
		sourceBytes.WriteString(s)
	}
	sourceBytes.Write(hash[:])
	if err := binary.Write(&sourceBytes, binary.BigEndian, uint64(0)); err != nil {
		t.Fatal(err)
	}
	if digest := sha256.Sum256(sourceBytes.Bytes()); moduleinterface.GenericHash(digest) != generic {
		t.Fatal("source encoding differs from production identity")
	}
	// Prefix ends immediately before the module count. These fixtures have one
	// module and no exported items, whose complete trailing encoding is fixed.
	if len(descriptor.Modules) != 1 || len(descriptor.Modules[0].Items) != 0 {
		t.Fatal("unexpected public fixture")
	}
	prefixLen := len(encoded) - (8 + 8 + len("ledger") + 8)
	return fmt.Sprintf("%x\n%x\n%s\n%x\n%s\n", encoded[:prefixLen], encoded, hash.String(), sourceBytes.Bytes(), generic.String())
}

const stateGraphInterfaceProbe = `module main;
import module_state_declarations as declarations;import module_state_layout as state;import module_state_interface as prefix;
import module_public_interface as public;import module_public_interface_model as model;import module_generic_interface as generic;
import module_catalog as catalog;import scanner as scanner;import syntax as syntax;import literal_types as literals;
import std::bytes as bytes;import std::vec as v;import std::io as io;
fn syntax::Parsed parse(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parse(raw);}};let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parse(raw);}};let p=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return parse(raw);}};assert(syntax::problem_count(ref p)==0);return p;}
fn model::Descriptor descriptor(bool bad){let names=switch(bytes::copy(b"ledger")){case .ok(p):{p}case .err(_):{assert(false);return descriptor(bad);}};var v::Vec<model::Module> modules=v::empty<model::Module>();assert(v::push<model::Module>(ref mut modules,model::Module{.path=model::Text{.start=0,.end=if(bad){7}else{6}},.items=model::Range{.first=0,.count=0}}).is_ok);return model::assemble(names,v::empty<model::Text>(),v::empty<model::Field>(),v::empty<model::Variant>(),v::empty<model::Item>(),modules);}
fn generic::Templates templates(usize mode){var bytes::Buffer names=bytes::empty();var v::Vec<generic::Template> rows=v::empty<generic::Template>();if(mode!=0){assert(bytes::append(ref mut names,b"ledgerexported0000000000000000000000000000000000000000000000000000000000000000").is_ok);assert(v::push<generic::Template>(ref mut rows,generic::Template{.module_path=model::Text{.start=0,.end=if(mode==2){5}else{6}},.name=model::Text{.start=6,.end=14},.closure=model::Text{.start=14,.end=78}}).is_ok);}return generic::assemble(names,rows);}
// Relocate the selected owner, add malformed unrelated owners, or damage just
// the selected facts. Local module/declaration numbers are intentionally opaque.
fn state::Plan altered(ref state::Plan p,usize mode){
 var bytes::Buffer names=bytes::empty();assert(bytes::append(ref mut names,state::names(p)).is_ok);var v::Vec<state::Field> fields=v::empty<state::Field>();var usize i=0;
 while(i<state::field_count(p)){var f=state::field_at(p,i);if(i==0){if(mode==3){f.prefix[0]^=1;}if(mode==4){f.name_end=1048577;}if(mode==5){f.value=8;}if(mode==8){f.end=1048577;}if(mode==9){f.key=1;}}assert(v::push<state::Field>(ref mut fields,f).is_ok);i+=1;}
 if(state::count(p)==0){return state::assemble(names,v::empty<state::Layout>(),fields);}
 var row=state::at(p,0);if(mode==1 || mode==7 || mode==13 || mode==14){row.owner=7;row.module_index=99;row.declaration=37;}if(mode==2){row.hash[0]^=1;}if(mode==6){row.uid_end=1048577;}
 var v::Vec<state::Layout> layouts=v::empty<state::Layout>();assert(v::push<state::Layout>(ref mut layouts,row).is_ok);
 if(mode==1 || mode==13 || mode==14){row.owner=9;row.uid_start=1048577;row.uid_end=1048576;row.field_start=1048577;row.field_count=1025;assert(v::push<state::Layout>(ref mut layouts,row).is_ok);}
 if(mode==10){assert(v::push<state::Layout>(ref mut layouts,row).is_ok);}return state::assemble(names,layouts,fields);
}
fn Result<bytes::Buffer,u32> call(ref state::Plan p,usize owner,ref model::Descriptor d,ref generic::Templates t,usize mode,bool graph,ref mut usize work){
 if(mode==0){return if(graph){prefix::prefix_graph(p,owner,work)}else{prefix::prefix(p,owner,work)};}
 if(mode==1){return if(graph){public::encode_graph(p,owner,d,work)}else{public::encode(p,owner,d,work)};}
 if(mode==2){return if(graph){public::hash_graph(p,owner,d,work)}else{public::hash(p,owner,d,work)};}
 if(mode==3){return if(graph){generic::encode_graph(p,owner,d,t,work)}else{generic::encode(p,owner,d,t,work)};}
 return if(graph){generic::hash_graph(p,owner,d,t,work)}else{generic::hash(p,owner,d,t,work)};
}
fn u32 code(ref state::Plan p,usize owner,ref model::Descriptor d,ref generic::Templates t,usize mode,bool graph,ref mut usize work){return switch(call(p,owner,d,t,mode,graph,work)){case .ok(_):{0}case .err(e):{e}};}
fn void bounds(ref state::Plan p,usize owner,ref model::Descriptor d,ref generic::Templates t,usize mode){
 var usize cost=0;let original=switch(call(p,owner,d,t,mode,true,ref mut cost)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);
 var usize work=1048576-cost;let exact=switch(call(p,owner,d,t,mode,true,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref exact)));
 work=1048576-cost+1;assert(code(p,owner,d,t,mode,true,ref mut work)==10 && work<=1048576);
 work=1048577;assert(code(p,owner,d,t,mode,true,ref mut work)==10 && work==1048577);
}
@effects(console)fn void output(ref bytes::Buffer b,bool hex){if(hex){var bytes::Buffer text=bytes::empty();assert(literals::hex(ref mut text,bytes::view(b)));discard io::write_bytes(bytes::view(ref text));}else{discard io::write_bytes(bytes::view(b));}discard io::write("\n");}
@effects(console)fn i32 main(){var bytes::Buffer input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] data=ref chunk;assert(bytes::append(ref mut input,data[..n]).is_ok);}
 let raw=bytes::view(ref input);assert(raw.len>2);let mode=raw[0] as usize;let dynamic=raw[1]!=0;
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parse(raw[2..])).is_ok);let c=switch(catalog::build(ref sources)){case .ok(p):{p}case .err(_):{return 3;}};
 let original=switch(declarations::build_graph(ref sources,ref c)){case .ok(p):{p}case .err(_):{return 4;}};
 let p=altered(ref original,mode);let usize owner=if(mode==1 || mode==7){7}else if(mode==13){9}else if(mode==14){8}else{0};let d=descriptor(mode==11);let t=templates(if(mode==12){1}else{0});
 var usize i=0;while(i<5){var usize work=0;
 if(mode>=2 && mode<=6 || mode>=8 && mode<=10 || mode==13){assert(code(ref p,owner,ref d,ref t,i,true,ref mut work)==9);}
 else if(mode==11 || mode==12){if(i>=if(mode==11){1}else{3}){assert(code(ref p,owner,ref d,ref t,i,true,ref mut work)==9);}}
 else{bounds(ref p,owner,ref d,ref t,i);work=0;let result=switch(call(ref p,owner,ref d,ref t,i,true,ref mut work)){case .ok(b):{b}case .err(_):{return 5;}};
 if(dynamic){work=0;assert(code(ref p,owner,ref d,ref t,i,false,ref mut work)==9);}else{var usize ordinary_work=0;let ordinary=switch(call(ref p,owner,ref d,ref t,i,false,ref mut ordinary_work)){case .ok(b):{b}case .err(_):{return 6;}};assert(ordinary_work==work && bytes::equal(bytes::view(ref result),bytes::view(ref ordinary)));}
 output(ref result,i==0 || i==1 || i==3);
 }i+=1;}return 0;
}`

func TestCandidateStateGraphInterface(t *testing.T) {
	seven := stateGraphSourceText(`storage::write("blob",b"\x00\xff");storage::write("title","한글");storage::write("labels","key",b"value");storage::commit();`)
	legacy := `module ledger;@contract_uid("fixture.legacy")state{cell bool flag;cell u32 count;cell u64 total;cell bytes20 account;cell bytes32 hash;map bytes20 u64 balances;}fn void transaction(){}`
	none := `module ledger;fn void transaction(){}`
	trials := []struct {
		name, text string
		mode       byte
		dynamic    bool
		want       string
	}{
		{"seven-codec", seven, 0, true, stateGraphInterfaceOracle(t, seven, true)},
		{"selected-owner-unrelated-malformed", seven, 1, true, stateGraphInterfaceOracle(t, seven, true)},
		{"selected-owner-renumbered", seven, 7, true, stateGraphInterfaceOracle(t, seven, true)},
		{"five-codec-identical", legacy, 0, false, stateGraphInterfaceOracle(t, legacy, false)},
		{"no-state-identical", none, 0, false, stateGraphInterfaceOracle(t, none, false)},
		{"missing-owner-ignores-malformed", seven, 14, false, stateGraphInterfaceOracle(t, none, false)},
	}
	for _, field := range []string{"cell bytes blob;", "cell text title;", "map bytes text labels;", "map text bytes labels;"} {
		text := `module ledger;@contract_uid("fixture.codec")state{` + field + `}fn void transaction(){storage::commit();}`
		trials = append(trials, struct {
			name, text string
			mode       byte
			dynamic    bool
			want       string
		}{field, text, 0, true, stateGraphInterfaceOracle(t, text, true)})
	}
	for mode, name := range map[byte]string{2: "layout-hash", 3: "field-prefix", 4: "name-range", 5: "unknown-codec", 6: "uid-range", 8: "field-span", 9: "cell-key", 10: "duplicate-selected", 11: "descriptor-range", 12: "zero-closure", 13: "selected-malformed-owner"} {
		trials = append(trials, struct {
			name, text string
			mode       byte
			dynamic    bool
			want       string
		}{name, seven, mode, true, ""})
	}
	candidateAtlasProbeSize(t, "module_generic_interface", stateGraphInterfaceProbe, 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				dynamic := byte(0)
				if trial.dynamic {
					dynamic = 1
				}
				input := append([]byte{trial.mode, dynamic}, []byte(trial.text)...)
				code, out := candidateAtlasProcess(t, image, input)
				if code != 0 || out != trial.want {
					t.Fatalf("exit=%d output=%q want=%q", code, out, trial.want)
				}
			})
		}
	}, "module_state_declarations", "module_catalog", "scanner", "syntax", "literal_types")
}
