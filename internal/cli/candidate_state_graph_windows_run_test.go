package cli

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func graphWindowsRunProbe() string {
	prefix := strings.Split(graphResponseProbe(), "@effects(console)fn i32 main(){")[0]
	prefix = strings.Replace(prefix, "module main;", `module main;import module_state_graph_windows_build as builder;import module_state_graph_windows_run as runner;import native_msvc_build as msvc;import windows_msvc_toolchain as tools;`, 1)
	return prefix + `
@effects(console)fn void result(ref response::Admitted admitted,ref runner::Work work){
 number(0);number(work.response_structure);number(work.response_bytes);let h=response::header(admitted);number(h.status);number(h.exit);number(h.trap);number(h.cost);number(h.operations);number(h.write_bytes);number(response::version(admitted));blob(response::trap_category(admitted));
 number(response::read_count(admitted));var usize i=0;while(i<response::read_count(admitted)){let row=response::read_at(admitted,i);number(row.operation);number(if(row.found){1}else{0});number(row.codec as usize);blob(response::read_value(admitted,i));i+=1;}
 number(response::write_count(admitted));i=0;while(i<response::write_count(admitted)){let row=response::write_at(admitted,i);var usize meter=0;let key=switch(identities::words(row.key,ref mut meter)){case .ok(p):{p}case .err(_):{return;}};blob(bytes::view(ref key));number(if(row.deleted){1}else{0});blob(response::write_value(admitted,i));i+=1;}
}
fn bool same_work(runner::Work a,runner::Work b){return a.request_structure==b.request_structure && a.request_bytes==b.request_bytes && a.response_structure==b.response_structure && a.response_bytes==b.response_bytes;}
@effects(filesystem,console,environment,clock)fn i32 main(){
 var incoming=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref incoming)>2097152-n || !bytes::append(ref mut incoming,b[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref incoming);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);let release=wire::number_from(raw,ref mut c,ref mut setup)!=0;let coordinate_size=wire::number_from(raw,ref mut c,ref mut setup);let source_size=wire::number_from(raw,ref mut c,ref mut setup);let runtime_size=wire::number_from(raw,ref mut c,ref mut setup);let frame_size=wire::number_from(raw,ref mut c,ref mut setup);
 if(c.code!=0 || coordinate_size>raw.len-c.offset || source_size>raw.len-c.offset-coordinate_size || runtime_size>raw.len-c.offset-coordinate_size-source_size || frame_size!=raw.len-c.offset-coordinate_size-source_size-runtime_size){return 3;}
 let coordinate=raw[c.offset..c.offset+coordinate_size];let actual=raw[c.offset+coordinate_size..c.offset+coordinate_size+source_size];let asset_at=c.offset+coordinate_size+source_size;let runtime_bytes=raw[asset_at..asset_at+runtime_size];let frame_bytes=raw[asset_at+runtime_size..];
 var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=coordinate.len}).is_ok);let table=coordinates::assemble(copy(coordinate),rows);
 var usize asset_work=0;let runtime=switch(runtime::load(copy(runtime_bytes),ref mut asset_work)){case .ok(p):{p}case .err(_):{return 4;}};asset_work=0;let framing=switch(frame::load(copy(frame_bytes),ref mut asset_work)){case .ok(p):{p}case .err(_):{return 5;}};
 let checked=switch(current(actual,ref table,owner,(mode&2)!=0)){case .ok(p):{p}case .err(_):{return 6;}};let hash=layout::at(program::layouts(graph::program(core::checked_state_graph(ref checked))),model::header(program::root(graph::program(core::checked_state_graph(ref checked)))).layout).hash;
 var prepare_work=transport::empty_work();let prepared=switch(transport::prepare(checked,ref runtime,ref framing,b"x86_64-windows-msvc",ref mut prepare_work)){case .ok(p):{p}case .err(_):{return 7;}};
 let tools=switch(tools::resolve()){case .ok(p):{p}case .err(e):{number(800);number(e.stage as usize);number(e.status as usize);number(e.os_code as usize);return 0;}};
 var usize build_work=0;let built=switch(builder::build(ref tools,prepared,ref runtime,msvc::Options{.release,.timeout_ms=60000,.stack_reserve=16777216},ref mut build_work)){case .ok(p):{p}case .err(e):{number(801);number(builder::stage(ref e) as usize);number(builder::status(ref e) as usize);number(builder::os_code(ref e) as usize);blob(builder::diagnostics(ref e));return 0;}};
 assert(builder::object_count(ref built)==2);let snapshot=switch(response_snapshot(builder::prepared(ref built),hash,mode)){case .ok(p):{p}case .err(_):{return 8;}};let limits=request::Limits{.writes=64,.bytes=65536,.value=4096,.key=4096};let options=runner::Options{.timeout_ms=60000};var work=runner::empty_work();
 let admitted=switch(runner::run(ref built,ref snapshot,limits,options,ref mut work)){case .ok(p):{p}case .err(e):{number(802);number(runner::stage(ref e) as usize);number(runner::status(ref e) as usize);number(runner::os_code(ref e) as usize);blob(runner::diagnostics(ref e));return 0;}};
 if((mode&4)!=0){
  let cost=work;var exact=runner::Work{.request_structure=16777216-cost.request_structure,.request_bytes=1073741824-cost.request_bytes,.response_structure=16777216-cost.response_structure,.response_bytes=1073741824-cost.response_bytes};
  switch(runner::run(ref built,ref snapshot,limits,options,ref mut exact)){case .ok(p):{let a=response::header(ref p);let b=response::header(ref admitted);assert(a.status==b.status && a.exit==b.exit && a.cost==b.cost && a.operations==b.operations && response::write_count(ref p)==response::write_count(ref admitted));assert(exact.request_structure==16777216 && exact.request_bytes==1073741824 && exact.response_structure==16777216 && exact.response_bytes==1073741824);}case .err(_):{return 9;}};
  var usize i=0;while(i<4){var short=runner::empty_work();if(i==0){short.request_structure=16777217-cost.request_structure;}if(i==1){short.request_bytes=1073741825-cost.request_bytes;}if(i==2){short.response_structure=16777217-cost.response_structure;}if(i==3){short.response_bytes=1073741825-cost.response_bytes;}
   switch(runner::run(ref built,ref snapshot,limits,options,ref mut short)){case .ok(_):{return 10;}case .err(e):{assert(runner::stage(ref e)==if(i<2){2}else{7} && runner::status(ref e)==1002);}};i+=1;}
 }
 if((mode&8)!=0){
  var usize i=0;while(i<6){var invalid=runner::empty_work();if(i==2){invalid.request_structure=16777217;}if(i==3){invalid.request_bytes=1073741825;}if(i==4){invalid.response_structure=16777217;}if(i==5){invalid.response_bytes=1073741825;}let before=invalid;let u32 timeout=if(i==0){0}else if(i==1){600001}else{60000};
   switch(runner::run(ref built,ref snapshot,limits,runner::Options{.timeout_ms=timeout},ref mut invalid)){case .ok(_):{return 11;}case .err(e):{assert(runner::stage(ref e)==1 && runner::status(ref e)==if(i<2){1001}else{1002} && same_work(invalid,before));}};i+=1;}
 }
 // The first admitted result remains owned across further native executions.
 {let changed=bytes::view_mut(ref mut incoming);var usize i=0;while(i<changed.len){changed[i]=255;i+=1;}}incoming=bytes::empty();result(ref admitted,ref work);return 0;
}`
}

func graphWindowsRunPacket(f stateBodyFixture, actual string, runtimePack, framePack []byte, mode, owner uint64, release bool) []byte {
	coordinate := "fypc1-sha256-" + hex.EncodeToString(f.coordinate[:])
	var packet []byte
	profile := uint64(0)
	if release {
		profile = 1
	}
	for _, n := range []uint64{mode, owner, profile, uint64(len(coordinate)), uint64(len(actual)), uint64(len(runtimePack)), uint64(len(framePack))} {
		packet = binary.BigEndian.AppendUint64(packet, n)
	}
	packet = append(packet, coordinate...)
	packet = append(packet, actual...)
	packet = append(packet, runtimePack...)
	return append(packet, framePack...)
}

func TestCandidateStateGraphWindowsRun(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows graph execution; Linux execution is batched")
	}
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	var trials []stateGraphAuthoredTrial
	for _, trial := range graphTransportTrials() {
		switch trial.name {
		case "empty-commit", "empty-reject", "trap", "six-storage-actions", "helper-transitive", "helper-branch", "map-local-key-value", "borrow-alias-view-copy", "present-borrow-copy", "branch-present", "seeded-branch", "text-unicode", "u64-max", "bytes20", "bytes32":
			trials = append(trials, trial)
		}
	}
	trials = append(trials, stateGraphAuthoredTrial{"snapshot-above-literal-cap", stateGraphAuthoredSource("storage_read", `storage::read("blob");storage::commit();`, ""), ""})
	candidateAtlasProbeSize(t, "module_state_graph_windows_run", graphWindowsRunProbe(), 512<<10, func(t *testing.T, image string) {
		var processes int
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				f, code := stateGraphAuthoredOracle(t, trial)
				if code != "" {
					t.Fatal(code)
				}
				mode := uint64(0)
				owner := uint64(0)
				if trial.name == "helper-transitive" {
					mode = 2 | 4 | 8
					owner = 511
				}
				if trial.name == "present-borrow-copy" {
					mode = 4 | 8
				}
				if trial.name == "seeded-branch" {
					mode = 16
				}
				if trial.name == "snapshot-above-literal-cap" {
					mode = 65536
				}
				temporary := t.TempDir()
				t.Setenv("TEMP", temporary)
				t.Setenv("TMP", temporary)
				processes++
				exit, out := candidateAtlasProcess(t, image, graphWindowsRunPacket(f, trial.text, runtimePack, framePack, mode, owner, strings.Contains(t.Name(), "/optimized/")))
				if exit != 0 {
					t.Fatalf("Fairy native orchestration exit=%d output=%q", exit, out)
				}
				if strings.HasPrefix(out, "80") {
					t.Fatalf("Fairy native orchestration stage: %s", out)
				}
				admit, _, got := graphResponseRead(t, out)
				if admit != 0 {
					t.Fatal(admit)
				}
				if entries, err := os.ReadDir(temporary); err != nil || len(entries) != 0 {
					t.Fatalf("native build/run scratch leaked: %v %v", entries, err)
				}
				if mode&16 == 0 && mode&65536 == 0 {
					graphNativeSemanticExpect(t, f.input, got)
				} else {
					_, _, _, handoff := graphNativeOracle(t, f, true)
					layout := graphNativeLayout(t, handoff)
					lowered, err := contractstatemir.LowerHIRTransactions(f.input)
					if err != nil {
						t.Fatal(err)
					}
					var key contractstate.Key
					fieldName := "flag"
					value := []byte{1}
					version := uint64(1)
					if mode&65536 != 0 {
						fieldName = "blob"
						value = make([]byte, 2<<20)
						version = 73
					}
					for _, field := range lowered[0].Program.Layout.Fields {
						if field.Name == fieldName {
							key = field.Prefix
						}
					}
					if key == (contractstate.Key{}) {
						t.Fatal("missing snapshot field")
					}
					want := contractstatenative.GraphResult{Status: 0, Exit: contractstatemir.ExitCommit, Version: version, Cost: 3, Operations: 1, Reads: []contractstatemir.Read{{Operation: 0, Found: true, Value: contractstatemir.Value{TypeID: "bool/v1", Encoded: value}}}}
					if fieldName == "blob" {
						want.Cost = 2
						want.Reads[0].Value.TypeID = "bytes/v1"
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("snapshot native result layout=%x got=%+v", layout, got)
					}
				}
			})
		}
		t.Logf("Fairy-owned native preparation/build/run/decode processes=%d", processes)
	})
}

func TestCandidateStateGraphWindowsRunSealedOwner(t *testing.T) {
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_state_graph_windows_run")
	writeTestSource(t, root, "src/main.fy", `module main;import module_state_graph_windows_build as build;import std::bytes as bytes;fn build::Built forge(){return build::Built{.prepared=0,.image=bytes::empty(),.objects=2};}fn i32 main(){return 0;}`)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostic bytes.Buffer
	_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed graph executable", &diagnostic)
	if code == ExitSuccess || !strings.Contains(diagnostic.String(), "E0103") {
		t.Fatalf("sealed built code=%d: %s", code, &diagnostic)
	}
}
