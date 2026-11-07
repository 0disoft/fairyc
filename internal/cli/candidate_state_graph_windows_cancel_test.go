//go:build windows

package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"golang.org/x/sys/windows"
)

// Reuse the real Prepared -> Built -> Store setup, without replacing any
// product module or introducing a runner/provider fixture.
func graphWindowsCancelProbe(t *testing.T) string {
	t.Helper()
	prefix, _, ok := strings.Cut(graphWindowsPersistentProbe(), " // All admission failures precede store.snapshot")
	if !ok {
		t.Fatal("persistent probe setup boundary changed")
	}
	prefix = strings.Replace(prefix, "module main;", "module main;import windows_cancel as cancel;", 1)
	return prefix + `
 var u8[512] event_name=[0;512];let event_size=switch(process::arg(2,ref mut event_name)){case .ok(p):{p}case .err(_):{return 30;}};let ref u8[] event_raw=ref event_name;
 let token=switch(cancel::open(event_raw[..event_size])){case .ok(p):{p}case .err(_):{return 31;}};
 assert(switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{false}});
 let snapshot=switch(response_snapshot(builder::prepared(ref built),hash,mode)){case .ok(p):{p}case .err(_):{return 32;}};
 if((mode&512)==0){
  // All runner admission errors must win over the already-latched token.
  var usize i=0;while(i<7){
   var work=runner::Work{.request_structure=7,.request_bytes=11,.response_structure=13,.response_bytes=17};var chosen=options;
   if(i==1){chosen.timeout_ms=0;}if(i==2){chosen.timeout_ms=600001;}
   if(i==3){work.request_structure=16777217;}if(i==4){work.request_bytes=1073741825;}if(i==5){work.response_structure=16777217;}if(i==6){work.response_bytes=1073741825;}
   let before=work;
   switch(runner::run_cancel(ref built,ref snapshot,limits,chosen,ref token,ref mut work)){case .ok(_):{return 33;}case .err(e):{
    assert(runner::stage(ref e)==if(i==0){9}else{1} && runner::status(ref e)==if(i==0){1009}else if(i<=2){1001}else{1002} && runner::os_code(ref e)==0 && runner::diagnostics(ref e).len==0);
   }};assert(same_work(work,before));i+=1;
  }
  // Valid cancellation and invalid options/limits/ledgers/counters all stop
  // before snapshot. Both cumulative ledgers and call counters are immutable.
  i=0;while(i<19){
   var work=persistent::empty_work();work.local=opening;work.execution=runner::Work{.request_structure=7,.request_bytes=11,.response_structure=13,.response_bytes=17};
   var chosen=options;var selected=limits;
   if(i==1){chosen.timeout_ms=0;}if(i==2){chosen.timeout_ms=600001;}
   if(i==3){selected.writes=0;}if(i==4){selected.bytes=0;}if(i==5){selected.value=0;}if(i==6){selected.key=0;}
   if(i==7){work.local.structure=16777217;}if(i==8){work.local.bytes=1073741825;}
   if(i==9){work.execution.request_structure=16777217;}if(i==10){work.execution.request_bytes=1073741825;}if(i==11){work.execution.response_structure=16777217;}if(i==12){work.execution.response_bytes=1073741825;}
   if(i==13){selected.writes=9223372036854775808;}if(i==14){selected.key=9223372036854775808;}
   if(i==15){work.calls.snapshots=16777216;}if(i==16){work.calls.runs=16777216;}if(i==17){work.calls.publishes=16777216;}if(i==18){selected.value=9223372036854775808;}
   let before=work;
   switch(persistent::run_cancel(ref built,ref instance,selected,chosen,ref token,ref mut work)){case .ok(_):{return 34;}case .err(e):{
    let cause=persistent::cause(ref e);let invalid_work=(i>=7 && i<=12)||(i>=15 && i<=17);
    assert(cause.stage==if(i==0){9}else{1} && cause.code==if(i==0){1009}else if(invalid_work){1002}else{1001});
    assert(cause.cancellation_kind==0 && cause.cancellation_os_code==0 && cause.runner_stage==0 && cause.runner_status==0 && cause.runner_os_code==0 && cause.runtime_status==0 && cause.storage.stage==0);
    switch(persistent::failure_admitted(ref e)){case .ok(_):{return 35;}case .err(code):{assert(code==0);}};
    switch(persistent::failure_runner(ref e)){case .ok(_):{return 36;}case .err(code):{assert(code==0);}};
   }};
   assert(work.local.structure==before.local.structure && work.local.bytes==before.local.bytes && same_work(work.execution,before.execution));
   assert(work.calls.snapshots==before.calls.snapshots && work.calls.runs==before.calls.runs && work.calls.publishes==before.calls.publishes);
   if(i<15 || i==18){assert(work.calls.snapshots==0 && work.calls.runs==0 && work.calls.publishes==0);}i+=1;
  }
  number(1009);number(7);number(19);return 0;
 }
 // The original APIs use none: the same external event stays latched while
 // both a real graph execution and the real persistent CAS succeed.
 var execution=runner::empty_work();let first=switch(runner::run(ref built,ref snapshot,limits,options,ref mut execution)){case .ok(p):{p}case .err(_):{return 37;}};
 assert(response::header(ref first).status==0 && response::header(ref first).exit==0);
 var work=persistent::empty_work();work.local=opening;
 let outcome=switch(persistent::run(ref built,ref instance,limits,options,ref mut work)){case .ok(p):{p}case .err(_):{return 38;}};
 let publication=persistent::publication(ref outcome);assert(publication.acknowledged && publication.consistent);
 assert(work.calls.snapshots==1 && work.calls.runs==1 && work.calls.publishes==1);
 assert(switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{false}});
 number(publication.receipt.version);number(publication.receipt.writes);
 // Both admitted owners outlive their input and the other native execution.
 incoming=bytes::empty();assert(response::header(ref first).status==0);result(persistent::admitted(ref outcome),ref work.execution);return 0;
}`
}

func TestCandidateStateGraphWindowsCancellation(t *testing.T) {
	probe := graphWindowsCancelProbe(t)
	t.Logf("cancellation probe sha256=%x", sha256.Sum256([]byte(probe)))
	for _, name := range []string{"windows_cancel", "module_state_graph_windows_build", "module_state_graph_windows_run", "module_state_graph_windows_persistent"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s.fy sha256=%x", name, sha256.Sum256(data))
	}
	var trial stateGraphAuthoredTrial
	for _, candidate := range graphTransportTrials() {
		if candidate.name == "helper-transitive" {
			trial = candidate
			break
		}
	}
	if trial.name == "" {
		t.Fatal("missing real helper-transitive graph")
	}
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatal(code)
	}
	schema := graphWindowsPersistentLayout(t, f)
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	t.Logf("runtime pack sha256=%x; frame pack sha256=%x", sha256.Sum256(runtimePack), sha256.Sum256(framePack))
	candidateAtlasProbeSize(t, "module_state_graph_windows_persistent", probe, 512<<10, func(t *testing.T, image string) {
		directory := t.TempDir()
		policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
		s, err := contractstate.CreateLocalPersistence(directory, schema, policy)
		if err != nil {
			t.Fatal(err)
		}
		s.Close()
		var unknown contractstate.Key
		unknown[0], unknown[31] = 255, 213
		seed := []contractstate.Entry{{Key: unknown, Value: []byte{0, 255, 13, 10}}}
		before := stateLocalCodecRecord(t, schema, 7, seed)
		statePath := filepath.Join(directory, "state-v1.bin")
		if err = os.WriteFile(statePath, before, 0600); err != nil {
			t.Fatal(err)
		}
		name, err := windows.UTF16PtrFromString(fmt.Sprintf("Local\\fairyc-cancel-%x", sha256.Sum256([]byte(directory))))
		if err != nil {
			t.Fatal(err)
		}
		// Independent Go-owned manual-reset event, initially signaled.
		event, err := windows.CreateEvent(nil, 1, 1, name)
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(event)
		reopen := func(t *testing.T, version uint64, entries []contractstate.Entry) {
			t.Helper()
			store, err := contractstate.OpenLocalPersistence(directory, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			snapshot, err := store.Snapshot(schema.Hash())
			if err != nil || snapshot.Version != version || !reflect.DeepEqual(snapshot.Entries, entries) {
				t.Fatalf("independent Go reopen=%+v want version=%d entries=%+v err=%v", snapshot, version, entries, err)
			}
		}
		for _, defaults := range []bool{false, true} {
			mode := uint64(2)
			if defaults {
				mode |= 512
			}
			exit, out := candidateAtlasProcess(t, image, graphWindowsRunPacket(f, trial.text, runtimePack, framePack, mode, 511, strings.Contains(t.Name(), "/optimized")), directory, windows.UTF16PtrToString(name))
			if exit != 0 {
				t.Fatalf("real cancellation/default probe defaults=%v exit=%d output=%q", defaults, exit, out)
			}
			if status, err := windows.WaitForSingleObject(event, 0); err != nil || status != windows.WAIT_OBJECT_0 {
				t.Fatal("external manual-reset token did not remain latched", status, err)
			}
			if !defaults {
				if out != "1009\n7\n19\n" {
					t.Fatalf("pre-cancel/admission assertions did not complete: %q", out)
				}
				after, err := os.ReadFile(statePath)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("pre-cancel changed actual persisted record", err)
				}
				reopen(t, 7, seed)
				t.Log("7 runner and 19 persistent cancellation/precedence cases; record byte-identical; independent Go reopen version=7")
				continue
			}
			r := graphResponseReader{t, bufio.NewReader(strings.NewReader(out))}
			version, writes := r.number(), r.number()
			remaining, err := io.ReadAll(r.r)
			if err != nil {
				t.Fatal(err)
			}
			admitted, _, got := graphResponseRead(t, string(remaining))
			if admitted != 0 || got.Version != 7 || version != 8 || writes == 0 || writes != uint64(len(got.Writes)) {
				t.Fatalf("default API receipt/admitted result=%d/%d %+v", version, writes, got)
			}
			got.Version = 0
			graphNativeSemanticExpect(t, f.input, got)
			reference, err := contractstate.NewReferencePersistence(schema, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = reference.CompareAndPublish(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: unknown, Value: seed[0].Value}}); err != nil {
				t.Fatal(err)
			}
			if _, err = reference.CompareAndPublish(schema.Hash(), 1, got.Writes); err != nil {
				t.Fatal(err)
			}
			want, err := reference.Snapshot(schema.Hash())
			if err != nil {
				t.Fatal(err)
			}
			reopen(t, 8, want.Entries)
			t.Logf("original build/run/persistent APIs succeed with external token latched; independent Go reopen version=%d writes=%d", version, writes)
		}
	})
}
