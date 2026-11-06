package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Reuse the real authored source -> Complete -> pinned runtime/frame -> Prepared
// fixture reader. Only the final consumer changes: Fairy owns compilation here.
func graphWindowsBuildProbe() string {
	p := strings.Replace(graphTransportProbe, "import module_state_graph_transport as transport;", "import module_state_graph_windows_build as build;import native_msvc_build as msvc;import windows_msvc_toolchain as tools;import module_state_graph_transport as transport;", 1)
	p = strings.Replace(p, "@effects(console)fn i32 main()", "@effects(filesystem,console,environment,clock)fn i32 main()", 1)
	at := strings.Index(p, " // Prepared owns the consumed Complete")
	return p[:at] + `
 let toolchain=switch(tools::resolve()){case .ok(p):{p}case .err(e):{number(99);number(e.stage as usize);number(e.status as usize);number(e.os_code as usize);number(0);blob(b"");return 0;}};
 var usize build_work=if((mode&128)!=0){16777216}else if((mode&256)!=0){16777217}else if((mode&512)!=0){16777216-transport::source(ref prepared).len-1}else{0};
 let options=msvc::Options{.release=(mode&1)!=0,.timeout_ms=if((mode&32)!=0){0}else if((mode&1024)!=0){600001}else{60000},.stack_reserve=if((mode&64)!=0){1}else{65536}};
 let built=switch(build::build(ref toolchain,prepared,ref runtime,options,ref mut build_work)){case .ok(p):{p}case .err(e):{
  number(build::stage(ref e) as usize);number(build::status(ref e) as usize);number(build::os_code(ref e) as usize);number(build::unit(ref e));number(build_work);blob(build::diagnostics(ref e));return 0;}};
 // Source input is destroyed after ownership transfers into Built.
 {let changed=bytes::view_mut(ref mut incoming);changed[0]=255;}incoming=bytes::empty();
 number(0);number(0);number(0);number(build::object_count(ref built));number(build_work);
 blob(build::data(ref built));blob(transport::program_hash(build::prepared(ref built)));blob(input::content(ref packet));return 0;
}`
}

type graphWindowsBuildResult struct {
	stage, status, osCode, unit, work uint64
	blobs                             [][]byte
}

func graphWindowsBuildRead(t *testing.T, raw []byte) graphWindowsBuildResult {
	t.Helper()
	r := bufio.NewReader(bytes.NewReader(raw))
	number := func() uint64 {
		t.Helper()
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("build number: %v %q", err, raw[:min(len(raw), 256)])
		}
		n, err := strconv.ParseUint(strings.TrimSuffix(line, "\n"), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	out := graphWindowsBuildResult{stage: number(), status: number(), osCode: number(), unit: number(), work: number()}
	count := 1
	if out.stage == 0 {
		count = 3
	}
	for range count {
		n := number()
		if n > 64<<20 {
			t.Fatal("unbounded build output", n)
		}
		blob := make([]byte, n)
		if _, err := io.ReadFull(r, blob); err != nil {
			t.Fatal(err)
		}
		if b, err := r.ReadByte(); b != '\n' || err != nil {
			t.Fatal("build blob delimiter", b, err)
		}
		out.blobs = append(out.blobs, blob)
	}
	if extra, _ := io.ReadAll(r); len(extra) != 0 {
		t.Fatalf("trailing build output %q", extra[:min(len(extra), 256)])
	}
	return out
}

func TestCandidateStateGraphWindowsBuild(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows Fairy MSVC graph owner; Linux verification stays batched")
	}
	root := t.TempDir()
	runtimePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "runtime.pack"))
	framePack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-frame.py", filepath.Join(root, "frame.pack"))
	trial := stateGraphAuthoredTrial{"build-owner", stateGraphAuthoredSource("storage_read,storage_write", `storage::write("count",42);let count=storage::read("count");storage::write("count",count);storage::commit();`, ""), ""}
	f, code := stateGraphAuthoredOracle(t, trial)
	if code != "" {
		t.Fatal("invalid build fixture", code)
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
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	candidateAtlasProbeSize(t, "module_state_graph_windows_build", graphWindowsBuildProbe(), 512<<10, func(t *testing.T, image string) {
		mode := uint64(0)
		profile := cdriver.LowProfile
		if strings.Contains(t.Name(), "/optimized") {
			mode, profile = 1, cdriver.OptimizedProfile
		}
		run := func(mode uint64, changes map[string]string) graphWindowsBuildResult {
			t.Helper()
			// Three compiler commands each keep their own original 60s deadline.
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, image)
			cmd.Stdin = bytes.NewReader(graphTransportPacket(f, trial.text, runtimePack, framePack, mode, 0, [2]uint64{}))
			env := map[string]string{"CL": "", "_CL_": "", "TMP": root, "TEMP": root}
			for key, value := range changes {
				env[key] = value
			}
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				if _, replace := env[strings.ToUpper(key)]; !replace {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			for key, value := range env {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil || stderr.Len() != 0 {
				t.Fatalf("Fairy build owner: %v stderr=%s output=%q", err, &stderr, stdout.Bytes()[:min(stdout.Len(), 512)])
			}
			// Both successful and failed commands must relinquish private scratch.
			leftovers, err := filepath.Glob(filepath.Join(root, "fairyc-native-*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatal("scratch escaped build", leftovers, err)
			}
			return graphWindowsBuildRead(t, stdout.Bytes())
		}
		got := run(mode, nil)
		if got.stage != 0 || got.unit != 2 || got.work != uint64(len(source)+len(support)+2) {
			t.Fatalf("build stage/status/os/unit/work=%d/%d/%d/%d/%d diagnostics=%q", got.stage, got.status, got.osCode, got.unit, got.work, got.blobs[0][:min(len(got.blobs[0]), 512)])
		}
		hash := want.ProgramHash()
		if !bytes.Equal(got.blobs[1], hash[:]) || !bytes.Equal(got.blobs[2], request.Bytes()) {
			t.Fatal("Built lost its consumed Prepared identity/request")
		}
		if _, err := pecoff.InspectExecutable(got.blobs[0]); err != nil {
			t.Fatal("Fairy returned an invalid PE", err)
		}
		actual := cdriver.WindowsHostArtifact{Bytes: got.blobs[0], Profile: profile}
		out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(actual, bytes.NewReader(request.Bytes()))
		if err != nil || out.ExitCode != 0 || out.Stderr != "" {
			t.Fatal("Fairy-built graph execution", out, err)
		}
		result, err := want.Decode(request, []byte(out.Stdout))
		if err != nil {
			t.Fatal("Fairy-built response admission", err)
		}
		graphNativeSemanticExpect(t, f.input, result)
		oracle, err := cdriver.BuildWindowsMSVCTarget(tc, profile, source, support)
		if err != nil {
			t.Fatal(err)
		}
		independent, err := cdriver.ExecuteWindowsMSVCTargetWithInput(oracle, bytes.NewReader(request.Bytes()))
		if err != nil || independent.ExitCode != out.ExitCode || independent.Stdout != out.Stdout || independent.Stderr != out.Stderr {
			t.Fatal("Go/Fairy independent native build behavior differs", independent, err)
		}
		badTemp := filepath.Join(root, "absent", "temp")
		for _, bit := range []uint64{32, 64, 128, 256, 512, 1024} {
			t.Run(fmt.Sprintf("pre-io-%d", bit), func(t *testing.T) {
				refused := run(mode|bit, map[string]string{"TMP": badTemp, "TEMP": badTemp})
				if refused.stage != 1 || len(refused.blobs[0]) != 0 || refused.osCode != 0 {
					t.Fatal("invalid options/work reached host I/O", refused)
				}
				if bit == 128 && refused.status != 1002 || bit == 256 && (refused.status != 1002 || refused.work != 16777217) || bit != 256 && refused.work > 16777216 {
					t.Fatal("build ledger refusal", refused)
				}
				if bit == 512 && (refused.status != 1002 || refused.unit != 1 || refused.work != 16777216) {
					t.Fatal("runtime source copy did not share the caller ledger", refused)
				}
			})
		}
		for _, failure := range []struct {
			name, cl string
			stage    uint64
		}{
			{"compile", "/FI\"" + filepath.Join(root, "missing-header.h") + "\"", 6},
			{"link", "/Dmain=fy_missing_main", 8},
		} {
			t.Run(failure.name, func(t *testing.T) {
				refused := run(mode, map[string]string{"CL": failure.cl})
				if refused.stage != failure.stage || refused.status == 0 || len(refused.blobs[0]) == 0 || refused.unit != map[uint64]uint64{6: 0, 8: 2}[failure.stage] {
					t.Fatal("lost precise compiler/link failure", refused)
				}
				if strings.Contains(strings.ToLower(strings.ReplaceAll(string(refused.blobs[0]), "\\", "/")), strings.ToLower(filepath.ToSlash(root))+"/fairyc-native-") {
					t.Fatal("scratch path leaked in diagnostics", string(refused.blobs[0]))
				}
			})
		}
	}, "module_state_graph_transport_request")
}

func TestCandidateStateGraphWindowsBuildSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, body string }{
		{"built-constructor", `fn build::Built forge(){return build::Built{.prepared=0,.image=bytes::empty(),.objects=2};}`},
		{"built-image", `fn void forge(ref mut build::Built p){p.image=bytes::empty();}`},
		{"built-prepared", `fn void forge(ref mut build::Built p){p.prepared=0;}`},
		{"built-count", `fn void forge(ref mut build::Built p){p.objects=2;}`},
		{"failure-constructor", `fn build::Failure forge(){return build::Failure{.stage=0,.status=0,.os_code=0,.unit=0,.diagnostics=bytes::empty()};}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_windows_build")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_windows_build as build;import std::bytes as bytes;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed graph MSVC build", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), "E0103") {
				t.Fatalf("sealed build owner code=%d: %s", code, &diagnostics)
			}
		})
	}
}
