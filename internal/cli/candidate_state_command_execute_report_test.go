//go:build windows

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/target"
)

func commandExecuteReportProbe(t *testing.T) string {
	t.Helper()
	prefix, _, ok := strings.Cut(graphWindowsPersistentProbe(), " // All admission failures precede store.snapshot")
	if !ok {
		t.Fatal("persistent setup changed")
	}
	prefix = strings.Replace(prefix, "module main;", "module main;import module_state_command_snapshot_report as report;import module_state_command_options as command;", 1)
	prefix = strings.Replace(prefix, "fn i32 main(){", "fn i32 main(){var output=bytes::empty();var i32 exit=0;{", 1)
	return prefix + `
 var work=persistent::empty_work();let outcome=switch(persistent::run(ref built,ref instance,limits,options,ref mut work)){case .ok(p):{p}case .err(e):{let cause=persistent::cause(ref e);number(cause.stage as usize);number(cause.code as usize);return 31;}};
 assert(work.calls.snapshots==1 && work.calls.runs==1);
 var selected=command::Options{.action=2,.release,.store_index=1,.source_index=2};var usize meter=17;
 output=switch(report::execute(selected,builder::prepared(ref built),ref outcome,ref mut meter)){case .ok(p):{p}case .err(code):{number(code as usize);return 32;}};
 exit=report::outcome_exit(ref outcome);let cost=meter-17;
 var usize exact=report::work_limit()-cost;let again=switch(report::execute(selected,builder::prepared(ref built),ref outcome,ref mut exact)){case .ok(p):{p}case .err(_):{return 33;}};
 assert(exact==report::work_limit() && bytes::equal(bytes::view(ref output),bytes::view(ref again)));
 var usize short=report::work_limit()-cost+1;switch(report::execute(selected,builder::prepared(ref built),ref outcome,ref mut short)){case .ok(_):{return 34;}case .err(code):{assert(code==1002 && short<=report::work_limit());}};
 var usize invalid=7;selected.action=3;switch(report::execute(selected,builder::prepared(ref built),ref outcome,ref mut invalid)){case .ok(_):{return 35;}case .err(code):{assert(code==1001 && invalid==7);}};selected.action=2;
 // Compile a genuinely different body under the SAME schema and coordinate.
 // No arbitrary report, receipt, or Outcome constructor participates.
 let different=switch(current(b"module ledger;@contract_uid(\"fixture.authored-graph\")state{cell bool flag;cell u32 count;cell u64 total;cell bytes20 account;cell bytes32 hash;cell bytes blob;cell text title;map u32 u32 values;map text bytes labels;}@effects(storage_write)fn void transaction(){storage::write(\"count\",12345);storage::reject();}",ref table,owner,false)){case .ok(p):{p}case .err(_):{return 36;}};
 var other_work=transport::empty_work();let other=switch(transport::prepare(different,ref runtime,ref framing,b"x86_64-windows-msvc",ref mut other_work)){case .ok(p):{p}case .err(_):{return 37;}};
 assert(!persistent::matches(ref outcome,ref other));switch(report::execute(selected,ref other,ref outcome,ref mut invalid)){case .ok(_):{return 38;}case .err(code):{assert(code==1001 && invalid==7);}};
 incoming=bytes::empty();
 }assert(io::write_bytes(bytes::view(ref output)).is_ok);return exit;
}`
}

func TestCandidateStateCommandExecuteReport(t *testing.T) {
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	var trials []stateGraphAuthoredTrial
	for _, trial := range graphTransportTrials() {
		switch trial.name {
		case "helper-transitive", "empty-commit", "empty-reject", "trap", "present-borrow-copy":
			trials = append(trials, trial)
		}
	}
	all := `storage::write("flag",true);storage::write("count",4294967295);storage::write("total",18446744073709551615);storage::write("account",b"abcdefghijklmnopqrst");storage::write("hash",b"0123456789abcdef0123456789abcdef");storage::write("blob",b"\x00\xff\r\n");storage::write("title","한글\n\"");storage::read("flag");storage::read("count");storage::read("total");storage::read("account");storage::read("hash");storage::read("blob");storage::read("title");storage::read("values",42);storage::write("blob",b"");let borrowed=storage::borrow("blob");storage::read("blob");storage::commit();`
	trials = append(trials, stateGraphAuthoredTrial{"all-read-codecs", stateGraphAuthoredSource("storage_read,storage_write", all, ""), ""}, stateGraphAuthoredTrial{"read-reject", stateGraphAuthoredSource("storage_read", `storage::read("blob");storage::reject();`, ""), ""})
	candidateAtlasProbeSize(t, "module_state_command_snapshot_report", commandExecuteReportProbe(t), 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
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
				version := uint64(7)
				if trial.name == "empty-commit" {
					version = math.MaxUint64
				}
				var unknown contractstate.Key
				unknown[0], unknown[31] = 255, 213
				seed := []contractstate.Entry{{Key: unknown, Value: []byte{0, 255, 13, 10}}}
				create := func() (string, *contractstate.LocalPersistence) {
					directory := t.TempDir()
					policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
					instance, err := contractstate.CreateLocalPersistence(directory, schema, policy)
					if err != nil {
						t.Fatal(err)
					}
					instance.Close()
					if err = os.WriteFile(filepath.Join(directory, "state-v1.bin"), stateLocalCodecRecord(t, schema, version, seed), 0600); err != nil {
						t.Fatal(err)
					}
					instance, err = contractstate.OpenLocalPersistence(directory, schema, policy)
					if err != nil {
						t.Fatal(err)
					}
					return directory, instance
				}
				directory, actual := create()
				actual.Close()
				_, oracle := create()
				defer oracle.Close()
				lowered, err := contractstatemir.LowerHIRTransactions(f.input)
				if err != nil || len(lowered) != 1 {
					t.Fatal(err)
				}
				var expectedOutcome contractstatemir.Outcome
				var category string
				func() {
					defer func() {
						if r := recover(); r != nil {
							trap, ok := r.(contractstatemir.Trap)
							if !ok {
								panic(r)
							}
							category = trap.Category
						}
					}()
					expectedOutcome, err = contractstatemir.ExecutePersistent(oracle, lowered[0].Program, contractstate.Limits{MaxWrites: 64, MaxWriteBytes: 65536, MaxValueBytes: 4096, MaxMapKeyBytes: 4096})
				}()
				if err != nil {
					t.Fatal(err)
				}
				// graph_decode.go clones wire views: even zero-byte observations are a
				// non-nil []byte. Keep the native report's empty-string JSON semantics.
				for i := range expectedOutcome.Reads {
					if expectedOutcome.Reads[i].Value.Encoded == nil {
						expectedOutcome.Reads[i].Value.Encoded = []byte{}
					}
				}
				var outcome *contractstatemir.Outcome
				wantExit := 0
				if category != "" {
					wantExit = 8
				} else {
					outcome = &expectedOutcome
					if expectedOutcome.Exit == contractstatemir.ExitReject {
						wantExit = 6
					}
				}
				release := strings.Contains(t.Name(), "/optimized/")
				profile := "low"
				if release {
					profile = "optimized"
				}
				expected := struct {
					Schema       string                    `json:"schema"`
					Action       string                    `json:"action"`
					LayoutHash   string                    `json:"layoutHash"`
					Outcome      *contractstatemir.Outcome `json:"outcome,omitempty"`
					TrapCategory string                    `json:"trapCategory,omitempty"`
					Engine       string                    `json:"engine"`
					Profile      string                    `json:"profile"`
					BodyHash     string                    `json:"bodyHash"`
					ProgramHash  string                    `json:"programHash"`
				}{"fairyc.local-state-run/v2", "execute", schema.Hash().String(), outcome, category, "native-graph", profile, prepared.BodyHash().String(), fmt.Sprintf("%x", prepared.ProgramHash())}
				want, err := json.Marshal(expected)
				if err != nil {
					t.Fatal(err)
				}
				want = append(want, '\n')
				exit, out := candidateAtlasProcess(t, image, graphWindowsRunPacket(f, trial.text, runtimePack, framePack, 2, 511, release), directory)
				if exit != wantExit || !bytes.Equal([]byte(out), want) {
					t.Fatalf("native exit=%d want=%d\ngot %s\nwant %s", exit, wantExit, out, want)
				}
				reopened, err := contractstate.OpenLocalPersistence(directory, schema, contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536})
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.Close()
				gotState, err := reopened.Snapshot(schema.Hash())
				if err != nil {
					t.Fatal(err)
				}
				wantState, err := oracle.Snapshot(schema.Hash())
				if err != nil {
					t.Fatal(err)
				}
				gotJSON, _ := json.Marshal(gotState)
				wantJSON, _ := json.Marshal(wantState)
				if !bytes.Equal(gotJSON, wantJSON) {
					t.Fatal("real durable state differs from independent Go execution")
				}
				t.Logf("real Outcome exit=%d owned report=%d bytes; budget exact/one-short and different same-layout Prepared refused", exit, len(out))
			})
		}
	}, "module_state_graph_windows_persistent")
}
