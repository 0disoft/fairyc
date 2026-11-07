//go:build windows

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
	"golang.org/x/sys/windows"
)

// Retain the existing real Prepared fixture and build-result wire reader. The
// only consumer change is the explicit token; no generated C or host shim is used.
func graphWindowsBuildCancelProbe() string {
	p := strings.Replace(graphWindowsBuildProbe(), "module main;", "module main;import windows_cancel as cancel;import std::process as process;", 1)
	p = strings.Replace(p, "else{0};\n let options=", "else{73};\n let options=", 1)
	p = strings.Replace(p, " let built=switch(build::build(", `
 var u8[512] event_name=[0;512];let event_size=switch(process::arg(1,ref mut event_name)){case .ok(p):{p}case .err(_):{return 12;}};let ref u8[] event_raw=ref event_name;
 let token=if((mode&2048)!=0){switch(cancel::open(event_raw[..event_size])){case .ok(p):{p}case .err(_):{return 13;}}}else{cancel::none()};
 let built=switch(build::build_cancel(`, 1)
	return strings.Replace(p, "options,ref mut build_work)", "options,ref token,ref mut build_work)", 1)
}

func TestCandidateStateGraphWindowsBuildCancellation(t *testing.T) {
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	trial := stateGraphAuthoredTrial{"build-cancel", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("count",42);let count=storage::read("count");storage::write("count",count);storage::commit();`, ""), ""}
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatal("invalid cancellation fixture", code)
	}
	_, _, _, handoff := graphNativeOracle(t, f, true)
	want, err := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	request, err := want.Input(contractstate.PersistentSnapshot{LayoutHash: graphNativeLayout(t, handoff)}, graphNativeLimits())
	if err != nil {
		t.Fatal(err)
	}
	source, err := want.Source()
	if err != nil {
		t.Fatal(err)
	}
	support, err := cstdlib.ContractStateGraphRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	probe := graphWindowsBuildCancelProbe()
	module, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "module_state_graph_windows_build.fy"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("module_state_graph_windows_build.fy sha256=%x; cancellation probe sha256=%x", sha256.Sum256(module), sha256.Sum256([]byte(probe)))
	candidateAtlasProbeSize(t, "module_state_graph_windows_build", probe, 512<<10, func(t *testing.T, image string) {
		profile := cdriver.LowProfile
		mode := uint64(0)
		if strings.Contains(t.Name(), "/optimized") {
			profile, mode = cdriver.OptimizedProfile, 1
		}
		name := windowsCancelName(t, "build")
		event, err := windows.CreateEvent(nil, 1, 1, windowsCancelWide(t, name))
		if err != nil {
			t.Fatal(err)
		}
		defer windows.CloseHandle(event)
		run := func(bits uint64, refuseIO bool) graphWindowsBuildResult {
			t.Helper()
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, image, name)
			cmd.WaitDelay = 2 * time.Second
			cmd.Stdin = bytes.NewReader(graphTransportPacket(f, trial.text, runtimePack, framePack, mode|bits, 0, [2]uint64{}))
			temp, cl := root, ""
			if refuseIO {
				temp = filepath.Join(root, "absent", "temp")
				cl = "/FI\"" + filepath.Join(root, "missing-header.h") + "\""
			}
			changes := map[string]string{"TMP": temp, "TEMP": temp, "CL": cl, "_CL_": ""}
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				if _, replace := changes[strings.ToUpper(key)]; !replace {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			for key, value := range changes {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("build cancellation: %v stderr=%s output=%q", err, &stderr, stdout.Bytes()[:min(stdout.Len(), 512)])
			}
			if leftovers, err := filepath.Glob(filepath.Join(root, "fairyc-native-*")); err != nil || len(leftovers) != 0 {
				t.Fatal("scratch escaped build cancellation", leftovers, err)
			}
			if _, err := os.Stat(filepath.Join(root, "absent")); !os.IsNotExist(err) {
				t.Fatal("pre-canceled build created temporary root", err)
			}
			windowsCancelPoll(t, event, windows.WAIT_OBJECT_0)
			return graphWindowsBuildRead(t, stdout.Bytes())
		}
		for _, trial := range []struct {
			name                      string
			bits, stage, status, work uint64
		}{
			{"pre-canceled", 0, 11, 1009, 73},
			{"invalid-timeout", 32, 1, 0, 73},
			{"invalid-stack", 64, 1, 0, 73},
			{"overlarge-timeout", 1024, 1, 0, 73},
			{"exhausted-work", 128, 1, 1002, 16777216},
			{"overdrawn-work", 256, 1, 1002, 16777217},
		} {
			t.Run(trial.name, func(t *testing.T) {
				got := run(2048|trial.bits, true)
				if got.stage != trial.stage || got.status != trial.status || got.osCode != 0 || got.unit != 0 || got.work != trial.work || len(got.blobs[0]) != 0 {
					t.Fatalf("cancellation/validation precedence: stage/status/os/unit/work=%d/%d/%d/%d/%d diagnostics=%q", got.stage, got.status, got.osCode, got.unit, got.work, got.blobs[0])
				}
			})
		}
		t.Run("none", func(t *testing.T) {
			got := run(0, false)
			if got.stage != 0 || got.status != 0 || got.osCode != 0 || got.unit != 2 || got.work != uint64(73+len(source)+len(support)+2) {
				t.Fatalf("none-token build: %+v", got)
			}
			hash := want.ProgramHash()
			if !bytes.Equal(got.blobs[1], hash[:]) || !bytes.Equal(got.blobs[2], request.Bytes()) {
				t.Fatal("none-token Built lost Prepared identity/request")
			}
			if _, err := pecoff.InspectExecutable(got.blobs[0]); err != nil {
				t.Fatal("none-token image", err)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(cdriver.WindowsHostArtifact{Bytes: got.blobs[0], Profile: profile}, bytes.NewReader(request.Bytes()))
			if err != nil || out.ExitCode != 0 || out.Stderr != "" {
				t.Fatal("none-token graph execution", out, err)
			}
			admitted, err := want.Decode(request, []byte(out.Stdout))
			if err != nil {
				t.Fatal("none-token response admission", err)
			}
			graphNativeSemanticExpect(t, f.input, admitted)
			t.Logf("none-token image sha256=%x work=%d", sha256.Sum256(got.blobs[0]), got.work)
		})
	}, "module_state_graph_transport_request")
}
