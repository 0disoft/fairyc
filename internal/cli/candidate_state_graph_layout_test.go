package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const stateGraphLayoutProbe = `module main;
import module_state_declarations as declarations;
import module_state_layout as state;
import module_state_artifact as codec;
import module_project_artifact_state as current;
import candidate_artifact_wire as wire;
import module_program_artifact_wire as joined;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn Result<syntax::Parsed,u32> parsed(ref u8[] raw){
 let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(11);}};
 let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(9);}};
 let result=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{return .err(9);}};
 if(syntax::problem_count(ref result)!=0){return .err(9);}return .ok(result);
}
@effects(console) fn i32 failed(u32 code){discard io::write("error:");discard io::write_u32(code);discard io::write("\n");return 3;}
fn u32 budgets(ref state::Plan plan,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog c){
 var usize cost=0;discard switch(declarations::build_graph_metered(sources,c,ref mut cost)){case .ok(p):{p}case .err(_):{return 71;}};
 var usize work=1048576-cost;discard switch(declarations::build_graph_metered(sources,c,ref mut work)){case .ok(p):{p}case .err(_):{return 72;}};if(work!=1048576){return 73;}
 work=1048576-cost+1;if(!switch(declarations::build_graph_metered(sources,c,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10}}){return 74;}
 cost=0;let payload=switch(codec::encode_graph(plan,ref mut cost)){case .ok(p):{p}case .err(_):{return 75;}};
 work=16777216-cost;discard switch(codec::encode_graph(plan,ref mut work)){case .ok(p):{p}case .err(_):{return 76;}};if(work!=16777216){return 77;}
 work=16777216-cost+1;if(!switch(codec::encode_graph(plan,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}}){return 78;}
 cost=0;discard switch(codec::decode_graph(wire::content(ref payload),ref mut cost)){case .ok(p):{p}case .err(_):{return 79;}};
 work=16777216-cost;discard switch(codec::decode_graph(wire::content(ref payload),ref mut work)){case .ok(p):{p}case .err(_):{return 80;}};if(work!=16777216){return 81;}
 work=16777216-cost+1;if(!switch(codec::decode_graph(wire::content(ref payload),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}}){return 82;}
 cost=0;discard switch(current::build_graph(plan,sources,c,ref mut cost)){case .ok(p):{p}case .err(_):{return 83;}};
 work=joined::work_limit()-cost;discard switch(current::build_graph(plan,sources,c,ref mut work)){case .ok(p):{p}case .err(_):{return 84;}};if(work!=joined::work_limit()){return 85;}
 work=joined::work_limit()-cost+1;if(!switch(current::build_graph(plan,sources,c,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}}){return 86;}return 0;
}
fn u32 primitive(ref state::Plan plan,usize mode,ref mut usize work){
 if(mode==0){return switch(state::prefix(plan,0,0,work)){case .ok(_):{0}case .err(code):{code}};}
 if(mode==1){return switch(state::digest(plan,0,work)){case .ok(_):{0}case .err(code):{code}};}
 if(mode==2){return switch(state::copy(plan,work)){case .ok(_):{0}case .err(code):{code}};}
 return state::validate_graph(plan,work);
}
fn u32 primitive_bounds(ref state::Plan plan){
 var bytes::Buffer out=bytes::empty();var usize work=1048573;
 if(state::append_checked(ref mut out,b"abc",ref mut work)!=10 || work!=1048573 || bytes::len(ref out)!=0){return 91;}
 work=1048572;if(state::append_checked(ref mut out,b"abc",ref mut work)!=0 || work!=1048576 || !bytes::equal(bytes::view(ref out),b"abc")){return 92;}
 if(state::append_checked(ref mut out,b"",ref mut work)!=10 || work!=1048576 || !bytes::equal(bytes::view(ref out),b"abc")){return 93;}
 var usize mode=0;while(mode<4){var usize cost=0;if(primitive(plan,mode,ref mut cost)!=0 || cost==0 || cost>1048576){return 94;}
 work=1048576-cost;if(primitive(plan,mode,ref mut work)!=0 || work!=1048576){return 95;}
 work=1048576-cost+1;if(primitive(plan,mode,ref mut work)!=10 || work>1048576){return 96;}
 work=1048577;if(primitive(plan,mode,ref mut work)!=10 || work!=1048577){return 97;}mode+=1;}return 0;
}
@effects(console) fn i32 main(){
 var bytes::Buffer input=bytes::empty();var u8[4096] block=[0;4096];
 while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] view=ref block;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,view[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);if(raw.len<2){return 2;}let mode=raw[0];var usize split=1;while(split<raw.len && raw[split]!=0){split+=1;}if(split==raw.len){return 2;}
 let p=switch(parsed(raw[1..split])){case .ok(p):{p}case .err(code):{return failed(code);}};
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);
 let c=switch(catalog::build(ref sources)){case .ok(p):{p}case .err(e):{return failed(e.code);}};var usize work=0;
 let plan=if(mode==0 || mode==1 || mode==6){switch(if(mode==1){declarations::build(ref sources,ref c)}else{declarations::build_graph(ref sources,ref c)}){case .ok(p):{p}case .err(e):{return failed(e.code);}}}else{switch(if(mode==3 || mode==5){codec::decode(raw[split+1..],ref mut work)}else{codec::decode_graph(raw[split+1..],ref mut work)}){case .ok(p):{p}case .err(code):{return failed(code);}}};
 if(mode==2 || mode==3){work=0;discard switch(if(mode==3){current::build(ref plan,ref sources,ref c,ref mut work)}else{current::build_graph(ref plan,ref sources,ref c,ref mut work)}){case .ok(p):{p}case .err(code):{return failed(code);}};}
 if(mode==6){let code=budgets(ref plan,ref sources,ref c);if(code!=0){return failed(code);}}
 if(mode==7){let code=primitive_bounds(ref plan);if(code!=0){return failed(code);}}
 work=0;let output=switch(if(mode==1 || mode==3 || mode==5){codec::encode(ref plan,ref mut work)}else{codec::encode_graph(ref plan,ref mut work)}){case .ok(p):{p}case .err(code):{return failed(code);}};
 assert(io::write_bytes(wire::content(ref output)).is_ok);return 0;
}
`

func stateGraphLayoutOracle(t *testing.T, text string, dynamic bool) ([]byte, string) {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatal("state layout syntax oracle", err, parsed.Tree.Diagnostics)
	}
	input := semantic.PackageInput{Files: []semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}}}
	analyze := semantic.AnalyzePackages
	if dynamic {
		analyze = semantic.AnalyzeExperimentalStatePackages
	}
	checked, err := analyze(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(checked.Diagnostics) != 0 {
		return nil, strings.TrimPrefix(checked.Diagnostics[0].Code, "E")
	}
	if checked.CompletedThrough != semantic.PhaseFlow || len(checked.HIR.StateLayouts) != 1 {
		t.Fatal("state layout oracle was not fully checked")
	}
	layout := checked.HIR.StateLayouts[0]
	builder, err := contractstate.NewLayoutBuilder(layout.ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range layout.Fields {
		if field.Kind == hir.StateFieldCell {
			_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
		} else {
			_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	canonical, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	marker := "FYSTATE0"
	if dynamic {
		marker = "FYSTATE1"
	}
	out := []byte(marker)
	number := func(n uint64) { out = binary.BigEndian.AppendUint64(out, n) }
	words := func(raw []byte) {
		for i := 0; i < len(raw); i += 4 {
			number(uint64(binary.BigEndian.Uint32(raw[i:])))
		}
	}
	names := []byte(layout.ContractUID)
	number(1)
	for _, n := range []uint64{0, 0, 2, uint64(layout.Span.StartByte), uint64(layout.Span.EndByte), 0, uint64(len(names)), 0, uint64(len(layout.Fields))} {
		number(n)
	}
	hash := canonical.Hash()
	words(hash[:])
	number(uint64(len(layout.Fields)))
	codecs := map[string]uint64{"": 0, "bool/v1": 1, "u32/be/v1": 2, "u64/be/v1": 3, "bytes20/v1": 4, "bytes32/v1": 5, "bytes/v1": 6, "utf8/v1": 7}
	for i, field := range canonical.Fields() {
		start := len(names)
		names = append(names, field.Name...)
		span := layout.Fields[i].Span
		for _, n := range []uint64{uint64(field.Kind), codecs[field.KeyTypeID], codecs[field.ValueTypeID], uint64(start), uint64(len(names)), uint64(span.StartByte), uint64(span.EndByte)} {
			number(n)
		}
		words(field.Prefix[:])
	}
	number(uint64(len(names)))
	return append(out, names...), ""
}

func stateGraphLayoutPacket(mode byte, text string, artifact []byte) []byte {
	packet := append([]byte{mode}, text...)
	packet = append(packet, 0)
	return append(packet, artifact...)
}

func TestCandidateStateGraphLayouts(t *testing.T) {
	const seven = `module main;@contract_uid("test.state-graph-layout")state{cell bool flag;cell u32 count;cell u64 total;cell bytes20 owner;cell bytes32 root;cell bytes blob;cell text label;map bytes text labels;map text bytes blobs;}`
	const five = `module main;@contract_uid("test.state-graph-layout")state{cell bool flag;cell u32 count;cell u64 total;map bytes20 u64 balances;cell bytes32 root;}`
	sources := []string{seven, five, `module main;@contract_uid("empty")state{}`}
	for _, n := range []int{254, 255, 256} {
		sources = append(sources, `module main;@contract_uid("`+strings.Repeat("x", n)+`")state{cell text value;}`)
	}
	for _, n := range []int{126, 127, 128} {
		sources = append(sources, `module main;@contract_uid("x")state{map text bytes `+strings.Repeat("x", n)+`;}`)
	}
	for _, field := range []string{"cell bytes<u64> value;", "cell custom::text value;", "cell text[] value;", "map text String values;", "cell bytes value;cell text value;"} {
		sources = append(sources, `module main;@contract_uid("x")state{`+field+`}`)
	}
	var cases []struct {
		text, problem string
		wire          []byte
	}
	for _, text := range sources {
		wire, problem := stateGraphLayoutOracle(t, text, true)
		cases = append(cases, struct {
			text, problem string
			wire          []byte
		}{text, problem, wire})
	}
	candidateAtlasProbeSize(t, "module_project_artifact_state", stateGraphLayoutProbe, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for i, trial := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, stateGraphLayoutPacket(0, trial.text, nil))
				if trial.problem != "" {
					if code != 3 || output != "error:"+trial.problem+"\n" {
						t.Fatalf("source refusal %d/%q, want %s", code, output, trial.problem)
					}
					return
				}
				if code != 0 || !bytes.Equal([]byte(output), trial.wire) {
					t.Fatalf("state source artifact differs from Go: code %d, lengths %d/%d; source %s", code, len(output), len(trial.wire), trial.text)
				}
				for _, mode := range []byte{2, 4, 6} {
					code, output := candidateAtlasProcess(t, image, stateGraphLayoutPacket(mode, trial.text, trial.wire))
					if code != 0 || !bytes.Equal([]byte(output), trial.wire) {
						t.Fatalf("current/roundtrip/budget mode %d: %d %q", mode, code, output)
					}
				}
			})
		}
		for _, text := range []string{five, seven} {
			want, problem := stateGraphLayoutOracle(t, text, false)
			code, output := candidateAtlasProcess(t, image, stateGraphLayoutPacket(1, text, nil))
			if problem != "" {
				if code != 3 || output != "error:"+problem+"\n" {
					t.Fatal("ordinary source admission widened", code, output)
				}
			} else if code != 0 || !bytes.Equal([]byte(output), want) {
				t.Fatal("ordinary artifact identity changed", code, len(output), len(want))
			}
		}
		graph, _ := stateGraphLayoutOracle(t, seven, true)
		legacy, problem := stateGraphLayoutOracle(t, five, false)
		if problem != "" {
			t.Fatal("legacy fixture rejected", problem)
		}
		assertPacket := func(name string, mode byte, text string, artifact, want []byte, problem string) {
			t.Helper()
			t.Run(name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, stateGraphLayoutPacket(mode, text, artifact))
				if problem != "" {
					if code != 3 || out != "error:"+problem+"\n" {
						t.Fatalf("got %d/%q, want error:%s", code, out, problem)
					}
				} else if code != 0 || !bytes.Equal([]byte(out), want) {
					t.Fatalf("got code %d, bytes %d, want %d", code, len(out), len(want))
				}
			})
		}
		assertPacket("primitive-budgets", 7, seven, graph, graph, "")
		assertPacket("current-trailing-comment", 2, seven+"\n// layout unchanged\n", graph, graph, "")
		assertPacket("ordinary-current-roundtrip", 3, five, legacy, legacy, "")
		assertPacket("graph-rejects-legacy-marker", 4, five, legacy, nil, "1001")
		assertPacket("ordinary-rejects-graph-marker", 5, seven, graph, nil, "1001")
		for name, changed := range map[string]string{
			"uid":          strings.Replace(seven, "test.state", "best.state", 1),
			"name":         strings.Replace(seven, "bool flag", "bool next", 1),
			"codec":        strings.Replace(seven, "u32 count", "u64 count", 1),
			"map-key":      strings.Replace(seven, "map bytes text", "map text  text", 1),
			"kind":         strings.Replace(seven, "map bytes text labels", "cell bytes     labels", 1),
			"shifted-span": "\n" + seven,
		} {
			assertPacket("current-rejects-"+name, 2, changed, graph, nil, "1001")
			other, problem := stateGraphLayoutOracle(t, changed, true)
			if problem != "" {
				t.Fatal("changed source must be valid", name, problem)
			}
			assertPacket("current-rejects-rehashed-"+name, 2, seven, other, nil, "1001")
		}
		for _, n := range []int{0, 7, 8, 15, 16, 151, 152, 159, 160, len(graph) - 1} {
			assertPacket(fmt.Sprintf("truncated-%d", n), 4, seven, graph[:n], nil, "1001")
		}
		for name, corrupt := range map[string][]byte{
			"trailing":         append(bytes.Clone(graph), 0),
			"marker":           bytes.Clone(graph),
			"layout-hash":      bytes.Clone(graph),
			"field-prefix":     bytes.Clone(graph),
			"field-word-width": bytes.Clone(graph),
			"codec":            bytes.Clone(graph),
			"name":             bytes.Clone(graph),
			"ordinary-seven":   bytes.Clone(graph),
		} {
			mode := byte(4)
			switch name {
			case "marker":
				corrupt[0] ^= 1
			case "layout-hash":
				corrupt[16+9*8+7] ^= 1
			case "field-prefix":
				corrupt[160+7*8+7] ^= 1
			case "field-word-width":
				corrupt[160] = 1
			case "codec":
				binary.BigEndian.PutUint64(corrupt[160+2*8:], 8)
			case "name":
				corrupt[len(corrupt)-1] ^= 1
			case "ordinary-seven":
				corrupt[7] = '0'
				mode = 5
			}
			assertPacket("corrupt-"+name, mode, seven, corrupt, nil, "1001")
		}
	}, "module_state_artifact")
}

// Replay the original authored layout assertions in their dependency closure.
// Whole-example test admission also loads unrelated generic test declarations.
func TestCandidateOriginalStateLayouts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "module_state_layout.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::module_state_layout;", "module main;", 1)
	main = strings.ReplaceAll(main, "@test", "")
	main += `
fn i32 main(){
 state_layout_owns_order_codecs_and_reference_canonical_hashes();
 state_declarations_reject_bad_authority_duplicates_and_nonstorage_codecs();
 state_uniqueness_uses_package_owners_and_precedes_codec_validation();
 linked_ir_retains_state_and_rejects_changed_layout_identity();
 return 0;
}`
	candidateAtlasProbeSize(t, "module_project_link", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, output := candidateAtlasProcess(t, image, nil); code != 0 || output != "" {
			t.Fatalf("original authored layouts: %d %q", code, output)
		}
	}, "module_state_layout", "module_signatures", "module_catalog", "module_project_compile", "module_unit_body_ir_model", "module_unit_body_ir_verify", "project_diagnostic_codes", "syntax", "scanner")
}
