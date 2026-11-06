package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const stateGraphCompleteProbe = `module main;
import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_source_identity as graph_identity;import module_state_graph_body_hash as graph_hash;import module_state_graph_source as authored;import module_state_graph_source_artifact as artifact;import module_project_interfaces as records;import module_public_interface as public;import module_public_type_text as coordinates;import project_cache_body_identity as body;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn Result<core::Complete,u32> admit(ref u8[] raw){
 var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let owner=wire::number_from(raw,ref mut c,ref mut setup);let count=wire::number_from(raw,ref mut c,ref mut setup);if(c.code!=0 || count>513){return .err(9001);}var rows=v::empty<coordinates::Coordinate>();var usize i=0;
 while(i<count){let owner=wire::number_from(raw,ref mut c,ref mut setup);let start=wire::number_from(raw,ref mut c,ref mut setup);let end=wire::number_from(raw,ref mut c,ref mut setup);if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start,.end}).is_ok){return .err(9002);}i+=1;}
 let size=wire::number_from(raw,ref mut c,ref mut setup);if(c.code!=0 || size>raw.len-c.offset){return .err(9003);}let names=switch(bytes::copy(raw[c.offset..c.offset+size])){case .ok(p):{p}case .err(_):{return .err(9004);}};c.offset+=size;let table=coordinates::assemble(names,rows);let actual=raw[c.offset..];var usize work=0;
 let current=switch(core::build_complete_state_graph(actual,ref table,owner,ref mut work)){case .ok(p):{p}case .err(p):{if(p.phase!=8 || p.stage!=1){return .err(9005);}return .err(p.code);}};let cost=work;
 if(!core::is_state_graph(ref current) || graph::owner(core::checked_state_graph(ref current))!=owner){return .err(9006);}
 work=16777216-cost;switch(core::build_complete_state_graph(actual,ref table,owner,ref mut work)){case .ok(_):{}case .err(_):{return .err(9007);}};if(work!=16777216){return .err(9008);}
 work=16777217-cost;switch(core::build_complete_state_graph(actual,ref table,owner,ref mut work)){case .ok(_):{return .err(9009);}case .err(p):{if(p.code!=1002 || work>16777216){return .err(9010);}}};
 work=16777217;switch(core::build_complete_state_graph(actual,ref table,owner,ref mut work)){case .ok(_):{return .err(9011);}case .err(p):{if(p.code!=1002 || work!=16777217){return .err(9012);}}};
 let original=graph::source(core::checked_state_graph(ref current));work=0;let saved=switch(artifact::encode(original,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9013);}};
 work=0;let restored=switch(core::restore_complete_state_graph(wire::content(ref saved),actual,ref table,owner,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9014);}};let restore_cost=work;
 work=16777216-restore_cost;switch(core::restore_complete_state_graph(wire::content(ref saved),actual,ref table,owner,ref mut work)){case .ok(_):{}case .err(_):{return .err(9015);}};if(work!=16777216){return .err(9016);}
 work=16777217-restore_cost;switch(core::restore_complete_state_graph(wire::content(ref saved),actual,ref table,owner,ref mut work)){case .ok(_):{return .err(9017);}case .err(p):{if(p.code!=1002 || work>16777216){return .err(9018);}}};
 work=16777217;switch(core::restore_complete_state_graph(wire::content(ref saved),actual,ref table,owner,ref mut work)){case .ok(_):{return .err(9019);}case .err(p):{if(p.code!=1002 || work!=16777217){return .err(9020);}}};
 work=0;let first=switch(body::derive(ref current,owner,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9021);}};let identity_cost=work;work=0;let second=switch(body::derive(ref restored,owner,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9022);}};
 if(!bytes::equal(body::canonical(ref first),body::canonical(ref second)) || !bytes::equal(body::digest(ref first),body::digest(ref second)) || !bytes::equal(body::coordinate(ref first),body::coordinate(ref second))){return .err(9023);}
 work=16777216-identity_cost;switch(body::derive(ref current,owner,ref mut work)){case .ok(_):{}case .err(_):{return .err(9024);}};if(work!=16777216){return .err(9025);}
 work=16777217-identity_cost;switch(body::derive(ref current,owner,ref mut work)){case .ok(_):{return .err(9026);}case .err(code):{if(code!=10 || work>16777216){return .err(9027);}}};
 work=0;switch(body::derive(ref current,if(owner==0){1}else{0},ref mut work)){case .ok(_):{return .err(9028);}case .err(code):{if(code!=9){return .err(9029);}}};
 work=0;let decoded_graph=core::checked_state_graph(ref restored);let decoded=switch(graph_hash::derive(graph::program(decoded_graph),graph_identity::context(graph::body_identity(decoded_graph)),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9037);}};
 if(!bytes::equal(graph_hash::canonical(ref decoded),body::canonical(ref first)) || !bytes::equal(graph_hash::digest(ref decoded),body::digest(ref first))){return .err(9038);}
 work=0;let fresh=switch(graph_hash::derive(graph::program(core::checked_state_graph(ref current)),graph_identity::context(graph::body_identity(core::checked_state_graph(ref current))),ref mut work)){case .ok(p):{p}case .err(_):{return .err(9039);}};
 if(!bytes::equal(graph_hash::canonical(ref fresh),body::canonical(ref first))){return .err(9040);}
 let a=core::public_interfaces(ref current);let b=core::public_interfaces(ref restored);if(records::count(a)!=1 || records::count(b)!=1){return .err(9030);}let ar=records::at(a,0);let br=records::at(b,0);
 if(records::owner(ar)!=owner || records::owner(br)!=owner || !bytes::equal(records::ordinary_hash(ar),records::ordinary_hash(br)) || !bytes::equal(records::source_hash(ar),records::source_hash(br)) || v::len<bytes::Buffer>(records::source_specializations(ar))!=0){return .err(9031);}
 let inline=records::inline_hash(ar);i=0;while(i<8){if(inline[i]!=0){return .err(9032);}i+=1;}
 var changed=switch(bytes::copy(actual)){case .ok(p):{p}case .err(_):{return .err(9033);}};if(!bytes::append_byte(ref mut changed,10).is_ok){return .err(9034);}work=0;
 switch(core::restore_complete_state_graph(wire::content(ref saved),bytes::view(ref changed),ref table,owner,ref mut work)){case .ok(_):{return .err(9035);}case .err(p):{if(p.phase!=8 || p.stage!=2 || p.code!=1001){return .err(9036);}}};
 // Actual source and coordinate table are temporary. The returned Complete owns both.
 return .ok(current);
}
@effects(console)fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let current=switch(admit(bytes::view(ref input))){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 // No source or coordinate buffer remains alive outside the Complete owner.
 {let raw=bytes::view_mut(ref mut input);if(raw.len!=0){raw[0]=99;}}input=bytes::empty();
 let owned=core::checked_state_graph(ref current);let source=graph::source(owned);let owner=graph::owner(owned);var usize work=0;let identity=switch(body::derive(ref current,owner,ref mut work)){case .ok(p):{p}case .err(_):{return 3;}};
 let record=records::at(core::public_interfaces(ref current),0);work=0;let encoded=switch(public::encode_graph(authored::layouts(source),authored::owner(source).package,records::descriptor(record),ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};
 if(!io::write_bytes(body::canonical(ref identity)).is_ok || !io::write("\n").is_ok || !io::write_bytes(body::digest(ref identity)).is_ok || !io::write("\n").is_ok || !io::write_bytes(records::ordinary_hash(record)).is_ok || !io::write("\n").is_ok || !io::write_bytes(records::source_hash(record)).is_ok || !io::write("\n").is_ok || !io::write_bytes(bytes::view(ref encoded)).is_ok){return 5;}return 0;
}`

func TestCandidateStateGraphComplete(t *testing.T) {
	type trial struct {
		name        string
		input, want []byte
	}
	var trials []trial
	for _, authored := range stateGraphAuthoredTrials() {
		fixture, code := stateGraphAuthoredOracle(t, authored)
		owner := uint64(7)
		coordinate := fixture.coordinate
		if code != "" {
			coordinate = packagegraph.CoordinateID{1}
		}
		namespace := coordinate.String()
		want := []byte(code)
		if code == "" {
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{7: coordinate}
			descriptor, err := moduleinterface.Extract(fixture.input, 7, coordinates)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := moduleinterface.Encode(descriptor)
			if err != nil {
				t.Fatal(err)
			}
			ordinary, err := moduleinterface.Compute(descriptor)
			if err != nil {
				t.Fatal(err)
			}
			source, err := moduleinterface.ComputeGeneric(descriptor, nil)
			if err != nil {
				t.Fatal(err)
			}
			want = append(bytes.Clone(fixture.canonical), []byte("\n"+fixture.hash+"\n"+ordinary.String()+"\n"+source.String()+"\n")...)
			want = append(want, encoded...)
		}
		packet := stateSourceIdentityPacket(owner, []stateSourceIdentityRow{{owner, 0, uint64(len(namespace))}}, namespace, authored.text)
		trials = append(trials, trial{authored.name, packet, want})
	}
	// Invalid coordinates never produce even a public-interface prefix.
	base := stateGraphAuthoredSource("", `storage::commit();`, "")
	for _, owner := range []uint64{0, 511, 512} {
		code := "1001"
		namespace := "fypc1-sha256-" + strings.Repeat("0", 64)
		trials = append(trials, trial{fmt.Sprintf("zero-coordinate-owner-%d", owner), stateSourceIdentityPacket(owner, []stateSourceIdentityRow{{owner, 0, uint64(len(namespace))}}, namespace, base), []byte(code)})
	}
	candidateAtlasProbeSize(t, "project_compiler_core", stateGraphCompleteProbe, 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, trial.input)
				if code != 0 || !bytes.Equal([]byte(output), trial.want) {
					t.Fatalf("exit=%d output=%d/%d prefix=%x", code, len(output), len(trial.want), []byte(output[:min(len(output), 80)]))
				}
			})
		}
	}, "project_cache_body_identity", "module_state_graph_source_artifact", "module_public_interface")
}

func TestCandidateStateGraphCompleteSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"complete-constructor", "E0103", `fn core::Complete forge(){return core::Complete{.program=0,.interfaces=0,.state_graphs=0};}`},
		{"graph-constructor", "E0103", `fn graph::Complete forge(){return graph::Complete{.cold=0,.restored=0,.identity=0,.interfaces=0,.owner=0};}`},
		{"immutable-program", "E1008", `fn void forge(ref graph::Complete p){let ref mut program::Verified g=ref mut graph::program(p);}`},
		{"immutable-graph", "E1008", `fn void forge(ref core::Complete p){let ref mut graph::Complete g=ref mut core::checked_state_graph(p);}`},
		{"immutable-source", "E1008", `fn void forge(ref graph::Complete p){let ref mut authored::Produced s=ref mut graph::source(p);}`},
		{"immutable-records", "E1008", `fn void forge(ref core::Complete p){let ref mut records::Records r=ref mut core::public_interfaces(p);}`},
		{"immutable-parsed", "E1008", `fn void forge(ref authored::Produced p){let ref mut syntax::Parsed s=ref mut authored::parsed(p);}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "project_compiler_core")
			writeTestSource(t, root, "src/main.fy", "module main;import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_source_identity as graph_identity;import module_state_graph_body_hash as graph_hash;import module_state_graph_source as authored;import module_project_interfaces as records;import syntax as syntax;import module_state_graph_program as program;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed state Complete", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), "main.fy:") || !strings.Contains(diagnostic.String(), trial.code) {
				t.Fatalf("sealed Complete code=%d diagnostics=%s", code, &diagnostic)
			}
		})
	}
}
