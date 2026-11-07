//go:build windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/contractstate"
	"golang.org/x/sys/windows"
)

// The actual main, emitted units, runtime assets and provider remain unchanged.
// Each wrapper calls its real OS adapter once and preserves rc/GetLastError.
// This observer only retains the real local event and schedules notification or
// cancellation after the real CAS lease succeeds (open=1, snapshot=2, CAS=3).
func stateCommandBoundarySupport(t *testing.T, support string) string {
	t.Helper()
	local := "void *fairyc_private_windows_cancel_local_v1(void) {"
	lease := "int32_t fairyc_private_windows_root_file_state_lock_v1(size_t file) {"
	for _, definition := range []string{local, lease} {
		if strings.Count(support, definition) != 1 {
			t.Fatalf("require exactly one actual native definition: %s", definition)
		}
	}
	renamedLocal := strings.Replace(local, "fairyc_private_windows_cancel_local_v1", "fairyc_test_command_original_local", 1)
	renamedLease := strings.Replace(lease, "fairyc_private_windows_root_file_state_lock_v1", "fairyc_test_command_original_lease", 1)
	changed := strings.Replace(strings.Replace(support, local, renamedLocal, 1), lease, renamedLease, 1)
	const observer = `
static HANDLE fairyc_test_command_token = NULL;
static volatile LONG fairyc_test_command_leases = 0;
static HANDLE fairyc_test_command_event(const WCHAR *variable) {
    WCHAR name[512];
    DWORD size = GetEnvironmentVariableW(variable, name, 512);
    if (size == 0 || size >= 512) { return NULL; }
    return OpenEventW(EVENT_MODIFY_STATE | SYNCHRONIZE, FALSE, name);
}
static void fairyc_test_command_notify(const WCHAR *variable) {
    HANDLE event = fairyc_test_command_event(variable);
    if (event != NULL) { (void)SetEvent(event); (void)CloseHandle(event); }
}
void *fairyc_private_windows_cancel_local_v1(void) {
    void *result = fairyc_test_command_original_local();
    DWORD saved = GetLastError();
    fairyc_test_command_token = (HANDLE)result;
    SetLastError(saved); return result;
}
int32_t fairyc_private_windows_root_file_state_lock_v1(size_t file) {
    int32_t result = fairyc_test_command_original_lease(file);
    DWORD saved = GetLastError();
    if (result != 0) {
        LONG call = InterlockedIncrement(&fairyc_test_command_leases);
        if (call > 3) { fairyc_test_command_notify(L"FAIRYC_TEST_COMMAND_BOUNDARY_EXTRA"); }
        if (call == 3) {
            DWORD client[1]; WCHAR mode[16];
            DWORD size = GetEnvironmentVariableW(L"FAIRYC_TEST_COMMAND_BOUNDARY_MODE", mode, 16);
            // The command is the only client of its new hidden console. No
            // existing console is attached, broadcast to, or reconfigured.
            if (GetConsoleProcessList(client, 1) == 1 && client[0] == GetCurrentProcessId() &&
                size > 0 && size < 16) {
                BOOL observed = TRUE;
                if (lstrcmpW(mode, L"cancel") == 0) {
                    observed = fairyc_test_command_token != NULL && SetEvent(fairyc_test_command_token) &&
                        WaitForSingleObject(fairyc_test_command_token, 0) == WAIT_OBJECT_0;
                }
                if (observed) { fairyc_test_command_notify(L"FAIRYC_TEST_COMMAND_BOUNDARY_READY"); }
                if (lstrcmpW(mode, L"peer") == 0) {
                    HANDLE release = fairyc_test_command_event(L"FAIRYC_TEST_COMMAND_BOUNDARY_RELEASE");
                    if (release != NULL) { (void)WaitForSingleObject(release, 15000); (void)CloseHandle(release); }
                }
            }
        }
    }
    SetLastError(saved); return result;
}
`
	modified := changed + observer
	recovered := strings.Replace(strings.Replace(strings.TrimSuffix(modified, observer), renamedLocal, local, 1), renamedLease, lease, 1)
	if recovered != support || strings.Count(observer, "fairyc_test_command_original_local();") != 1 || strings.Count(observer, "fairyc_test_command_original_lease(file);") != 1 {
		t.Fatal("observer changed support beyond two once-calling definition wrappers")
	}
	t.Logf("native support original/recovered sha256=%x; observed support sha256=%x; observer sha256=%x", sha256.Sum256([]byte(support)), sha256.Sum256([]byte(modified)), sha256.Sum256([]byte(observer)))
	return modified
}

func TestCandidateStateCommandWindowsBoundary(t *testing.T) {
	entryPath := filepath.Join("..", "..", "tools", "dev", "fairyc-state", "main.fy")
	entry, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatal(err)
	}
	// Detect concurrent input changes instead of silently reporting evidence for
	// a mixture of the parent's old and new product/entry/runtime inputs.
	paths := []string{entryPath, filepath.Join("..", "candidatehost", "windows_process.c"), "candidate_windows_console_test.go"}
	for _, name := range []string{"module_state_command_windows", "module_state_command_snapshot_report", "module_state_graph_windows_persistent", "windows_cancel", "native_runtime_asset_pin"} {
		paths = append(paths, filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy"))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		t.Logf("input %s sha256=%x", path, hash)
		t.Cleanup(func() {
			current, err := os.ReadFile(path)
			if err != nil || sha256.Sum256(current) != hash {
				t.Errorf("PARENT INPUT CHANGED during boundary verification: %s err=%v", path, err)
			}
		})
	}
	root := t.TempDir()
	peer := filepath.Join(root, "fairyc-state-go.exe")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", peer, "./tools/dev/fairyc-state")
	build.Dir = filepath.Join("..", "..")
	build.WaitDelay = time.Second
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("actual Go native command build: %v %s", err, out)
	}
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	t.Logf("actual runtime pack sha256=%x; frame pack sha256=%x", sha256.Sum256(runtimePack), sha256.Sum256(framePack))
	var trial stateGraphAuthoredTrial
	for _, item := range graphTransportTrials() {
		if item.name == "helper-transitive" {
			trial = item
			break
		}
	}
	if trial.name == "" {
		t.Fatal("missing actual helper-transitive source")
	}
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatal(code)
	}
	schema := graphWindowsPersistentLayout(t, f)
	policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
	sourcePath := filepath.Join(root, "ledger.fy")
	if err = os.WriteFile(sourcePath, []byte(trial.text), 0600); err != nil {
		t.Fatal(err)
	}
	seed := func(t *testing.T) string {
		t.Helper()
		directory := t.TempDir()
		store, err := contractstate.CreateLocalPersistence(directory, schema, policy)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		var unknown contractstate.Key
		unknown[0], unknown[31] = 255, 213
		// Seven REAL provider CAS operations establish version 7. No record,
		// provider, metadata or transaction result is substituted in the child.
		for version := uint64(0); version < 7; version++ {
			receipt, err := store.CompareAndPublish(schema.Hash(), version, []contractstate.PersistentWrite{{Key: unknown, Value: []byte{0, 255, 13, 10}}})
			if err != nil || receipt.Version != version+1 {
				t.Fatalf("Go provider seed receipt=%+v err=%v", receipt, err)
			}
		}
		return directory
	}
	reopen := func(t *testing.T, directory string) contractstate.PersistentSnapshot {
		t.Helper()
		store, err := contractstate.OpenLocalPersistence(directory, schema, policy)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		snapshot, err := store.Snapshot(schema.Hash())
		if err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	windowsConsoleBuild(t, string(entry), stateCommandBoundarySupport, func(t *testing.T, image string) {
		profile := "low"
		if strings.Contains(t.Name(), "/optimized") {
			profile = "optimized"
		}
		for name, data := range map[string][]byte{"fairyc-state-graph-runtime.pack": runtimePack, "fairyc-state-graph-frame.pack": framePack} {
			if err := os.WriteFile(filepath.Join(filepath.Dir(image), name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		args := func(action, directory string) []string {
			return []string{action, directory, sourcePath, "--engine", "native-graph", "--profile", profile}
		}
		reference := seed(t)
		wantCode, wantJSON, wantErr := stateCommandProcess(t, peer, root, args("execute", reference)...)
		if wantCode != 0 || len(wantErr) != 0 || !json.Valid(wantJSON) {
			t.Fatalf("actual Go native reference code=%d JSON=%s stderr=%s", wantCode, wantJSON, wantErr)
		}
		wantSnapshot := reopen(t, reference)
		if wantSnapshot.Version != 8 {
			t.Fatal("reference must commit exactly once", wantSnapshot)
		}
		wantBytes, err := os.ReadFile(filepath.Join(reference, "state-v1.bin"))
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"closed-stdout", "cancel", "peer"} {
			t.Run(mode, func(t *testing.T) {
				directory := seed(t)
				newEvent := func(label string) (windows.Handle, string) {
					t.Helper()
					name := fmt.Sprintf("Local\\fairyc-command-boundary-%x", sha256.Sum256([]byte(t.Name()+label+directory)))
					native, err := windows.UTF16PtrFromString(name)
					if err != nil {
						t.Fatal(err)
					}
					event, err := windows.CreateEvent(nil, 1, 0, native)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = windows.CloseHandle(event) })
					return event, name
				}
				ready, readyName := newEvent("ready")
				extra, extraName := newEvent("extra")
				release, releaseName := newEvent("release")
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				defer cancel()
				command := exec.CommandContext(ctx, image, args("execute", directory)...)
				command.Dir, command.WaitDelay = root, 2*time.Second
				command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, HideWindow: true}
				for _, setting := range os.Environ() {
					name, _, _ := strings.Cut(setting, "=")
					if !strings.HasPrefix(strings.ToUpper(name), "FAIRYC_TEST_COMMAND_BOUNDARY_") {
						command.Env = append(command.Env, setting)
					}
				}
				command.Env = append(command.Env, "FAIRYC_TEST_COMMAND_BOUNDARY_MODE="+mode, "FAIRYC_TEST_COMMAND_BOUNDARY_READY="+readyName, "FAIRYC_TEST_COMMAND_BOUNDARY_EXTRA="+extraName, "FAIRYC_TEST_COMMAND_BOUNDARY_RELEASE="+releaseName)
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				var pipeWriter *os.File
				if mode == "closed-stdout" {
					reader, writer, err := os.Pipe()
					if err != nil {
						t.Fatal(err)
					}
					if err = reader.Close(); err != nil {
						writer.Close()
						t.Fatal(err)
					}
					pipeWriter = writer
					t.Cleanup(func() { _ = writer.Close() })
					command.Stdout = writer // Real broken pipe; no test writer/provider.
				}
				if err = command.Start(); err != nil {
					t.Fatal(err)
				}
				var waited bool
				var commandErr error
				waitCommand := func(terminate bool) error {
					if !waited {
						if terminate {
							_ = command.Process.Kill()
						}
						waited = true
						commandErr = command.Wait()
					}
					return commandErr
				}
				// Every started child is reaped exactly once, including Fatal paths.
				// Wait also drains the stdout/stderr copy goroutines before reads.
				t.Cleanup(func() { _ = waitCommand(true) })
				if pipeWriter != nil {
					if err = pipeWriter.Close(); err != nil {
						_ = waitCommand(true)
						t.Fatal(err)
					}
				}
				if mode == "peer" {
					status, waitErr := windows.WaitForSingleObject(ready, 15000)
					if waitErr != nil || status != windows.WAIT_OBJECT_0 {
						_ = windows.SetEvent(release)
						_ = waitCommand(true)
						t.Fatalf("actual CAS lease boundary unavailable: status=%d err=%v stdout=%s stderr=%s", status, waitErr, &stdout, &stderr)
					}
					peerCode, peerOut, peerErr := stateCommandProcess(t, peer, root, args("snapshot", directory)...)
					// Release the actual lease owner before any assertion can exit.
					if err = windows.SetEvent(release); err != nil {
						_ = waitCommand(true)
						t.Fatalf("release actual CAS lease: %v stdout=%s stderr=%s", err, &stdout, &stderr)
					}
					if peerCode != 7 || len(peerOut) != 0 || len(peerErr) == 0 {
						_ = waitCommand(true)
						t.Fatalf("actual Go peer busy code=%d stdout=%s stderr=%s", peerCode, peerOut, peerErr)
					}
					t.Logf("real Go peer snapshot refused with exit=7 while actual CAS lease held: %s", peerErr)
				}
				exitCode := 0
				if err = waitCommand(false); err != nil {
					if exit, ok := err.(*exec.ExitError); ok {
						exitCode = exit.ExitCode()
					} else {
						t.Fatal(err)
					}
				}
				if mode == "closed-stdout" {
					if exitCode != 4 || stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("report output failed")) || !bytes.Contains(stderr.Bytes(), []byte("may already be published")) {
						t.Fatalf("broken stdout code=%d stdout=%s stderr=%s", exitCode, &stdout, &stderr)
					}
					t.Logf("actual closed stdout: exit=4; retained publication warning=%s", &stderr)
				} else if exitCode != 0 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) || !bytes.Equal(stdout.Bytes(), wantJSON) {
					t.Fatalf("known publication JSON code=%d stdout=%s stderr=%s Go=%s", exitCode, &stdout, &stderr, wantJSON)
				}
				if status, err := windows.WaitForSingleObject(ready, 0); err != nil || status != windows.WAIT_OBJECT_0 {
					t.Fatal("third successful lease/cancellation observation missing", status, err)
				}
				if status, err := windows.WaitForSingleObject(extra, 0); err != nil || status != uint32(windows.WAIT_TIMEOUT) {
					t.Fatal("command acquired an unexpected fourth lease/retried", status, err)
				}
				for attempt := 0; attempt < 2; attempt++ {
					got := reopen(t, directory)
					if got.Version != 8 || !reflect.DeepEqual(got, wantSnapshot) {
						t.Fatalf("Go provider reopen commit/one-version mismatch: got=%+v want=%+v", got, wantSnapshot)
					}
				}
				actualBytes, err := os.ReadFile(filepath.Join(directory, "state-v1.bin"))
				if err != nil || !bytes.Equal(actualBytes, wantBytes) {
					t.Fatal("actual durable record differs from independent Go native publication", err)
				}
				t.Logf("actual main mode=%s exit=%d; exactly three successful leases; two independent Go reopens version=8; Go native durable bytes sha256=%x", mode, exitCode, sha256.Sum256(actualBytes))
			})
		}
	})
}
