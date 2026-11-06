package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const graphFrameTool = "tools/package-candidate-state-graph-frame.py"
const graphFramePin = "examples/lexer_candidate/src/module_state_graph_frame_pin.fy"
const graphFrameSource = "internal/contractstatenative/graph_native_c.go"

func graphFramePython() string {
	if runtime.GOOS == "windows" {
		return "python"
	}
	return "python3"
}

func graphFramePackage(t *testing.T, script, output string, args ...string) []byte {
	t.Helper()
	command := append([]string{"-B", script, "--output", output}, args...)
	if out, err := exec.CommandContext(t.Context(), graphFramePython(), command...).CombinedOutput(); err != nil {
		t.Fatalf("package graph frame: %v %s", err, out)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Read the actual authority and use Go's parser and string semantics, independent
// of the Python regex, pin offsets, and the pack's declared section length.
func graphFrameExpected(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../..", graphFrameSource))
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), graphFrameSource, data, 0)
	if err != nil {
		t.Fatal(err)
	}
	var found []byte
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "graphNativeFrameC" {
			return true
		}
		if found != nil || len(spec.Values) != 1 {
			t.Fatal("ambiguous graph frame authority")
		}
		literal, ok := spec.Values[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			t.Fatal("graph frame is not a Go string literal")
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		found = []byte(value)
		return false
	})
	if len(found) == 0 || found[0] != '\n' || found[len(found)-1] != '\n' {
		t.Fatal("missing frame or lost boundary newlines")
	}
	// A second, public selector must append exactly this entire raw value.
	f := stateGraphBodyFixtures(t)[0]
	_, _, _, handoff := graphNativeOracle(t, f, false)
	transport, err := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	source, err := transport.Source()
	if err != nil || !strings.HasSuffix(source, string(found)) {
		t.Fatalf("public GraphTransport.Source frame suffix: %v", err)
	}
	return found
}

func TestCandidateStateGraphFrameAssetsPin(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{graphFrameTool, graphFramePin, graphFrameSource} {
		data, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, path, string(data))
	}
	script, output := filepath.Join(root, graphFrameTool), filepath.Join(root, "frame.pack")
	first := graphFramePackage(t, script, output)
	if second := graphFramePackage(t, script, output); !bytes.Equal(first, second) {
		t.Fatal("nondeterministic frame pack")
	}
	frame := graphFrameExpected(t)
	header := []byte("FYCSGRAPHFRAME1\n")
	start := len(header) + 8
	if len(first) != start+len(frame) || !bytes.Equal(first[:len(header)], header) || binary.LittleEndian.Uint32(first[len(header):]) != 1 || binary.LittleEndian.Uint32(first[len(header)+4:]) != uint32(len(frame)) || !bytes.Equal(first[start:], frame) {
		t.Fatal("pack envelope or complete raw frame differs from Go")
	}
	pin, err := os.ReadFile(filepath.Join(root, graphFramePin))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{fmt.Sprintf("return %d;", len(first)), fmt.Sprintf("return %d;", start), fmt.Sprintf("%x", sha256.Sum256(first))} {
		if !bytes.Contains(pin, []byte(expected)) {
			t.Fatalf("missing independently derived pin fact %q", expected)
		}
	}
	reject := func(want string, args ...string) {
		t.Helper()
		if err := os.Remove(output); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		command := append([]string{"-B", script, "--output", output}, args...)
		out, err := exec.CommandContext(t.Context(), graphFramePython(), command...).CombinedOutput()
		if err == nil || !strings.Contains(string(out), want) {
			t.Fatalf("want %q rejection: %v %s", want, err, out)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("rejected pack published output: %v", err)
		}
	}
	if err := os.Remove(filepath.Join(root, graphFramePin)); err != nil {
		t.Fatal(err)
	}
	reject("pin drift")
	if updated := graphFramePackage(t, script, output, "--update-pin"); !bytes.Equal(first, updated) {
		t.Fatal("explicit missing-pin update changed pack")
	}
	writeTestSource(t, root, graphFramePin, "stale pin\n")
	reject("pin drift")
	graphFramePackage(t, script, output, "--update-pin")
	original, err := os.ReadFile(filepath.Join(root, graphFrameSource))
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(original, []byte("#include <stdio.h>"), []byte("/* source drift */\n#include <stdio.h>"), 1)
	writeTestSource(t, root, graphFrameSource, string(changed))
	reject("pin drift")
	if updated := graphFramePackage(t, script, output, "--update-pin"); bytes.Equal(first, updated) {
		t.Fatal("explicit source update did not update pack")
	}
	for _, trial := range []struct{ name, source, want string }{
		{"CRLF", strings.ReplaceAll(string(original), "\n", "\r\n"), "noncanonical graph frame source"},
		{"declaration", strings.Replace(string(original), "const graphNativeFrameC", "const otherFrame", 1), "anchored graphNativeFrameC"},
		{"suffix", string(original) + "const unexpected = 1\n", "anchored graphNativeFrameC"},
		{"input-size", "package contractstatenative\nconst graphNativeFrameC = `\n" + strings.Repeat("x", 1048576) + "\n`\n", "exceeds 1MiB"},
		{"output-size", "package contractstatenative\nconst graphNativeFrameC = `\n" + strings.Repeat("x", 8388608) + "\n`\n", "exceeds 8MiB"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			writeTestSource(t, root, graphFrameSource, trial.source)
			reject(trial.want, "--update-pin")
		})
	}
}

const graphFrameAssetsProbe = `module main;
import module_state_graph_frame_assets as assets;
import std::bytes as bytes;
import std::io as io;
fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn bool load_error(ref u8[] raw,usize initial,u32 expected){var usize work=initial;return switch(assets::load(copy(raw),ref mut work)){case .ok(_):{false}case .err(code):{code==expected && work==initial}};}
// The admitted owner escapes both a local input copy and a local scope.
fn assets::Bundle owned(ref u8[] raw){
 var usize work=0;let result={let local=copy(raw);switch(assets::load(local,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);owned(raw)}}};
 let cost=raw.len*2+1;assert(work==cost && cost<=16777216);
 work=16777216-cost;assert(assets::load(copy(raw),ref mut work).is_ok && work==16777216);
 assert(load_error(raw,16777217-cost,1002));assert(load_error(raw,16777217,1002));
 return result;
}
// A parent may explicitly copy a view for use after Bundle destruction.
fn bytes::Buffer detached(ref u8[] raw){let bundle=owned(raw);return copy(assets::content(ref bundle));}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>1048578-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 if(bytes::len(ref input)==0){return 3;}let mode=bytes::view(ref input)[0];
 if(mode==1){var usize work=0;switch(assets::load(copy(bytes::view(ref input)[1..]),ref mut work)){case .ok(_):{discard io::write("accepted");}case .err(code):{discard io::write_u32(code);}};return 0;}
 let expected=detached(bytes::view(ref input)[1..]);let bundle=owned(bytes::view(ref input)[1..]);
 {let mutable=bytes::view_mut(ref mut input);var usize i=0;while(i<mutable.len){mutable[i]=0;i+=1;}}input=bytes::empty();
 assert(bytes::equal(assets::content(ref bundle),bytes::view(ref expected)));
 var changed=copy(assets::content(ref bundle));{let raw=bytes::view_mut(ref mut changed);raw[0]=0;}
 assert(bytes::equal(assets::content(ref bundle),bytes::view(ref expected)));
 if(!io::write_bytes(assets::content(ref bundle)).is_ok){return 4;}return 0;
}`

func TestCandidateStateGraphFrameAssets(t *testing.T) {
	pack := graphFramePackage(t, "../../"+graphFrameTool, filepath.Join(t.TempDir(), "frame.pack"))
	want := string(graphFrameExpected(t))
	candidateAtlasProbeSize(t, "module_state_graph_frame_assets", graphFrameAssetsProbe, 32<<10, func(t *testing.T, image string) {
		if code, out := candidateAtlasProcess(t, image, append([]byte{0}, pack...)); code != 0 || out != want {
			t.Fatalf("full raw frame parity: exit=%d got=%d want=%d", code, len(out), len(want))
		}
		for _, offset := range []int{0, 14, 15, 19, 22, 23, len(pack) / 2, len(pack) - 1} {
			bad := bytes.Clone(pack)
			bad[offset] ^= 1
			if code, out := candidateAtlasProcess(t, image, append([]byte{1}, bad...)); code != 0 || out != "1001" {
				t.Fatalf("corruption at %d: exit=%d output=%q", offset, code, out)
			}
		}
		for _, length := range []int{0, 1, 22, len(pack) - 1, len(pack) + 1, 1048576, 1048577} {
			bad := make([]byte, length)
			copy(bad, pack)
			want := "1001"
			if length > 1048576 {
				want = "1002"
			}
			if code, out := candidateAtlasProcess(t, image, append([]byte{1}, bad...)); code != 0 || out != want {
				t.Fatalf("size %d: exit=%d output=%q want=%q", length, code, out, want)
			}
		}
		t.Log("complete frame bytes, corruption, bounds, exact/short/overdrawn work and detached lifetime passed; whole graph build is the parent's gate")
	})
}

func TestCandidateStateGraphFrameAssetsSealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"construct", "E0103", `fn assets::Bundle forge(){return assets::Bundle{.data=bytes::empty()};}`},
		{"read-field", "E0103", `fn usize forge(ref assets::Bundle p){return bytes::len(ref p.data);}`},
		{"mutate-field", "E0103", `fn void forge(ref mut assets::Bundle p){p.data=bytes::empty();}`},
		{"mutable-view", "E1008", `fn void forge(ref assets::Bundle p){let raw=assets::content(p);raw[0]=0;}`},
		{"escape-owner", "E2102", `fn ref u8[] forge(assets::Bundle p){return assets::content(ref p);}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_frame_assets")
			writeTestSource(t, root, "src/main.fy", fmt.Sprintf("module main;import module_state_graph_frame_assets as assets;import std::bytes as bytes;%s fn i32 main(){return 0;}", trial.body))
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed frame Bundle", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), trial.code) {
				t.Fatalf("sealed owner: code=%d diagnostics=%s", code, &diagnostic)
			}
		})
	}
}
