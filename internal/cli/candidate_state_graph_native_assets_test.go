package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func graphNativeAssetsPackage(t *testing.T, script, output string, args ...string) []byte {
	t.Helper()
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	command := append([]string{"-B", script, "--output", output}, args...)
	out, err := exec.Command(python, command...).CombinedOutput()
	if err != nil {
		t.Fatalf("package graph assets: %v %s", err, out)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Copy the distribution tool and its five actual source authorities into an
// isolated checkout. Drift controls must never edit the live C files or pin.
func TestCandidateStateGraphNativeAssetsPin(t *testing.T) {
	root := t.TempDir()
	paths := []string{"tools/package-candidate-state-graph-runtime.py", "examples/lexer_candidate/src/module_state_graph_native_asset_pin.fy"}
	for _, name := range []string{"overlay.h", "overlay.c", "graph.h", "graph.c", "graph_key.c"} {
		paths = append(paths, "internal/cstdlib/stage0_contract_state_"+name)
	}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, path, string(data))
	}
	script := filepath.Join(root, paths[0])
	output := filepath.Join(root, "runtime.pack")
	first := graphNativeAssetsPackage(t, script, output)
	if second := graphNativeAssetsPackage(t, script, output); !bytes.Equal(first, second) {
		t.Fatal("graph bundle is nondeterministic")
	}
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	reject := func(want string) {
		t.Helper()
		if err := os.Remove(output); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		out, err := exec.Command(python, "-B", script, "--output", output).CombinedOutput()
		if err == nil || !strings.Contains(string(out), want) {
			t.Fatalf("expected %q rejection: %v %s", want, err, out)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("failed packaging published output: %v", err)
		}
	}
	pin := filepath.Join(root, paths[1])
	if err := os.Remove(pin); err != nil {
		t.Fatal(err)
	}
	reject("pin drift")
	if regenerated := graphNativeAssetsPackage(t, script, output, "--update-pin"); !bytes.Equal(first, regenerated) {
		t.Fatal("regenerating missing pin changed bundle")
	}
	writeTestSource(t, root, paths[1], "stale pin\n")
	reject("pin drift")
	graphNativeAssetsPackage(t, script, output, "--update-pin")
	source := paths[len(paths)-1]
	original, err := os.ReadFile(filepath.Join(root, source))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, source, string(original)+"/* explicit update control */\n")
	reject("pin drift")
	if updated := graphNativeAssetsPackage(t, script, output, "--update-pin"); bytes.Equal(first, updated) {
		t.Fatal("source update did not change pinned bundle")
	}
	writeTestSource(t, root, source, strings.ReplaceAll(string(original), "\n", "\r\n"))
	reject("noncanonical graph runtime source")
}

const graphNativeAssetsProbe = `module main;
import module_state_graph_native_assets as assets;
import std::bytes as bytes;
import std::io as io;
fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn bool load_error(ref u8[] raw,usize initial,u32 expected){var usize work=initial;return switch(assets::load(copy(raw),ref mut work)){case .ok(_):{false}case .err(code):{code==expected && work==initial}};}
fn assets::Bundle owned(ref u8[] raw){
 var usize work=0;let result=switch(assets::load(copy(raw),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);owned(raw)}};let cost=work;
 assert(cost>0 && cost<=16777216);
 work=16777216-cost;assert(assets::load(copy(raw),ref mut work).is_ok && work==16777216);
 assert(load_error(raw,16777217-cost,1002));assert(load_error(raw,16777217,1002));
 return result;
}
// Returned bytes survive destruction of this temporary Bundle and its input.
fn bytes::Buffer detached(ref u8[] raw,bool header){var usize work=0;let bundle=switch(assets::load(copy(raw),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);owned(raw)}};return switch(if(header){assets::declarations(ref bundle,ref mut work)}else{assets::runtime(ref bundle,ref mut work)}){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>1048578-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 if(bytes::len(ref input)==0){return 3;}let mode=bytes::view(ref input)[0];
 if(mode==1){var usize work=0;switch(assets::load(copy(bytes::view(ref input)[1..]),ref mut work)){case .ok(_):{discard io::write("accepted");}case .err(code):{discard io::write_u32(code);}};return 0;}
 let expected_header=detached(bytes::view(ref input)[1..],true);let expected_runtime=detached(bytes::view(ref input)[1..],false);
 let bundle=owned(bytes::view(ref input)[1..]);
 // The original caller buffer is destroyed before either extraction.
 {let mutable=bytes::view_mut(ref mut input);var usize i=0;while(i<mutable.len){mutable[i]=0;i+=1;}}input=bytes::empty();
 var usize work=0;var header=switch(assets::declarations(ref bundle,ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};let header_cost=work;
 work=0;var implementation=switch(assets::runtime(ref bundle,ref mut work)){case .ok(p):{p}case .err(_):{return 5;}};let runtime_cost=work;
 assert(bytes::equal(bytes::view(ref header),bytes::view(ref expected_header)));assert(bytes::equal(bytes::view(ref implementation),bytes::view(ref expected_runtime)));
 work=16777216-header_cost;assert(assets::declarations(ref bundle,ref mut work).is_ok && work==16777216);
 work=16777217-header_cost;switch(assets::declarations(ref bundle,ref mut work)){case .ok(_):{return 6;}case .err(code):{assert(code==1002 && work==16777217-header_cost);}};
 work=16777216-runtime_cost;assert(assets::runtime(ref bundle,ref mut work).is_ok && work==16777216);
 work=16777217-runtime_cost;switch(assets::runtime(ref bundle,ref mut work)){case .ok(_):{return 7;}case .err(code):{assert(code==1002 && work==16777217-runtime_cost);}};
 work=16777217;switch(assets::declarations(ref bundle,ref mut work)){case .ok(_):{return 8;}case .err(code):{assert(code==1002 && work==16777217);}};
 switch(assets::runtime(ref bundle,ref mut work)){case .ok(_):{return 9;}case .err(code):{assert(code==1002 && work==16777217);}};
 // Mutating an extracted output cannot mutate the opaque Bundle.
 {let raw=bytes::view_mut(ref mut header);raw[0]=0;}{let raw=bytes::view_mut(ref mut implementation);raw[0]=0;}
 work=0;let again_header=switch(assets::declarations(ref bundle,ref mut work)){case .ok(p):{p}case .err(_):{return 10;}};
 let again_runtime=switch(assets::runtime(ref bundle,ref mut work)){case .ok(p):{p}case .err(_):{return 11;}};
 assert(bytes::equal(bytes::view(ref again_header),bytes::view(ref expected_header)));assert(bytes::equal(bytes::view(ref again_runtime),bytes::view(ref expected_runtime)));
 if(!io::write_bytes(bytes::view(ref again_header)).is_ok || !io::write_bytes(bytes::view(ref again_runtime)).is_ok){return 12;}return 0;
}`

func TestCandidateStateGraphNativeAssets(t *testing.T) {
	root := t.TempDir()
	pack := graphNativeAssetsPackage(t, "../../tools/package-candidate-state-graph-runtime.py", filepath.Join(root, "graph.pack"))
	declarations, err := cstdlib.ContractStateGraphDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := cstdlib.ContractStateGraphRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	candidateAtlasProbe(t, "module_state_graph_native_assets", graphNativeAssetsProbe, func(t *testing.T, image string) {
		// Compare complete Fairy-emitted bytes against the live Go selectors,
		// independently of the manifest or the packaging implementation.
		code, output := candidateAtlasProcess(t, image, append([]byte{0}, pack...))
		if code != 0 || output != declarations+implementation {
			t.Fatalf("full graph runtime byte parity: exit=%d got=%d want=%d", code, len(output), len(declarations)+len(implementation))
		}
		for _, offset := range []int{0, 17, 25, len(pack) / 2, len(pack) - 1} {
			bad := append([]byte(nil), pack...)
			bad[offset] ^= 1
			code, out := candidateAtlasProcess(t, image, append([]byte{1}, bad...))
			if code != 0 || out != "1001" {
				t.Fatalf("corruption at %d: exit=%d output=%q", offset, code, out)
			}
		}
		for _, length := range []int{0, 1, 25, len(pack) - 1, len(pack) + 1, 1048577} {
			bad := make([]byte, length)
			copy(bad, pack)
			code, out := candidateAtlasProcess(t, image, append([]byte{1}, bad...))
			want := "1001"
			if length > 1048576 {
				want = "1002"
			}
			if code != 0 || out != want {
				t.Fatalf("size %d: exit=%d output=%q want=%q", length, code, out, want)
			}
		}
		profile := cdriver.LowProfile
		if strings.HasSuffix(t.Name(), "/"+cdriver.OptimizedProfile.Name) {
			profile = cdriver.OptimizedProfile
		}
		tc, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		// Use ONLY the bytes emitted by Fairy for the linked graph implementation.
		probe, err := cdriver.BuildWindowsMSVCProbe(tc, profile, output+graphNativeAssetsRuntimeMain)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "graph-call-"+profile.Name+".exe")
		if err := os.WriteFile(path, probe.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		if code, out := candidateAtlasProcess(t, path, nil); code != 0 || out != "" {
			t.Fatalf("genuine graph call: exit=%d output=%q", code, out)
		}
		t.Logf("full-byte parity, pin/corruption/size/work/lifetime controls and linked graph calls passed (%s)", profile.Name)
	})
}

const graphNativeAssetsRuntimeMain = `
int main(void) {
    fycs_overlay_v1 tx; fycs_overlay_entry_v1 writes[2];
    uint8_t active[64], scratch[64], values[64], key[32] = {1};
    uint8_t input[4] = {0, 0, 0, 42}, found = 0, digest[32];
    fycs_graph_v1 graph; fycs_graph_epoch_v1 epochs[2];
    fycs_graph_value_v1 value, read;
    if (fycs_overlay_begin_v1(&tx, NULL, 0, writes, 2, active, scratch, 64, 64, 32) != 0) return 1;
    if (fycs_graph_begin_v1(&graph, &tx, values, 64, epochs, 2) != 0) return 2;
    if (fycs_graph_value_copy_v1(&graph, FYCS_CODEC_U32, input, 4, &value) != 0) return 3;
    input[3] = 0;
    if (value.bytes[3] != 42 || fycs_graph_write_v1(&graph, key, &value, 0) != 0) return 4;
    if (fycs_graph_read_v1(&graph, key, FYCS_CODEC_U32, &read, &found) != 0 || found != 1 || read.length != 4 || read.bytes[3] != 42) return 5;
    if (fycs_graph_digest_v1(key, 32, read.bytes, read.length, digest) != 0) return 6;
    if (fycs_overlay_commit_v1(&tx) != 0) return 7;
    if (fycs_graph_read_v1(&graph, key, FYCS_CODEC_U32, &read, &found) != FYCS_OV_CLOSED) return 8;
    return 0;
}
`

func TestCandidateStateGraphNativeAssetsSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"construct", "E0103", `fn assets::Bundle forge(){return assets::Bundle{.data=bytes::empty()};}`},
		{"read-field", "E0103", `fn usize forge(ref assets::Bundle p){return bytes::len(ref p.data);}`},
		{"mutate-field", "E0103", `fn void forge(ref mut assets::Bundle p){p.data=bytes::empty();}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_native_assets")
			writeTestSource(t, root, "src/main.fy", fmt.Sprintf("module main;import module_state_graph_native_assets as assets;import std::bytes as bytes;%s fn i32 main(){return 0;}", trial.body))
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed graph asset Bundle", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), trial.code) {
				t.Fatalf("sealed owner: code=%d diagnostics=%s", code, &diagnostic)
			}
		})
	}
}
