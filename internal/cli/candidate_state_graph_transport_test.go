package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const graphTransportProbe = `module main;
import module_state_graph_transport as transport;import module_state_graph_transport_request as input;import module_state_graph_request as request;import module_state_graph_native_assets as runtime;import module_state_graph_frame_assets as frame;import module_state_graph_native as native;import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_source as source;import module_state_graph_source_artifact as saved;import module_state_graph_program as program;import module_state_graph_model as model;import module_state_layout as layout;import module_public_type_text as coordinates;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{bytes::empty()}};}
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok);assert(io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok);assert(io::write("\n").is_ok);}
fn Result<core::Complete,u32> current(ref u8[] actual,ref coordinates::Coordinates table,usize owner,bool restored){var usize work=0;
 if(restored){let produced=switch(source::produce(actual,ref mut work)){case .ok(p):{p}case .err(code):{return .err(code);}};work=0;let packet=switch(saved::encode(ref produced,ref mut work)){case .ok(p):{p}case .err(code):{return .err(code);}};work=0;return switch(core::restore_complete_state_graph(wire::content(ref packet),actual,table,owner,ref mut work)){case .ok(p):{.ok(p)}case .err(p):{.err(p.code)}};}
 return switch(core::build_complete_state_graph(actual,table,owner,ref mut work)){case .ok(p):{.ok(p)}case .err(p):{.err(p.code)}};
}
fn Result<request::Snapshot,u32> seed_snapshot(ref transport::Prepared prepared,u32[8] hash,bool seeded){
 if(!seeded){return .ok(request::snapshot_of(hash,0,bytes::empty(),v::empty<request::Entry>()));}
 let proof=graph::program(core::checked_state_graph(transport::checked(prepared)));let layouts=program::layouts(proof);let schema=layout::at(layouts,model::header(program::root(proof)).layout);var usize i=0;
 while(i<schema.field_count){let field=layout::field_at(layouts,schema.field_start+i);if(field.kind==1 && bytes::equal(layout::name(layouts,schema.field_start+i),b"flag")){var entries=v::empty<request::Entry>();if(!v::push<request::Entry>(ref mut entries,request::Entry{.key=field.prefix,.start=0,.end=1}).is_ok){return .err(1003);}return .ok(request::snapshot_of(hash,1,copy(b"\x01"),entries));}i+=1;}return .err(1001);
}
@effects(console)fn i32 main(){
 var incoming=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref incoming)>2097152-n || !bytes::append(ref mut incoming,b[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref incoming);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);var work=transport::empty_work();work.entry=wire::number_from(raw,ref mut c,ref mut setup);work.framing=wire::number_from(raw,ref mut c,ref mut setup);let coordinate_size=wire::number_from(raw,ref mut c,ref mut setup);let source_size=wire::number_from(raw,ref mut c,ref mut setup);let runtime_size=wire::number_from(raw,ref mut c,ref mut setup);let frame_size=wire::number_from(raw,ref mut c,ref mut setup);
 if(c.code!=0 || coordinate_size>raw.len-c.offset || source_size>raw.len-c.offset-coordinate_size || runtime_size>raw.len-c.offset-coordinate_size-source_size || frame_size!=raw.len-c.offset-coordinate_size-source_size-runtime_size){return 3;}
 let coordinate=raw[c.offset..c.offset+coordinate_size];let actual=raw[c.offset+coordinate_size..c.offset+coordinate_size+source_size];let runtime_bytes=raw[c.offset+coordinate_size+source_size..c.offset+coordinate_size+source_size+runtime_size];let frame_bytes=raw[c.offset+coordinate_size+source_size+runtime_size..];
 var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=coordinate.len}).is_ok);let table=coordinates::assemble(copy(coordinate),rows);
 var usize asset_work=0;let runtime=switch(runtime::load(copy(runtime_bytes),ref mut asset_work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work.entry);number(work.framing);return 0;}};asset_work=0;let framing=switch(frame::load(copy(frame_bytes),ref mut asset_work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work.entry);number(work.framing);return 0;}};
 let checked=switch(current(actual,ref table,owner,(mode&2)!=0)){case .ok(p):{p}case .err(code):{number(code as usize);number(work.entry);number(work.framing);return 0;}};
 let hash=layout::at(program::layouts(graph::program(core::checked_state_graph(ref checked))),model::header(program::root(graph::program(core::checked_state_graph(ref checked)))).layout).hash;
 let target=if((mode&8)!=0){b"x86_64-linux-gnu"}else{b"x86_64-windows-msvc"};let prepared=switch(transport::prepare(checked,ref runtime,ref framing,target,ref mut work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work.entry);number(work.framing);return 0;}};
 if((mode&4)!=0){let cost=work;
  var exact=transport::Work{.entry=16777216-cost.entry,.framing=16777216-cost.framing};let fresh=switch(current(actual,ref table,owner,false)){case .ok(p):{p}case .err(_):{return 4;}};let replay=switch(transport::prepare(fresh,ref runtime,ref framing,target,ref mut exact)){case .ok(p):{p}case .err(_):{return 5;}};if(exact.entry!=16777216 || exact.framing!=16777216 || !bytes::equal(transport::source(ref replay),transport::source(ref prepared)) || !bytes::equal(transport::program_hash(ref replay),transport::program_hash(ref prepared))){return 6;}
  var usize meter=0;while(meter<2){var short=transport::Work{.entry=if(meter==0){16777217-cost.entry}else{0},.framing=if(meter==1){16777217-cost.framing}else{0}};let fresh=switch(current(actual,ref table,owner,false)){case .ok(p):{p}case .err(_):{return 7;}};switch(transport::prepare(fresh,ref runtime,ref framing,target,ref mut short)){case .ok(_):{return 8;}case .err(code):{if(code!=1002 || short.entry>16777216 || short.framing>16777216){return 9;}}};meter+=1;}
 }
 let snapshot=switch(seed_snapshot(ref prepared,hash,(mode&16)!=0)){case .ok(p):{p}case .err(_):{return 11;}};let limits=request::Limits{.writes=64,.bytes=65536,.value=4096,.key=4096};var request_work=request::empty_work();let packet=switch(input::encode(ref prepared,ref snapshot,limits,ref mut request_work)){case .ok(p):{p}case .err(_):{return 10;}};
 // Prepared owns the consumed Complete and copied identities beyond caller data.
 {let changed=bytes::view_mut(ref mut incoming);changed[0]=255;}incoming=bytes::empty();
 number(0);number(work.entry);number(work.framing);blob(transport::source(ref prepared));blob(transport::program_hash(ref prepared));blob(input::content(ref packet));blob(input::digest(ref packet));let entry=transport::entry(ref prepared);blob(native::program_hash(entry));return 0;
}`

type graphTransportResult struct {
	code  uint64
	work  [2]uint64
	blobs [5][]byte
}

func graphTransportRead(t *testing.T, raw string) graphTransportResult {
	t.Helper()
	r := bufio.NewReader(strings.NewReader(raw))
	n := func() uint64 {
		t.Helper()
		line, e := r.ReadString('\n')
		if e != nil {
			t.Fatalf("transport number: %v %q", e, raw)
		}
		value, e := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
		if e != nil {
			t.Fatal(e)
		}
		return value
	}
	out := graphTransportResult{code: n(), work: [2]uint64{n(), n()}}
	if out.code == 0 {
		for i := range out.blobs {
			size := n()
			if size > 8<<20 {
				t.Fatal("unbounded frame result", size)
			}
			out.blobs[i] = make([]byte, size)
			if _, e := io.ReadFull(r, out.blobs[i]); e != nil {
				t.Fatal(e)
			}
			if b, e := r.ReadByte(); e != nil || b != '\n' {
				t.Fatal("frame delimiter", b, e)
			}
		}
	}
	if extra, _ := io.ReadAll(r); len(extra) != 0 {
		t.Fatalf("trailing result bytes %q", extra)
	}
	return out
}

func graphTransportPacket(f stateBodyFixture, actual string, runtime, frame []byte, mode, owner uint64, work [2]uint64) []byte {
	coordinate := "fypc1-sha256-" + hex.EncodeToString(f.coordinate[:])
	var out []byte
	for _, n := range []uint64{mode, owner, work[0], work[1], uint64(len(coordinate)), uint64(len(actual)), uint64(len(runtime)), uint64(len(frame))} {
		out = binary.BigEndian.AppendUint64(out, n)
	}
	out = append(out, coordinate...)
	out = append(out, actual...)
	out = append(out, runtime...)
	return append(out, frame...)
}

func graphTransportTrials() []stateGraphAuthoredTrial {
	var out []stateGraphAuthoredTrial
	for _, trial := range stateGraphSourceTrials() {
		switch trial.name {
		case "operations-1024", "reads-1024", "source-65536":
			continue
		}
		if trial.code == "" {
			out = append(out, stateGraphAuthoredTrial{trial.name, trial.text, ""})
		}
	}
	for _, trial := range stateGraphAuthoredTrials() {
		if trial.code == "" {
			out = append(out, trial)
		}
	}
	out = append(out, stateGraphAuthoredTrial{"map-only", `module ledger;@contract_uid("fixture.maps")state{map u32 u32 values;}@effects(storage_read,storage_write)fn void transaction(){storage::write("values",1,7);storage::read("values",1);storage::commit();}`, ""},
		stateGraphAuthoredTrial{"present-borrow-copy", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("blob",b"\x00\xffx");let raw=storage::borrow("blob");let part=storage::view(raw,1,2);let value=storage::copy(part);storage::write("blob",value);storage::commit();`, ""), ""},
		stateGraphAuthoredTrial{"branch-present", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("flag",true);let flag=storage::read("flag");if(flag){storage::write("count",42);}else{storage::reject();}storage::commit();`, ""), ""})
	out = append(out, stateGraphAuthoredTrial{"seeded-branch", stateGraphAuthoredSource("storage_read", `let flag=storage::read("flag");if(flag){storage::commit();}else{storage::reject();}`, ""), ""})
	return out
}

func TestCandidateStateGraphTransport(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native graph transport; Linux execution is batched")
	}
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	candidateAtlasProbeSize(t, "module_state_graph_transport_request", graphTransportProbe, 512<<10, func(t *testing.T, image string) {
		var processes, native int
		run := func(f stateBodyFixture, actual string, runtime, frame []byte, mode, owner uint64, work [2]uint64) graphTransportResult {
			t.Helper()
			processes++
			code, output := candidateAtlasProcess(t, image, graphTransportPacket(f, actual, runtime, frame, mode, owner, work))
			if code != 0 {
				t.Fatalf("transport probe exit=%d output=%q", code, output)
			}
			return graphTransportRead(t, output)
		}
		for _, trial := range graphTransportTrials() {
			t.Run(trial.name, func(t *testing.T) {
				f, code := stateGraphAuthoredOracle(t, trial)
				if code != "" {
					t.Fatal("invalid transport fixture", code)
				}
				_, _, _, handoff := graphNativeOracle(t, f, true)
				prepared, e := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
				if e != nil {
					t.Fatal(e)
				}
				want, e := prepared.Source()
				if e != nil {
					t.Fatal(e)
				}
				hash := prepared.ProgramHash()
				mode := uint64(0)
				if trial.name == "seeded-branch" {
					mode = 16
				}
				if trial.name == "helper-transitive" || trial.name == "trap" {
					mode = 4
				}
				got := run(f, trial.text, runtimePack, framePack, mode, 0, [2]uint64{})
				if got.code != 0 {
					t.Fatalf("transport preparation code=%d work=%v", got.code, got.work)
				}
				if string(got.blobs[0]) != want {
					at := 0
					for at < len(got.blobs[0]) && at < len(want) && got.blobs[0][at] == want[at] {
						at++
					}
					t.Fatalf("transport C differs at%d got%q want%q", at, got.blobs[0][max(0, at-50):min(len(got.blobs[0]), at+100)], want[max(0, at-50):min(len(want), at+100)])
				}
				if !bytes.Equal(got.blobs[1], hash[:]) {
					t.Fatal("transport identity differs")
				}
				snapshot := contractstate.PersistentSnapshot{LayoutHash: graphNativeLayout(t, handoff)}
				if mode == 16 {
					lowered, e := contractstatemir.LowerHIRTransactions(f.input)
					if e != nil {
						t.Fatal(e)
					}
					for _, field := range lowered[0].Program.Layout.Fields {
						if field.Kind == contractstate.FieldCell && field.Name == "flag" {
							snapshot.Entries = []contractstate.Entry{{Key: field.Prefix, Value: []byte{1}}}
							snapshot.Version = 1
						}
					}
					if len(snapshot.Entries) != 1 {
						t.Fatal("missing seeded bool cell")
					}
				}
				original, e := prepared.Input(snapshot, graphNativeLimits())
				if e != nil {
					t.Fatal(e)
				}
				sum := sha256.Sum256(original.Bytes())
				if !bytes.Equal(got.blobs[2], original.Bytes()) || !bytes.Equal(got.blobs[3], sum[:]) {
					t.Fatal("Fairy request or its digest differs from live Go")
				}
				_, entryHash, _, _ := graphNativeOracle(t, f, true)
				if !bytes.Equal(got.blobs[4], entryHash[:]) {
					t.Fatal("audited entry identity differs")
				}
				switch trial.name {
				case "empty-commit", "empty-reject", "trap", "helper-transitive", "map-text-empty-bytes", "present-borrow-copy", "branch-present", "map-only", "seeded-branch":
					native++
					profile := cdriver.LowProfile
					if strings.Contains(t.Name(), "/optimized/") {
						profile = cdriver.OptimizedProfile
					}
					tc, e := cdriver.ResolveWindowsHostMSVC()
					if e != nil {
						t.Fatal(e)
					}
					support, e := cstdlib.ContractStateGraphRuntimeSource()
					if e != nil {
						t.Fatal(e)
					}
					built, e := cdriver.BuildWindowsMSVCTarget(tc, profile, string(got.blobs[0]), support)
					if e != nil {
						t.Fatal(e)
					}
					out, e := cdriver.ExecuteWindowsMSVCTargetWithInput(built, bytes.NewReader(got.blobs[2]))
					if e != nil || out.ExitCode != 0 || out.Stderr != "" {
						t.Fatal("Fairy frame execution", e, out)
					}
					result, e := prepared.Decode(original, []byte(out.Stdout))
					if e != nil {
						t.Fatal("actual framed response admission", e)
					}
					if mode == 16 {
						// One read, one branch terminator and one commit terminator.
						if result.Status != 0 || result.Exit != contractstatemir.ExitCommit || result.Version != 1 || result.Cost != 3 || result.Operations != 1 || len(result.Reads) != 1 || !result.Reads[0].Found || !bytes.Equal(result.Reads[0].Value.Encoded, []byte{1}) || len(result.Writes) != 0 {
							t.Fatalf("seeded branch semantic result %+v", result)
						}
					} else {
						graphNativeSemanticExpect(t, f.input, result)
					}
					// Complete request identity is checked by the actual C process.
					bad := bytes.Clone(got.blobs[2])
					bad[72] ^= 1
					rejected, e := cdriver.ExecuteWindowsMSVCTargetWithInput(built, bytes.NewReader(bad))
					if e != nil || rejected.ExitCode == 0 || rejected.Stdout != "" || rejected.Stderr != "" {
						t.Fatal("foreign request identity accepted", e, rejected)
					}
				}
				if trial.name == "helper-transitive" {
					restored := run(f, trial.text, runtimePack, framePack, 2, 511, [2]uint64{})
					if restored.code != 0 || !bytes.Equal(restored.blobs[0], got.blobs[0]) || !bytes.Equal(restored.blobs[1], got.blobs[1]) {
						t.Fatal("restored Complete framing", restored.code)
					}
					for _, v := range []struct {
						mode  uint64
						work  [2]uint64
						frame []byte
						want  uint64
					}{{8, [2]uint64{}, framePack, 1001}, {0, [2]uint64{16777217, 0}, framePack, 1002}, {0, [2]uint64{0, 16777217}, framePack, 1002}, {0, [2]uint64{}, framePack[:len(framePack)-1], 1001}} {
						p := run(f, trial.text, runtimePack, v.frame, v.mode, 0, v.work)
						if p.code != v.want {
							t.Fatal("refusal", p.code, v.want)
						}
					}
				}
			})
		}
		t.Logf("Fairy transport processes=%d; actual C frames=%d (each also refuses a foreign request)", processes, native)
	})
}

func TestCandidateStateGraphTransportSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, body string }{
		{"prepared-constructor", `fn transport::Prepared forge(){return transport::Prepared{.checked=0,.entry=0,.plan=0,.source=bytes::empty(),.program=bytes::empty()};}`},
		{"prepared-source", `fn void forge(ref mut transport::Prepared p){p.source=bytes::empty();}`},
		{"input-constructor", `fn input::Input forge(){return input::Input{.packet=0};}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_transport_request")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_transport as transport;import module_state_graph_transport_request as input;import std::bytes as bytes;"+trial.body+"fn i32 main(){return 0;}")
			workspace, e := workspaceloader.LoadVectorCandidate(root)
			if e != nil {
				t.Fatal(e)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed native graph framing", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "E0103") {
				t.Fatalf("sealed graph owner code=%d: %s", code, &diagnostics)
			}
		})
	}
}
