//go:build windows

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/target"
)

func commandSnapshotReportProbe(t *testing.T, other string) string {
	t.Helper()
	prefix, _, ok := strings.Cut(graphWindowsRunProbe(), " let tools=switch(tools::resolve())")
	if !ok {
		t.Fatal("Prepared setup boundary changed")
	}
	prefix = strings.Replace(prefix, "module main;", "module main;import module_state_command_snapshot_report as report;import module_state_command_options as options;import module_state_local_windows as store;import module_state_local_codec as codec;import std::process as process;", 1)
	prefix = strings.Replace(prefix, "fn i32 main(){", "fn i32 main(){var output=bytes::empty();{", 1)
	return prefix + fmt.Sprintf(`
 let proof=graph::program(core::checked_state_graph(transport::checked(ref prepared)));
 var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(p):{p}case .err(_):{return 20;}};let ref u8[] root=ref path;
 let limits=codec::Limits{.bytes=16777216,.entries=65536,.value=65536};var opening=codec::empty_work();
 let instance=switch(store::open(root[..size],program::layouts(proof),model::header(program::root(proof)).layout,limits,false,ref mut opening)){case .ok(p):{p}case .err(_):{return 21;}};
 let state=switch(store::snapshot(ref instance,hash,ref mut opening)){case .ok(p):{p}case .err(_):{return 22;}};
 var chosen=options::Options{.action=if((mode&512)!=0){1}else{3},.release,.store_index=1,.source_index=2};var usize work=19;
 output=switch(report::snapshot(chosen,ref prepared,ref state,ref mut work)){case .ok(p):{p}case .err(code):{number(code as usize);return 23;}};let cost=work-19;
 var usize exact=report::work_limit()-cost;
 let replay=switch(report::snapshot(chosen,ref prepared,ref state,ref mut exact)){case .ok(p):{p}case .err(_):{return 24;}};
 assert(exact==report::work_limit() && bytes::equal(bytes::view(ref replay),bytes::view(ref output)));
 var usize short=report::work_limit()-cost+1;
 switch(report::snapshot(chosen,ref prepared,ref state,ref mut short)){case .ok(_):{return 25;}case .err(code):{assert(code==1002 && short<=report::work_limit());}};
 var usize over=report::work_limit()+1;
 switch(report::snapshot(chosen,ref prepared,ref state,ref mut over)){case .ok(_):{return 26;}case .err(code):{assert(code==1002 && over==report::work_limit()+1);}};
 var usize i=0;while(i<4){chosen.action=if(i==0){0}else if(i==1){2}else if(i==2){4}else{4294967295};var usize invalid=7;
  switch(report::snapshot(chosen,ref prepared,ref state,ref mut invalid)){case .ok(_):{return 27;}case .err(code):{assert(code==1001 && invalid==7);}};i+=1;
 }
 chosen.action=3;
 let different=switch(current(b%q,ref table,owner,false)){case .ok(p):{p}case .err(_):{return 28;}};
 let other=graph::program(core::checked_state_graph(ref different));var mismatch_work=codec::empty_work();
 let mismatch=switch(codec::initial(program::layouts(other),model::header(program::root(other)).layout,limits,ref mut mismatch_work)){case .ok(p):{p}case .err(_):{return 29;}};
 var usize invalid=7;switch(report::snapshot(chosen,ref prepared,ref mismatch,ref mut invalid)){case .ok(_):{return 30;}case .err(code):{assert(code==1001 && invalid==7);}};
 // All source, Prepared, real store/State and asset owners leave scope before
 // observing output. The serializer accepts none of the Go expected JSON.
 incoming=bytes::empty();
 }assert(io::write_bytes(bytes::view(ref output)).is_ok);return 0;
}`, other)
}

func TestCandidateStateCommandSnapshotReport(t *testing.T) {
	var trial stateGraphAuthoredTrial
	var other string
	for _, item := range graphTransportTrials() {
		if item.name == "helper-transitive" {
			trial = item
		}
		if item.name == "empty-commit" {
			other = item.text
		}
	}
	if trial.name == "" || other == "" {
		t.Fatal("missing authored graph fixtures")
	}
	other = strings.Replace(other, "fixture.authored-graph", "fixture.snapshot-report-other", 1)
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatal(code)
	}
	schema := graphWindowsPersistentLayout(t, f)
	_, _, _, handoff := graphNativeOracle(t, f, true)
	prepared, err := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	candidateAtlasProbeSize(t, "module_state_command_snapshot_report", commandSnapshotReportProbe(t, other), 512<<10, func(t *testing.T, image string) {
		for _, action := range []string{"init", "snapshot"} {
			for _, version := range []uint64{0, math.MaxUint64} {
				for _, seeded := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/seeded-%v", action, version, seeded), func(t *testing.T) {
						directory := t.TempDir()
						policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
						instance, err := contractstate.CreateLocalPersistence(directory, schema, policy)
						if err != nil {
							t.Fatal(err)
						}
						instance.Close()
						entries := []contractstate.Entry{}
						if seeded {
							var a, b, c contractstate.Key
							a[0], a[31] = 0, 255
							b[0], b[31] = 128, 13
							c[0], c[31] = 255, 213
							entries = []contractstate.Entry{{Key: a, Value: nil}, {Key: b, Value: []byte{}}, {Key: c, Value: []byte{0, 255, 13, 10, 34, 92, 1}}}
						}
						record := stateLocalCodecRecord(t, schema, version, entries)
						path := filepath.Join(directory, "state-v1.bin")
						if err := os.WriteFile(path, record, 0600); err != nil {
							t.Fatal(err)
						}
						// The independent Go report uses an actual reopened provider snapshot.
						reopened, err := contractstate.OpenLocalPersistence(directory, schema, policy)
						if err != nil {
							t.Fatal(err)
						}
						snapshot, err := reopened.Snapshot(schema.Hash())
						reopened.Close()
						if err != nil {
							t.Fatal(err)
						}
						mode := uint64(2)
						if action == "init" {
							mode |= 512
						}
						release := strings.Contains(t.Name(), "/optimized/")
						exit, out := candidateAtlasProcess(t, image, graphWindowsRunPacket(f, trial.text, runtimePack, framePack, mode, 511, release), directory)
						if exit != 0 {
							t.Fatalf("native report exit=%d output=%q", exit, out)
						}
						profile := "low"
						if release {
							profile = "optimized"
						}
						// Fields and omitempty behavior mirror tools/dev/fairyc-state's embedded
						// report/nativeReport; JSON numbers are never decoded through float64.
						expected := struct {
							Schema      string                            `json:"schema"`
							Action      string                            `json:"action"`
							LayoutHash  string                            `json:"layoutHash"`
							Snapshot    *contractstate.PersistentSnapshot `json:"snapshot,omitempty"`
							Engine      string                            `json:"engine"`
							Profile     string                            `json:"profile"`
							BodyHash    string                            `json:"bodyHash"`
							ProgramHash string                            `json:"programHash"`
						}{"fairyc.local-state-run/v2", action, schema.Hash().String(), &snapshot, "native-graph", profile, prepared.BodyHash().String(), fmt.Sprintf("%x", prepared.ProgramHash())}
						want, err := json.Marshal(expected)
						if err != nil {
							t.Fatal(err)
						}
						want = append(want, '\n')
						if !bytes.Equal([]byte(out), want) {
							t.Fatalf("JSON mismatch\ngot %s\nwant %s", out, want)
						}
						var decoded struct {
							Snapshot contractstate.PersistentSnapshot `json:"snapshot"`
						}
						if err := json.Unmarshal([]byte(out), &decoded); err != nil || !reflect.DeepEqual(decoded.Snapshot, snapshot) {
							t.Fatalf("snapshot semantic mismatch: %+v err=%v", decoded, err)
						}
						after, err := os.ReadFile(path)
						if err != nil || !bytes.Equal(record, after) {
							t.Fatal("serializer changed durable bytes", err)
						}
						t.Logf("real Prepared program=%x; actual snapshot version=%d entries=%d; owned JSON=%d bytes", prepared.ProgramHash(), version, len(entries), len(out))
					})
				}
			}
		}
	}, "module_state_graph_windows_persistent")
}
