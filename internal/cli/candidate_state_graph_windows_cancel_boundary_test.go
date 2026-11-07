//go:build windows

package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"golang.org/x/sys/windows"
)

const graphCancelBoundaryEventEnv = "FAIRYC_TEST_GRAPH_CANCEL_BOUNDARY_EVENT"
const graphCancelBoundaryCallEnv = "FAIRYC_TEST_GRAPH_CANCEL_BOUNDARY_CALL"

func graphWindowsCancelBoundaryProbe(t *testing.T) string {
	t.Helper()
	prefix, _, ok := strings.Cut(graphWindowsPersistentProbe(), " // All admission failures precede store.snapshot")
	if !ok {
		t.Fatal("persistent probe setup boundary changed")
	}
	prefix = strings.Replace(prefix, "module main;", "module main;import windows_cancel as cancel;", 1)
	return prefix + `
 var u8[512] event_name=[0;512];let event_size=switch(process::arg(2,ref mut event_name)){case .ok(p):{p}case .err(_):{return 30;}};let ref u8[] event_raw=ref event_name;
 let token=switch(cancel::open(event_raw[..event_size])){case .ok(p):{p}case .err(_):{return 31;}};
 assert(!switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{return 32;}});
 let usize expected=if((mode&512)!=0){0}else if((mode&1024)!=0){1}else{2};
 var work=persistent::empty_work();work.local=opening;
 switch(persistent::run_cancel(ref built,ref instance,limits,options,ref token,ref mut work)){
 case .ok(outcome):{
  assert(expected==2 && work.calls.snapshots==1 && work.calls.runs==1 && work.calls.publishes==1);
  // The wrapper signals during real CAS lease acquisition, before this API
  // returns. The acknowledged publication cannot be revoked by that token.
  assert(switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{false}});
  let publication=persistent::publication(ref outcome);assert(publication.acknowledged && publication.consistent);
  number(0);number(work.calls.snapshots);number(work.calls.runs);number(work.calls.publishes);number(1);
  number(publication.receipt.version);number(publication.receipt.writes);var usize meter=0;
  let receipt_hash=switch(identities::words(publication.receipt.layout,ref mut meter)){case .ok(p):{p}case .err(_):{return 33;}};blob(bytes::view(ref receipt_hash));
  incoming=bytes::empty();result(persistent::admitted(ref outcome),ref work.execution);
 }
 case .err(error):{
  let cause=persistent::cause(ref error);assert(expected<2 && cause.stage==9 && cause.code==1009);
  assert(cause.cancellation_kind==0 && cause.cancellation_os_code==0 && cause.runner_stage==0 && cause.runner_status==0 && cause.runner_os_code==0 && cause.runtime_status==0 && cause.storage.stage==0);
  assert(work.calls.snapshots==1 && work.calls.runs==expected && work.calls.publishes==0);
  assert(switch(cancel::requested(ref token)){case .ok(p):{p}case .err(_):{false}});
  switch(persistent::failure_runner(ref error)){case .ok(_):{return 34;}case .err(code):{assert(code==0);}};
  number(1);number(work.calls.snapshots);number(work.calls.runs);number(work.calls.publishes);number(1);number(cause.stage as usize);number(cause.code as usize);
  if(expected==0){
   assert(same_work(work.execution,runner::empty_work()));
   switch(persistent::failure_admitted(ref error)){case .ok(_):{return 35;}case .err(code):{assert(code==0);}};number(0);
  }else{
   // Drop all incoming graph/asset bytes before borrowing the retained owner.
   incoming=bytes::empty();let admitted=switch(persistent::failure_admitted(ref error)){case .ok(p):{p}case .err(_):{return 36;}};
   number(1);result(admitted,ref work.execution);
  }
 }};return 0;
}`
}

// Scheduling-only observer in a COPY of native support. Each wrapper calls the
// renamed real function exactly once and returns its exact result/last error.
// It never supplies storage data, metadata, a graph response, or a fake provider.
const graphCancelBoundaryNotify = `
static void fairyc_test_graph_cancel_notify(LONG successful_call) {
    WCHAR selected[8], name[512];
    DWORD selected_size = GetEnvironmentVariableW(L"FAIRYC_TEST_GRAPH_CANCEL_BOUNDARY_CALL", selected, 8);
    if (selected_size != 1 || selected[0] < L'1' || selected[0] > L'9' ||
        successful_call != (LONG)(selected[0] - L'0')) { return; }
    DWORD name_size = GetEnvironmentVariableW(L"FAIRYC_TEST_GRAPH_CANCEL_BOUNDARY_EVENT", name, 512);
    if (name_size == 0 || name_size >= 512) { return; }
    HANDLE event = OpenEventW(EVENT_MODIFY_STATE, FALSE, name);
    if (event != NULL) { (void)SetEvent(event); (void)CloseHandle(event); }
}
`

func graphCancelBoundarySupport(t *testing.T, support, boundary string) string {
	t.Helper()
	var definition, symbol, wrapper string
	switch boundary {
	case "cleanup":
		symbol = "fy_std_windows_dir_remove_all_v0"
		definition = "struct fy_std_windows_file_result_v0 fy_std_windows_dir_remove_all_v0(\n    const uint8_t *path\n) {"
		wrapper = `
struct fy_std_windows_file_result_v0 fy_std_windows_dir_remove_all_v0(const uint8_t *path) {
    static volatile LONG successful_calls = 0;
    struct fy_std_windows_file_result_v0 result = fairyc_test_original_fy_std_windows_dir_remove_all_v0(path);
    DWORD original_error = GetLastError();
    if (result.status == FY_STD_WINDOWS_FILE_OK_V0) {
        fairyc_test_graph_cancel_notify(InterlockedIncrement(&successful_calls));
    }
    SetLastError(original_error);
    return result;
}
`
	case "lease":
		symbol = "fairyc_private_windows_root_file_state_lock_v1"
		definition = "int32_t fairyc_private_windows_root_file_state_lock_v1(size_t file) {"
		wrapper = `
int32_t fairyc_private_windows_root_file_state_lock_v1(size_t file) {
    static volatile LONG successful_calls = 0;
    int32_t result = fairyc_test_original_fairyc_private_windows_root_file_state_lock_v1(file);
    DWORD original_error = GetLastError();
    if (result != 0) {
        fairyc_test_graph_cancel_notify(InterlockedIncrement(&successful_calls));
    }
    SetLastError(original_error);
    return result;
}
`
	default:
		t.Fatalf("unknown OS boundary %q", boundary)
	}
	if n := strings.Count(support, definition); n != 1 {
		t.Fatalf("expected exactly one real %s definition, found %d", symbol, n)
	}
	renamedDefinition := strings.Replace(definition, symbol, "fairyc_test_original_"+symbol, 1)
	renamed := strings.Replace(support, definition, renamedDefinition, 1)
	observer := graphCancelBoundaryNotify + wrapper
	modified := renamed + observer
	// Removing only the appended observer and reversing only the definition
	// rename must reproduce the original support byte-for-byte.
	recovered := strings.Replace(strings.TrimSuffix(modified, observer), renamedDefinition, definition, 1)
	if recovered != support || strings.Count(wrapper, "fairyc_test_original_"+symbol+"(") != 1 {
		t.Fatal("instrumentation changed more than one definition plus its observer")
	}
	t.Logf("boundary=%s support original sha256=%x; observed sha256=%x; recovered sha256=%x; observer sha256=%x", boundary, sha256.Sum256([]byte(support)), sha256.Sum256([]byte(modified)), sha256.Sum256([]byte(recovered)), sha256.Sum256([]byte(observer)))
	return modified
}

func graphCancelBoundaryFilesHash(t *testing.T, root string) [32]byte {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".fy") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%d:%s:%d:", len(relative), relative, len(data))
		h.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func graphCancelBoundaryBuild(t *testing.T, probe string, verify func(*testing.T, string, string)) {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_state_graph_windows_persistent")
	writeTestSource(t, root, "src/main.fy", probe)
	inputs := graphCancelBoundaryFilesHash(t, filepath.Join(root, "src"))
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 512<<10)
	program := build.Header + strings.Join(build.Sources, "\x00")
	programHash := sha256.Sum256([]byte(program))
	t.Logf("unchanged FY import closure sha256=%x; unchanged emitted header/units sha256=%x", inputs, programHash)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, boundary := range []string{"lease", "cleanup"} {
				t.Run(boundary, func(t *testing.T) {
					observed := graphCancelBoundarySupport(t, support, boundary)
					image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, observed, time.Minute, 16<<20)
					if err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(root, profile.Name+"-"+boundary+".exe")
					if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
						t.Fatal(err)
					}
					t.Logf("native image sha256=%x", sha256.Sum256(image.Bytes))
					verify(t, path, boundary)
				})
			}
		})
	}
	if graphCancelBoundaryFilesHash(t, filepath.Join(root, "src")) != inputs || sha256.Sum256([]byte(build.Header+strings.Join(build.Sources, "\x00"))) != programHash {
		t.Fatal("observer changed FY inputs or emitted program units")
	}
}

func TestCandidateStateGraphWindowsCancellationBoundary(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("native Windows graph boundary probe requires amd64 MSVC")
	}
	probe := graphWindowsCancelBoundaryProbe(t)
	t.Logf("boundary probe sha256=%x", sha256.Sum256([]byte(probe)))
	for _, name := range []string{"windows_cancel", "module_state_graph_windows_build", "module_state_graph_windows_run", "module_state_graph_windows_persistent", "module_state_local_windows"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
		if err != nil {
			t.Fatal(err)
		}
		before := sha256.Sum256(data)
		t.Logf("%s.fy sha256=%x", name, before)
		t.Cleanup(func() {
			after, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
			if err != nil || sha256.Sum256(after) != before {
				t.Errorf("product source changed during boundary verification: %s err=%v", name, err)
			}
		})
	}
	var trial stateGraphAuthoredTrial
	for _, candidate := range graphTransportTrials() {
		if candidate.name == "helper-transitive" {
			trial = candidate
			break
		}
	}
	if trial.name == "" {
		t.Fatal("missing helper-transitive graph")
	}
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatal(code)
	}
	schema := graphWindowsPersistentLayout(t, f)
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	runtimeHash, frameHash := sha256.Sum256(runtimePack), sha256.Sum256(framePack)
	t.Logf("unchanged graph runtime pack sha256=%x; frame pack sha256=%x", runtimeHash, frameHash)
	graphCancelBoundaryBuild(t, probe, func(t *testing.T, image, boundary string) {
		cases := []struct {
			name string
			call string
			mode uint64
			runs uint64
			cas  bool
		}{{"after-snapshot", "2", 512, 0, false}, {"after-CAS-entry", "3", 2048, 1, true}}
		if boundary == "cleanup" {
			cases = cases[:0]
			cases = append(cases, struct {
				name string
				call string
				mode uint64
				runs uint64
				cas  bool
			}{"after-run-cleanup", "2", 1024, 1, false})
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				directory := t.TempDir()
				policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
				store, err := contractstate.CreateLocalPersistence(directory, schema, policy)
				if err != nil {
					t.Fatal(err)
				}
				store.Close()
				var unknown contractstate.Key
				unknown[0], unknown[31] = 255, 213
				seed := []contractstate.Entry{{Key: unknown, Value: []byte{0, 255, 13, 10}}}
				before := stateLocalCodecRecord(t, schema, 7, seed)
				statePath := filepath.Join(directory, "state-v1.bin")
				if err = os.WriteFile(statePath, before, 0600); err != nil {
					t.Fatal(err)
				}
				name := fmt.Sprintf("Local\\fairyc-cancel-%x", sha256.Sum256([]byte(directory)))
				native, err := windows.UTF16PtrFromString(name)
				if err != nil {
					t.Fatal(err)
				}
				event, err := windows.CreateEvent(nil, 1, 0, native)
				if err != nil {
					t.Fatal(err)
				}
				defer windows.CloseHandle(event)
				if status, err := windows.WaitForSingleObject(event, 0); err != nil || status != uint32(windows.WAIT_TIMEOUT) {
					t.Fatal("Go-owned event must start clear", status, err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, image, directory, name)
				command.WaitDelay = 2 * time.Second
				for _, setting := range os.Environ() {
					key, _, _ := strings.Cut(setting, "=")
					if !strings.EqualFold(key, graphCancelBoundaryEventEnv) && !strings.EqualFold(key, graphCancelBoundaryCallEnv) {
						command.Env = append(command.Env, setting)
					}
				}
				command.Env = append(command.Env, graphCancelBoundaryEventEnv+"="+name, graphCancelBoundaryCallEnv+"="+test.call)
				command.Stdin = bytes.NewReader(graphWindowsRunPacket(f, trial.text, runtimePack, framePack, 2|test.mode, 511, strings.Contains(t.Name(), "/optimized/")))
				var output, stderr bytes.Buffer
				command.Stdout, command.Stderr = &output, &stderr
				if err = command.Run(); err != nil || stderr.Len() != 0 {
					t.Fatalf("boundary process err=%v stdout=%q stderr=%q", err, &output, &stderr)
				}
				if status, err := windows.WaitForSingleObject(event, 0); err != nil || status != windows.WAIT_OBJECT_0 {
					t.Fatal("OS boundary observer did not signal", status, err)
				}
				r := graphResponseReader{t, bufio.NewReader(strings.NewReader(output.String()))}
				failure := r.number()
				if failure != bool64(!test.cas) || r.number() != 1 || r.number() != test.runs || r.number() != bool64(test.cas) || r.number() != 1 {
					t.Fatalf("boundary outcome/call counts/token observation=%q", &output)
				}
				var writes []contractstate.PersistentWrite
				if test.cas {
					version, count := r.number(), r.number()
					hash := schema.Hash()
					if version != 8 || count == 0 || !bytes.Equal(r.blob(), hash[:]) {
						t.Fatal("CAS acknowledged/consistent receipt mismatch", &output)
					}
					remaining, err := io.ReadAll(r.r)
					if err != nil {
						t.Fatal(err)
					}
					admitted, _, got := graphResponseRead(t, string(remaining))
					if admitted != 0 || got.Version != 7 || uint64(len(got.Writes)) != count {
						t.Fatal("CAS admitted result/receipt mismatch", got)
					}
					got.Version = 0
					graphNativeSemanticExpect(t, f.input, got)
					writes = got.Writes
				} else {
					if r.number() != 9 || r.number() != 1009 || r.number() != test.runs {
						t.Fatal("cancellation cause/retained owner mismatch", &output)
					}
					remaining, err := io.ReadAll(r.r)
					if err != nil {
						t.Fatal(err)
					}
					if test.runs == 0 {
						if len(remaining) != 0 {
							t.Fatalf("snapshot-only refusal had a response: %q", remaining)
						}
					} else {
						admitted, _, got := graphResponseRead(t, string(remaining))
						if admitted != 0 || got.Version != 7 || len(got.Writes) == 0 {
							t.Fatal("post-run cancellation lost actual admitted owner", got)
						}
						got.Version = 0
						graphNativeSemanticExpect(t, f.input, got)
					}
					after, err := os.ReadFile(statePath)
					if err != nil || !bytes.Equal(before, after) {
						t.Fatal("cancellation changed actual durable bytes", err)
					}
				}
				wantVersion, wantEntries := uint64(7), seed
				if test.cas {
					reference, err := contractstate.NewReferencePersistence(schema, nil)
					if err != nil {
						t.Fatal(err)
					}
					if _, err = reference.CompareAndPublish(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: unknown, Value: seed[0].Value}}); err != nil {
						t.Fatal(err)
					}
					if _, err = reference.CompareAndPublish(schema.Hash(), 1, writes); err != nil {
						t.Fatal(err)
					}
					want, err := reference.Snapshot(schema.Hash())
					if err != nil {
						t.Fatal(err)
					}
					wantVersion, wantEntries = 8, want.Entries
				}
				reopened, err := contractstate.OpenLocalPersistence(directory, schema, policy)
				if err != nil {
					t.Fatal(err)
				}
				defer reopened.Close()
				snapshot, err := reopened.Snapshot(schema.Hash())
				if err != nil || snapshot.Version != wantVersion || !reflect.DeepEqual(snapshot.Entries, wantEntries) {
					t.Fatalf("independent Go reopen=%+v want version=%d entries=%+v err=%v", snapshot, wantVersion, wantEntries, err)
				}
				t.Logf("OS observer=%s successful call=%s; calls=1/%d/%d; token observed before API return; retained admitted=%v; Go reopen version=%d entries=%d", boundary, test.call, test.runs, bool64(test.cas), test.runs != 0, snapshot.Version, len(snapshot.Entries))
			})
		}
	})
	if sha256.Sum256(runtimePack) != runtimeHash || sha256.Sum256(framePack) != frameHash {
		t.Fatal("observer changed graph runtime/frame assets")
	}
}
