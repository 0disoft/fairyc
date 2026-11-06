package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const stateGraphNativeProbe = `module main;
import module_state_graph_native as native;import module_state_graph_native_assets as assets;import project_compiler_core as core;import project_compiler_state_graph as graph;import module_state_graph_source as source;import module_state_graph_source_artifact as saved;import module_public_type_text as coordinates;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{bytes::empty()}};}
@effects(console)fn void number(usize n){assert(io::write_usize(n).is_ok);assert(io::write("\n").is_ok);}
@effects(console)fn void blob(ref u8[] raw){number(raw.len);assert(io::write_bytes(raw).is_ok);assert(io::write("\n").is_ok);}
@effects(console)fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] b=ref block;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,b[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let mode=wire::number_from(raw,ref mut c,ref mut setup);let owner=wire::number_from(raw,ref mut c,ref mut setup);var work=wire::number_from(raw,ref mut c,ref mut setup);let size=wire::number_from(raw,ref mut c,ref mut setup);let length=wire::number_from(raw,ref mut c,ref mut setup);let packed=wire::number_from(raw,ref mut c,ref mut setup);
 if(c.code!=0 || size>raw.len-c.offset || length>raw.len-c.offset-size || packed!=raw.len-c.offset-size-length){return 3;}let coordinate=raw[c.offset..c.offset+size];let actual=raw[c.offset+size..c.offset+size+length];let bundle=raw[c.offset+size+length..];
 // Own the output beyond both Complete and runtime-bundle lifetimes.
 let artifact={var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=coordinate.len}).is_ok);let table=coordinates::assemble(copy(coordinate),rows);
  var usize source_work=0;let checked=if((mode&2)!=0){let current=switch(source::produce(actual,ref mut source_work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work);return 0;}};source_work=0;let packet=switch(saved::encode(ref current,ref mut source_work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work);return 0;}};source_work=0;switch(core::restore_complete_state_graph(wire::content(ref packet),actual,ref table,owner,ref mut source_work)){case .ok(p):{p}case .err(p):{number(p.code as usize);number(work);return 0;}}}else{switch(core::build_complete_state_graph(actual,ref table,owner,ref mut source_work)){case .ok(p):{p}case .err(p):{number(p.code as usize);number(work);return 0;}}};
  var usize asset_work=0;let runtime=switch(assets::load(copy(bundle),ref mut asset_work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work);return 0;}};
  let target=if((mode&8)!=0){b"x86_64-linux-gnu"}else{b"x86_64-windows-msvc"};let audit=(mode&1)!=0;
  let result=switch(native::emit(ref checked,ref runtime,target,audit,ref mut work)){case .ok(p):{p}case .err(code):{number(code as usize);number(work);return 0;}};
  if((mode&4)!=0){let cost=work;var usize exact=16777216-cost;let repeated=switch(native::emit(ref checked,ref runtime,target,audit,ref mut exact)){case .ok(p):{p}case .err(_):{return 4;}};if(exact!=16777216 || !bytes::equal(native::source(ref result),native::source(ref repeated)) || !bytes::equal(native::program_hash(ref result),native::program_hash(ref repeated))){return 5;}
   exact=16777217-cost;switch(native::emit(ref checked,ref runtime,target,audit,ref mut exact)){case .ok(_):{return 6;}case .err(code):{if(code!=1002 || exact>16777216){return 7;}}};exact=16777217;switch(native::emit(ref checked,ref runtime,target,audit,ref mut exact)){case .ok(_):{return 8;}case .err(code):{if(code!=1002 || exact!=16777217){return 9;}}};
  }result};
 // Input mutation cannot rewrite the already owned C or any receipt identity.
 {let changed=bytes::view_mut(ref mut input);changed[0]=255;}input=bytes::empty();
 number(0);number(work);blob(native::source(ref artifact));blob(native::layout_hash(ref artifact));blob(native::body_hash(ref artifact));blob(native::program_hash(ref artifact));blob(native::trap_json(ref artifact));number(if(native::audited(ref artifact)){1}else{0});return 0;
}`

type graphNativeResult struct {
	code, work, audit uint64
	blobs             [5][]byte
}

func graphNativeLimits() contractstate.Limits {
	return contractstate.Limits{MaxWrites: 64, MaxWriteBytes: 65536, MaxValueBytes: 4096, MaxMapKeyBytes: 4096}
}

func graphNativeRead(t *testing.T, raw string) graphNativeResult {
	t.Helper()
	r := bufio.NewReader(strings.NewReader(raw))
	n := func() uint64 {
		t.Helper()
		line, e := r.ReadString('\n')
		if e != nil {
			t.Fatalf("native number: %v %q", e, raw)
		}
		v, e := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	out := graphNativeResult{code: n(), work: n()}
	if out.code == 0 {
		for i := range out.blobs {
			size := n()
			if size > 8<<20 {
				t.Fatal("output bound", size)
			}
			out.blobs[i] = make([]byte, size)
			if _, e := io.ReadFull(r, out.blobs[i]); e != nil {
				t.Fatal(e)
			}
			if b, e := r.ReadByte(); e != nil || b != '\n' {
				t.Fatal("blob delimiter", b, e)
			}
		}
		out.audit = n()
	}
	if extra, _ := io.ReadAll(r); len(extra) != 0 {
		t.Fatalf("unexpected suffix %q", extra)
	}
	return out
}

func graphNativePacket(f stateBodyFixture, actual string, bundle []byte, mode, owner, work uint64) []byte {
	coordinate := "fypc1-sha256-" + hex.EncodeToString(f.coordinate[:])
	var out []byte
	for _, v := range []uint64{mode, owner, work, uint64(len(coordinate)), uint64(len(actual)), uint64(len(bundle))} {
		out = binary.BigEndian.AppendUint64(out, v)
	}
	out = append(out, coordinate...)
	out = append(out, actual...)
	return append(out, bundle...)
}

func graphNativeOracle(t *testing.T, f stateBodyFixture, audit bool) (string, [32]byte, []byte, contractstatenative.Handoff) {
	t.Helper()
	lowered, e := contractstatemir.LowerHIRTransactions(f.input)
	if e != nil || len(lowered) != 1 {
		t.Fatal(e)
	}
	h, e := contractstatenative.Prepare(f.input, hir.PackageID(7), lowered[0].Function, f.coordinate)
	if e != nil {
		t.Fatal(e)
	}
	a, e := contractstatenative.EmitGraph(h, target.X8664WindowsMSVC)
	if e != nil {
		t.Fatal(e)
	}
	source := a.Source()
	var traps []string
	bodies := [][]contractstatemir.Block{lowered[0].Program.Blocks}
	for _, helper := range lowered[0].Program.Helpers {
		bodies = append(bodies, helper.Blocks)
	}
	for _, blocks := range bodies {
		for _, b := range blocks {
			if b.Terminator.TrapCategory != "" {
				traps = append(traps, b.Terminator.TrapCategory)
			}
		}
	}
	slices.Sort(traps)
	traps = slices.Compact(traps)
	metadata, e := json.Marshal(traps)
	if e != nil {
		t.Fatal(e)
	}
	hash := a.ProgramHash()
	if audit {
		transport, e := contractstatenative.PrepareGraphTransport(h, target.X8664WindowsMSVC)
		if e != nil {
			t.Fatal(e)
		}
		framed, e := transport.Source()
		if e != nil {
			t.Fatal(e)
		}
		source, _, _ = strings.Cut(framed, "\n#include <string.h>\nstatic const uint8_t fygr_identity[96]")
		if source == framed {
			t.Fatal("missing frame boundary")
		}
		layout, body := a.LayoutHash(), a.BodyHash()
		identity := append([]byte("fairyc.native-state-graph-entry/v1\x00"+string(target.X8664WindowsMSVC)+"\x00"), layout[:]...)
		identity = append(identity, body[:]...)
		identity = append(identity, metadata...)
		hash = sha256.Sum256(append(identity, source...))
	}
	return source, hash, metadata, h
}

func TestCandidateStateGraphNativeEmission(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native graph emission; Linux execution remains batched")
	}
	var trials []stateGraphAuthoredTrial
	for _, p := range stateGraphSourceTrials() {
		if p.code == "" {
			trials = append(trials, stateGraphAuthoredTrial{p.name, p.text, ""})
		}
	}
	trials = append(trials, stateGraphAuthoredTrials()...)
	trials = append(trials, stateGraphAuthoredTrial{"read-written-branch", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("flag",true);let f=storage::read("flag");if(f){storage::write("count",42);}else{storage::write("count",99);}storage::commit();`, ""), ""})
	trials = append(trials,
		stateGraphAuthoredTrial{"present-borrow-view-copy", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("blob",b"\x00\xffx");let raw=storage::borrow("blob");let part=storage::view(raw,1,2);let copied=storage::copy(part);storage::write("blob",copied);storage::commit();`, ""), ""},
		stateGraphAuthoredTrial{"dynamic-map-key-value", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("title","한글");let key=storage::read("title");storage::write("labels",key,b"\x00\xff");let value=storage::read("labels",key);storage::write("blob",value);storage::commit();`, ""), ""})
	bundlePath := t.TempDir() + "/graph-runtime.pack"
	cmd := exec.Command("python", "-B", "../../tools/package-candidate-state-graph-runtime.py", "--output", bundlePath)
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("runtime pack: %v %s", e, out)
	}
	bundle, e := os.ReadFile(bundlePath)
	if e != nil {
		t.Fatal(e)
	}
	candidateAtlasProbeSize(t, "module_state_graph_native", stateGraphNativeProbe, 512<<10, func(t *testing.T, image string) {
		var processes int
		run := func(f stateBodyFixture, actual string, packed []byte, mode, owner, work uint64) graphNativeResult {
			t.Helper()
			processes++
			code, out := candidateAtlasProcess(t, image, graphNativePacket(f, actual, packed, mode, owner, work))
			if code != 0 {
				t.Fatalf("native exit=%d output=%q", code, out)
			}
			return graphNativeRead(t, out)
		}
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				f, code := stateGraphAuthoredOracle(t, trial)
				if code != "" {
					p := run(f, trial.text, bundle, 0, 0, 0)
					if strconv.FormatUint(p.code, 10) != code {
						t.Fatalf("admission got%d want%s", p.code, code)
					}
					return
				}
				for _, audit := range []bool{false, true} {
					mode := uint64(0)
					if audit {
						mode = 1
					}
					if trial.name == "read-written-branch" || trial.name == "helper-transitive" || trial.name == "trap" {
						mode |= 4
					}
					p := run(f, trial.text, bundle, mode, 0, 0)
					want, hash, metadata, h := graphNativeOracle(t, f, audit)
					if p.code != 0 {
						t.Fatalf("emission code=%d work=%d", p.code, p.work)
					}
					if string(p.blobs[0]) != want {
						at := 0
						for at < len(p.blobs[0]) && at < len(want) && p.blobs[0][at] == want[at] {
							at++
						}
						t.Fatalf("C bytes differ at%d candidate%d Go%d: got%q want%q", at, len(p.blobs[0]), len(want), p.blobs[0][max(0, at-60):min(len(p.blobs[0]), at+100)], want[max(0, at-60):min(len(want), at+100)])
					}
					a, _ := contractstatenative.EmitGraph(h, target.X8664WindowsMSVC)
					layout, body := a.LayoutHash(), a.BodyHash()
					if !bytes.Equal(p.blobs[1], layout[:]) || !bytes.Equal(p.blobs[2], body[:]) || !bytes.Equal(p.blobs[3], hash[:]) || !bytes.Equal(p.blobs[4], metadata) || p.audit != mode&1 {
						t.Fatalf("artifact identity mismatch %+v", p)
					}
					switch trial.name {
					case "read-written-branch", "helper-transitive", "borrow-alias-view-copy", "copy-survives-mutation", "empty-commit", "empty-reject", "trap", "map-text-empty-bytes", "bytes-binary", "text-unicode", "present-borrow-view-copy", "dynamic-map-key-value":
						graphNativeExecute(t, h, f.input, string(p.blobs[0]), audit)
					}
				}
				if trial.name == "helper-transitive" {
					p := run(f, trial.text, bundle, 2, 511, 0)
					want, _, _, _ := graphNativeOracle(t, f, false)
					if p.code != 0 || string(p.blobs[0]) != want {
						t.Fatal("restored owner emission", p.code)
					}
					for _, v := range []struct {
						mode, work uint64
						packed     []byte
						want       uint64
					}{{8, 0, bundle, 1001}, {0, 16777216, bundle, 1002}, {0, 16777217, bundle, 1002}, {0, 0, bundle[:len(bundle)-1], 1001}} {
						p := run(f, trial.text, v.packed, v.mode, 0, v.work)
						if p.code != v.want {
							t.Fatal("refusal", p.code, v.want)
						}
					}
				}
			})
		}
		t.Logf("Fairy emission/refusal processes=%d", processes)
	})
}

func TestCandidateStateGraphNativeSealedArtifact(t *testing.T) {
	for _, trial := range []struct{ name, body string }{
		{"constructor", `fn native::Artifact forge(){return native::Artifact{.source=bytes::empty(),.layout=bytes::empty(),.body=bytes::empty(),.program=bytes::empty(),.traps=bytes::empty(),.audit=false};}`},
		{"mutate-source", `fn void forge(ref mut native::Artifact p){p.source=bytes::empty();}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_native")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_native as native;import std::bytes as bytes;"+trial.body+"fn i32 main(){return 0;}")
			workspace, e := workspaceloader.LoadVectorCandidate(root)
			if e != nil {
				t.Fatal(e)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed graph native Artifact", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), "E0103") {
				t.Fatalf("forged Artifact code=%d diagnostic=%s", code, &diagnostic)
			}
		})
	}
}

func graphNativeExecute(t *testing.T, h contractstatenative.Handoff, originalHIR hir.Package, candidate string, audit bool) {
	t.Helper()
	// The framing program is an existing independent test driver. The operation
	// source under test is always the Fairy-emitted buffer, never regenerated Go C.
	prepared, e := contractstatenative.PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if e != nil {
		t.Fatal(e)
	}
	framed, e := prepared.Source()
	if e != nil {
		t.Fatal(e)
	}
	var suffix string
	var input []byte
	if audit {
		_, tail, ok := strings.Cut(framed, "\n#include <string.h>\nstatic const uint8_t fygr_identity[96]")
		if !ok {
			t.Fatal("frame boundary")
		}
		suffix = "\n#include <string.h>\nstatic const uint8_t fygr_identity[96]" + tail
		original, e := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: graphNativeLayout(t, h)}, graphNativeLimits())
		if e != nil {
			t.Fatal(e)
		}
		input = original.Bytes()
	} else {
		raw, e := os.ReadFile("../contractstatenative/graph_execute_probe.c")
		if e != nil {
			t.Fatal(e)
		}
		suffix = string(raw)
		l := graphNativeLimits()
		for _, n := range []uint64{uint64(l.MaxWrites), l.MaxWriteBytes, uint64(l.MaxValueBytes), uint64(l.MaxMapKeyBytes)} {
			input = binary.BigEndian.AppendUint64(input, n)
		}
	}
	// Byte equality was checked above. Compile only the actual Fairy output;
	// audited execution is compared with the independent Go MIR interpreter.
	profile := cdriver.LowProfile
	if strings.Contains(t.Name(), "/optimized/") {
		profile = cdriver.OptimizedProfile
	}
	graphNativeExecuteImage(t, h, originalHIR, candidate, suffix, input, prepared, audit, profile)
}

func graphNativeLayout(t *testing.T, h contractstatenative.Handoff) contractstate.LayoutHash {
	t.Helper()
	a, e := contractstatenative.EmitGraph(h, target.X8664WindowsMSVC)
	if e != nil {
		t.Fatal(e)
	}
	return a.LayoutHash()
}

func graphNativeExecuteImage(t *testing.T, h contractstatenative.Handoff, originalHIR hir.Package, candidate, suffix string, input []byte, prepared contractstatenative.GraphTransport, audit bool, profile cdriver.Profile) {
	t.Helper()
	runtime, e := cstdlib.ContractStateGraphRuntimeSource()
	if e != nil {
		t.Fatal(e)
	}
	tc, e := cdriver.ResolveWindowsHostMSVC()
	if e != nil {
		t.Fatal(e)
	}
	image, e := cdriver.BuildWindowsMSVCTarget(tc, profile, candidate+suffix, runtime)
	if e != nil {
		t.Fatal("compile genuine operation C", e)
	}
	out, e := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input))
	if e != nil || out.ExitCode != 0 || out.Stderr != "" {
		t.Fatal("execute operation C", e, out)
	}
	if audit {
		request, e := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: graphNativeLayout(t, h)}, graphNativeLimits())
		if e != nil {
			t.Fatal(e)
		}
		result, e := prepared.Decode(request, []byte(out.Stdout))
		if e != nil {
			t.Fatal("Go path witness decoder refused actual Fairy C", e)
		}
		graphNativeSemanticExpect(t, originalHIR, result)
		t.Logf("actual %s C exit=%s cost=%d reads=%d writes=%d trap=%s", profile.Name, result.Exit, result.Cost, len(result.Reads), len(result.Writes), result.TrapCategory)
	} else if len(out.Stdout) < 28 {
		t.Fatal("short legacy observer execution record")
	}
}
