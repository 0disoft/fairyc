//go:build windows

package cli

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"golang.org/x/sys/windows"
)

func graphWindowsPersistentProbe() string {
	source := strings.Split(graphWindowsRunProbe(), " assert(builder::object_count(ref built)==2);")[0]
	source = strings.Replace(source, "module main;", `module main;import module_state_graph_windows_persistent as persistent;import module_state_local_windows as store;import module_state_local_codec as codec;import std::process as process;`, 1)
	return source + `
 assert(builder::object_count(ref built)==2);
 var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(p):{p}case .err(_):{return 20;}};let ref u8[] root=ref path;
 let proof=graph::program(core::checked_state_graph(transport::checked(builder::prepared(ref built))));
 var opening=codec::empty_work();let instance=switch(store::open(root[..size],program::layouts(proof),model::header(program::root(proof)).layout,codec::Limits{.bytes=16777216,.entries=65536,.value=65536},false,ref mut opening)){case .ok(p):{p}case .err(e):{number(803);number(e.stage as usize);number(e.code as usize);number(e.os_code as usize);return 0;}};
 var limits=request::Limits{.writes=64,.bytes=65536,.value=4096,.key=4096};if((mode&256)!=0){limits.writes=1;}let options=runner::Options{.timeout_ms=60000};
 // All admission failures precede store.snapshot, and leave both ledgers intact.
 var usize i=0;while(i<14){var bad=persistent::empty_work();var selected=limits;var chosen=options;
  if(i==0){chosen.timeout_ms=0;}if(i==1){chosen.timeout_ms=600001;}
  if(i==2){selected.writes=0;}if(i==3){selected.bytes=0;}if(i==4){selected.value=0;}if(i==5){selected.key=0;}
  if(i==6){bad.local.structure=16777217;}if(i==7){bad.local.bytes=1073741825;}
  if(i==8){bad.execution.request_structure=16777217;}if(i==9){bad.execution.request_bytes=1073741825;}if(i==10){bad.execution.response_structure=16777217;}if(i==11){bad.execution.response_bytes=1073741825;}
  if(i==12){selected.writes=9223372036854775808;}if(i==13){selected.key=9223372036854775808;}
  let before=bad;
  switch(persistent::run(ref built,ref instance,selected,chosen,ref mut bad)){case .ok(_):{return 21;}case .err(e):{let cause=persistent::cause(ref e);assert(cause.stage==1 && cause.code==if(i>=6 && i<=11){1002}else{1001});switch(persistent::failure_admitted(ref e)){case .ok(_):{return 22;}case .err(code):{assert(code==0);}};}};
  assert(bad.calls.snapshots==0 && bad.calls.runs==0 && bad.calls.publishes==0 && bad.local.structure==before.local.structure && bad.local.bytes==before.local.bytes && same_work(bad.execution,before.execution));i+=1;
 }
 var work=persistent::empty_work();work.local=opening;work.execution=runner::Work{.request_structure=7,.request_bytes=11,.response_structure=13,.response_bytes=17};
 // Exercise a real runner request-admission failure and its borrowed cause API.
 var short=persistent::empty_work();short.execution.request_structure=16777216;
 switch(persistent::run(ref built,ref instance,limits,options,ref mut short)){case .ok(_):{return 24;}case .err(e):{let cause=persistent::cause(ref e);assert(cause.stage==4 && cause.runner_stage==2 && cause.runner_status==1002 && cause.runner_os_code==0);switch(persistent::failure_runner(ref e)){case .ok(error):{assert(runner::stage(error)==2 && runner::status(error)==1002 && runner::diagnostics(error).len==0);}case .err(_):{return 25;}};switch(persistent::failure_admitted(ref e)){case .ok(_):{return 26;}case .err(_):{}};}};
 assert(short.calls.snapshots==1 && short.calls.runs==1 && short.calls.publishes==0);
 let outcome=switch(persistent::run(ref built,ref instance,limits,options,ref mut work)){case .ok(p):{p}case .err(e):{
  let cause=persistent::cause(ref e);number(1);number(cause.stage as usize);number(cause.code as usize);number(cause.storage.stage as usize);number(cause.storage.code as usize);number(cause.storage.os_code as usize);number(cause.storage.operation as usize);number(if(cause.storage.observed){1}else{0});number(cause.storage.current.version);number(cause.runner_stage as usize);number(cause.runner_status as usize);number(cause.runner_os_code as usize);number(cause.runtime_status);number(work.calls.snapshots);number(work.calls.runs);number(work.calls.publishes);
  switch(persistent::failure_admitted(ref e)){case .ok(p):{number(1);result(p,ref work.execution);}case .err(_):{number(0);}};return 0;
 }};
 let publication=persistent::publication(ref outcome);number(0);number(if(publication.acknowledged){1}else{0});number(if(publication.consistent){1}else{0});number(publication.receipt.version);number(publication.receipt.writes);var usize identity_work=0;let receipt_hash=switch(identities::words(publication.receipt.layout,ref mut identity_work)){case .ok(p):{p}case .err(_):{return 23;}};blob(bytes::view(ref receipt_hash));number(work.calls.snapshots);number(work.calls.runs);number(work.calls.publishes);number(work.local.structure);number(work.local.bytes);
 assert(work.local.structure>opening.structure && work.local.bytes>opening.bytes && work.execution.request_structure>7 && work.execution.request_bytes>11 && work.execution.response_structure>13 && work.execution.response_bytes>17);
 // The response remains owned after discarding all incoming source/asset bytes.
 incoming=bytes::empty();result(persistent::admitted(ref outcome),ref work.execution);return 0;
}`
}

func graphWindowsPersistentLayout(t *testing.T, f stateBodyFixture) contractstate.Layout {
	t.Helper()
	lowered, err := contractstatemir.LowerHIRTransactions(f.input)
	if err != nil || len(lowered) != 1 {
		t.Fatal(err)
	}
	declaration := lowered[0].Program.Layout
	b, err := contractstate.NewLayoutBuilder(declaration.ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range declaration.Fields {
		if field.Kind == contractstate.FieldCell {
			_, err = contractstate.DeclareEncodedCell(b, field.Name, field.ValueTypeID)
		} else {
			_, err = contractstate.DeclareEncodedMap(b, field.Name, field.KeyTypeID, field.ValueTypeID)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	schema, err := b.Build()
	if err != nil || schema.Hash() != declaration.Hash {
		t.Fatal("layout mismatch", err)
	}
	return schema
}

func TestCandidateStateGraphWindowsPersistent(t *testing.T) {
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	var trials []stateGraphAuthoredTrial
	for _, trial := range graphTransportTrials() {
		switch trial.name {
		case "empty-commit", "empty-reject", "trap", "helper-transitive", "present-borrow-copy", "map-local-key-value":
			trials = append(trials, trial)
		}
	}
	if len(trials) != 6 {
		t.Fatalf("representative graphs=%d", len(trials))
	}
	candidateAtlasProbeSize(t, "module_state_graph_windows_persistent", graphWindowsPersistentProbe(), 512<<10, func(t *testing.T, image string) {
		run := func(t *testing.T, trial stateGraphAuthoredTrial, version uint64, readonly bool) {
			f, code := stateGraphAuthoredOracle(t, trial)
			if code != "" {
				t.Fatal(code)
			}
			schema := graphWindowsPersistentLayout(t, f)
			directory := t.TempDir()
			policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
			s, err := contractstate.CreateLocalPersistence(directory, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			s.Close()
			var unknown contractstate.Key
			unknown[0] = 255
			unknown[31] = 213
			seed := []contractstate.Entry{{Key: unknown, Value: []byte{0, 255, 13, 10}}}
			before := stateLocalCodecRecord(t, schema, version, seed)
			statePath := filepath.Join(directory, "state-v1.bin")
			if err = os.WriteFile(statePath, before, 0600); err != nil {
				t.Fatal(err)
			}
			if readonly {
				name, err := windows.UTF16PtrFromString(statePath)
				if err != nil {
					t.Fatal(err)
				}
				if err = windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_READONLY); err != nil {
					t.Fatal(err)
				}
				defer windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL)
			}
			mode, owner := uint64(0), uint64(0)
			if trial.name == "helper-transitive" {
				mode = 2
				owner = 511
			}
			if trial.name == "runtime-write-limit" {
				mode = 256
			}
			exit, out := candidateAtlasProcess(t, image, graphWindowsRunPacket(f, trial.text, runtimePack, framePack, mode, owner, strings.Contains(t.Name(), "/optimized/")), directory)
			if exit != 0 {
				t.Fatalf("Fairy persistent exit=%d output=%q", exit, out)
			}
			r := graphResponseReader{t, bufio.NewReader(strings.NewReader(out))}
			failure := r.number()
			maxRefusal := version == math.MaxUint64 && trial.name == "helper-transitive"
			runtimeRefusal := trial.name == "runtime-write-limit"
			acknowledged := false
			receiptVersion, receiptWrites := uint64(0), uint64(0)
			if maxRefusal || runtimeRefusal || readonly {
				if failure != 1 {
					t.Fatalf("expected typed refusal: %s", out)
				}
				stage, code := r.number(), r.number()
				storageStage, storageCode, osCode, operation, observed, current := r.number(), r.number(), r.number(), r.number(), r.number(), r.number()
				for i := 0; i < 3; i++ {
					if r.number() != 0 {
						t.Fatal("unexpected runner cause")
					}
				}
				runtimeStatus := r.number()
				if maxRefusal && (stage != 6 || code != 1006 || runtimeStatus != 0 || storageStage != 0) {
					t.Fatal("max cause", out)
				}
				if runtimeRefusal && (stage != 8 || code != 0 || runtimeStatus != 4 || storageStage != 0) {
					t.Fatal("runtime cause", out)
				}
				if readonly && (stage != 7 || storageStage != 8 || storageCode != 0 || osCode == 0 || operation == 0 || observed != 1 || current != version) {
					t.Fatal("rename cause", out)
				}
				if r.number() != 1 || r.number() != 1 || r.number() != bool64(readonly) || r.number() != 1 {
					t.Fatal("failure attempt counts/admitted owner")
				}
			} else {
				if failure != 0 {
					t.Fatalf("Fairy persistence failure: %s", out)
				}
				acknowledged = r.number() == 1
				if r.number() != 1 {
					t.Fatal("receipt mismatch")
				}
				receiptVersion, receiptWrites = r.number(), r.number()
				hash := r.blob()
				if acknowledged {
					h := schema.Hash()
					if !bytes.Equal(hash, h[:]) {
						t.Fatal("receipt layout")
					}
				}
				snapshots, runs, publishes := r.number(), r.number(), r.number()
				if snapshots != 1 || runs != 1 || publishes != bool64(acknowledged) {
					t.Fatalf("one attempt counts=%d/%d/%d", snapshots, runs, publishes)
				}
				if r.number() == 0 || r.number() == 0 {
					t.Fatal("empty local work")
				}
			}
			remaining, err := io.ReadAll(r.r)
			if err != nil {
				t.Fatal(err)
			}
			admitted, _, got := graphResponseRead(t, string(remaining))
			if admitted != 0 || got.Version != version {
				t.Fatalf("response code/version=%d/%d", admitted, got.Version)
			}
			actualVersion := got.Version
			got.Version = 0
			if runtimeRefusal {
				if got.Status != 4 || got.Exit != "" || len(got.Writes) != 0 || got.WriteBytes != 0 || got.Operations != 2 || got.Cost == 0 {
					t.Fatalf("write-limit response evidence=%+v", got)
				}
			} else {
				graphNativeSemanticExpect(t, f.input, got)
			}
			got.Version = actualVersion
			commit := got.Status == 0 && got.Exit == contractstatemir.ExitCommit && !maxRefusal && !readonly
			if acknowledged != commit {
				t.Fatalf("publication=%v outcome=%+v", acknowledged, got)
			}
			reopened, err := contractstate.OpenLocalPersistence(directory, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			snapshot, err := reopened.Snapshot(schema.Hash())
			if err != nil {
				t.Fatal(err)
			}
			expectedVersion := version
			if commit && len(got.Writes) != 0 {
				expectedVersion++
			}
			if snapshot.Version != expectedVersion {
				t.Fatalf("reopen version=%d want=%d", snapshot.Version, expectedVersion)
			}
			if commit && (receiptVersion != expectedVersion || receiptWrites != uint64(len(got.Writes))) {
				t.Fatal("receipt version/writes")
			}
			reference, err := contractstate.NewReferencePersistence(schema, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = reference.CompareAndPublish(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: unknown, Value: seed[0].Value}}); err != nil {
				t.Fatal(err)
			}
			if commit {
				if _, err = reference.CompareAndPublish(schema.Hash(), 1, got.Writes); err != nil {
					t.Fatal(err)
				}
			}
			want, err := reference.Snapshot(schema.Hash())
			if err != nil || !reflect.DeepEqual(snapshot.Entries, want.Entries) {
				t.Fatalf("Go reopen bytes=%+v want=%+v err=%v", snapshot.Entries, want.Entries, err)
			}
			after, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			if (!commit || len(got.Writes) == 0) && !bytes.Equal(before, after) {
				t.Fatal("nonpublication/empty commit changed record")
			}
		}
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) { run(t, trial, 7, false) })
		}
		for _, trial := range trials {
			if trial.name == "empty-commit" || trial.name == "helper-transitive" {
				t.Run(trial.name+"-max", func(t *testing.T) { run(t, trial, math.MaxUint64, false) })
				if trial.name == "helper-transitive" {
					t.Run("actual-rename-refusal", func(t *testing.T) { run(t, trial, 7, true) })
				}
			}
		}
		runtimeTrial := stateGraphAuthoredTrial{"runtime-write-limit", stateGraphAuthoredSource("storage_write", `outer();storage::commit();`, `@effects(storage_write)fn void outer(){storage::write("count",7);storage::write("flag",true);return;}`), ""}
		t.Run("typed-runtime-failure", func(t *testing.T) { run(t, runtimeTrial, 7, false) })
	})
}

func bool64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}
