package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

type bytesEqualCase struct {
	Name     string `json:"name"`
	LeftHex  string `json:"leftHex"`
	RightHex string `json:"rightHex"`
	Equal    bool   `json:"equal"`
}

func bytesEqualCases() []bytesEqualCase {
	var cases []bytesEqualCase
	add := func(name string, left, right []byte) {
		cases = append(cases, bytesEqualCase{name, hex.EncodeToString(left), hex.EncodeToString(right), bytes.Equal(left, right)})
	}
	add("empty", nil, nil)
	add("empty/nonempty", nil, []byte{0})
	add("proper-prefix", []byte{1, 2}, []byte{1, 2, 3})
	add("different-length", []byte{9}, []byte{1, 2})
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	add("all-bytes", all, all)
	for _, at := range []int{0, 127, 255} {
		other := bytes.Clone(all)
		other[at] ^= 255
		add(fmt.Sprintf("mismatch-%d", at), all, other)
	}
	for _, raw := range []string{"a\x00b", "\xef\xbb\xbf", "\xc0\xaf\xff\x80", "한글"} {
		add("binary/text-"+hex.EncodeToString([]byte(raw)), []byte(raw), []byte(raw))
	}
	add("BOM-not-trimmed", []byte("\xef\xbb\xbf0"), []byte("0"))
	add("no-normalization", []byte("é"), []byte("e\u0301"))
	add("no-folding", []byte("A"), []byte("a"))
	add("no-trimming", []byte(" a"), []byte("a"))
	return cases
}

func bytesEqualSource(cases []bytesEqualCase) string {
	var source strings.Builder
	source.WriteString(`module main;
import std::bytes as bytes;
fn bool forward(ref u8[] left, ref u8[] right) { return bytes::equal(left, right); }
fn ref u8[] retain(ref u8[] view) { return view; }
fn void check(ref u8[] left, ref u8[] right, bool want) {
    assert(bytes::equal(left, right) == want);
    assert(bytes::equal(right, left) == want);
    assert(forward(left, right) == want && forward(right, left) == want);
}
fn void preserved(ref u8[] actual, ref u8[] saved) {
    assert(actual.len == saved.len);
    var usize at = 0;
    while (at < actual.len) { assert(actual[at] == saved[at]); at += 1; }
}
fn i32 main() {
`)
	literal := func(raw []byte) string {
		parts := make([]string, len(raw))
		for i, v := range raw {
			parts[i] = fmt.Sprint(v)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	for _, tc := range cases {
		left, _ := hex.DecodeString(tc.LeftHex)
		right, _ := hex.DecodeString(tc.RightHex)
		// Different surrounding bytes cannot affect equal subviews.
		a := append(append([]byte{85}, left...), 170)
		b := append(append([]byte{17}, right...), 238)
		fmt.Fprintf(&source, "    { var u8[%d] a = %s; var u8[%d] b = %s; let u8[%d] savedA = %s; let u8[%d] savedB = %s; let ref u8[] av = ref a; let ref u8[] bv = ref b; check(av[1..%d], bv[1..%d], %t); preserved(av, ref savedA); preserved(bv, ref savedB); }\n",
			len(a), literal(a), len(b), literal(b), len(a), literal(a), len(b), literal(b), len(left)+1, len(right)+1, tc.Equal)
	}
	source.WriteString(`
    assert(bytes::equal(b"", b""));
    assert(!bytes::equal(b"", b"x") && !bytes::equal(b"x", b""));
    {
        var u8[5] owner = [77, 9, 9, 9, 88];
        let u8[5] saved = [77, 9, 9, 9, 88];
        let ref u8[] view = ref owner;
        check(view, view, true);
        check(view[1..3], view[2..4], true);
        check(view[0..0], view[5..5], true);
        check(view[0..2], view[1..3], false);
        preserved(view, ref saved);
        owner[0] = 66;
        assert(owner[0] == 66);
    }
    {
        var u8[2] data = [3, 4];
        let ref mut u8[] view = ref mut data;
        assert(bytes::equal(view, view));
        assert(forward(view, b"\x03\x04"));
        let bool result = bytes::equal(view, b"\x03\x04");
        view[0] = 9;
        assert(result && bytes::equal(view, b"\x09\x04"));
        let ref u8[] child = retain(view);
        assert(bytes::equal(child, b"\x09\x04"));
        view[1] = 7;
        assert(bytes::equal(view, b"\x09\x07"));
    }
    return 0;
}
`)
	return source.String()
}

func TestBytesEqualExecution(t *testing.T) {
	cases := bytesEqualCases()
	text := bytesEqualSource(cases)
	checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, text))
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	entry := tokenScannerEntry(t, checked)
	encoded, err := mircodec.Encode(checked.MIR, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, entry)
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("interpreter: %v %+v", err, outcome)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		if _, err := cbackend.Emit(decoded, entry, profile); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d bytes.Equal oracle pairs, both orders, owner preservation, overlap and reborrows passed", len(cases))
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 native evidence; interpreter already checked")
	}
	evidenceRoot, err := filepath.Abs(filepath.Join("..", "..", ".cache", "bytes-equal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	hash := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	definition, err := sdkstdlib.Load()
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	compilerBytes, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	buildInfo := "unavailable"
	if info, ok := debug.ReadBuildInfo(); ok {
		buildInfo = info.String()
	}
	var images []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			var diagnostics bytes.Buffer
			artifact, code := compileWindowsProgram(decoded, entry, target.X8664WindowsMSVC, profile, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("native build: %s", diagnostics.String())
			}
			exe := filepath.Join(evidenceRoot, profile.Name+".exe")
			if err := os.WriteFile(exe, artifact.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, exe)
			command.Dir = evidenceRoot
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			runErr := command.Run()
			// Decode native byte pipes explicitly as UTF-8, with no host codepage fallback.
			if !utf8.Valid(stdout.Bytes()) || !utf8.Valid(stderr.Bytes()) {
				t.Fatal("invalid UTF-8 native output")
			}
			if runErr != nil || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("native run: %v stdout=%q stderr=%q", runErr, stdout.String(), stderr.String())
			}
			images = append(images, map[string]any{"profile": profile.Name, "exeSHA256": hash(artifact.Bytes), "exeBytes": len(artifact.Bytes), "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version, "argv": command.Args, "cwd": command.Dir, "stdin": "closed", "outputEncoding": "UTF-8", "stdout": stdout.String(), "stderr": stderr.String(), "stdoutHex": hex.EncodeToString(stdout.Bytes()), "stderrHex": hex.EncodeToString(stderr.Bytes()), "exitCode": command.ProcessState.ExitCode()})
		})
	}
	if t.Failed() {
		return
	}
	receipt := map[string]any{"schema": "fairyc.bytes-equal/v1", "sdk": sdkstdlib.Version, "sdkSourceIdentity": definition.SourceIdentity, "language": sdkstdlib.LanguageRevision, "runtime": sdkstdlib.RuntimeABI, "sourceSHA256": hash([]byte(text)), "cases": cases, "oracle": "Go bytes.Equal", "interpreter": "passed after MIR codec round trip", "cEmission": []string{"x86_64-linux-gnu", "x86_64-windows-msvc"}, "compilerIdentity": map[string]any{"kind": "in-process CLI compiler in Go test executable", "executableSHA256": hash(compilerBytes), "goVersion": runtime.Version(), "buildInfo": buildInfo}, "images": images}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "main.fy"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestBytesEqualBorrowConflicts(t *testing.T) {
	for name, body := range map[string]string{
		"later argument view store":  "bytes::equal(view, { view[0] = 2; view });",
		"later argument owner store": "bytes::equal(view, { data[0] = 2; view });",
		"live child mutation":        "let ref u8[] child = retain(view); view[0] = 2; assert(bytes::equal(child, b\"\\x01\"));",
		"live owner mutation":        "let ref u8[] child = ref data; data[0] = 2; assert(bytes::equal(child, b\"\\x01\"));",
		"live resource move":         "var bytes::Buffer owner = bytes::empty(); let ref u8[] child = bytes::view(ref owner); let bytes::Buffer moved = owner; assert(bytes::equal(child, b\"\"));",
	} {
		t.Run(name, func(t *testing.T) {
			text := "module main; import std::bytes as bytes; fn ref u8[] retain(ref u8[] view) { return view; } fn i32 main() { var u8[1] data = [1]; let ref mut u8[] view = ref mut data; " + body + " return 0; }"
			checked, err := workspacecompiler.Check(checkedIntegerConversionWorkspace(t, text))
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified || len(checked.Semantic.Diagnostics) == 0 || len(checked.Semantic.Unsupported) != 0 {
				t.Fatalf("missing borrow rejection: %+v %+v", checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			for _, diagnostic := range checked.Semantic.Diagnostics {
				want := "E2101"
				if name == "live resource move" {
					want = "E2105"
				}
				if diagnostic.Code != want {
					t.Fatalf("unexpected diagnostic: %+v", diagnostic)
				}
			}
		})
	}
}

func TestBytesEqualSDKCache(t *testing.T) {
	text := "module main; import std::bytes as bytes; fn i32 main() { assert(bytes::equal(b\"abc\", b\"abc\")); return 0; }"
	workspace := checkedIntegerConversionWorkspace(t, text)
	sdk, found := workspace.Package(sdkstdlib.PackagePath)
	if !found || sdk.Manifest.Package.Version != "0.0.0-stage0.77" {
		t.Fatal("wrong SDK bundle")
	}
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := session.Check(workspace)
		if err != nil || !result.Compiler.Verified {
			t.Fatalf("cache check: %v %+v", err, result)
		}
		if attempt == 0 && result.Cache.Misses != 2 || attempt == 1 && result.Cache.Outcome != workspacecompiler.OutcomeAllHit {
			t.Fatalf("unexpected cache result: %+v", result.Cache)
		}
		outcome, err := interpreter.Execute(result.Compiler.MIR, tokenScannerEntry(t, result.Compiler))
		if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
			t.Fatalf("cached execution: %v %+v", err, outcome)
		}
	}
}

func TestBytesEqualSDKExportIdentity(t *testing.T) {
	// Model an older source bundle by removing only the added function. Do not
	// rewrite historical fixtures or depend on a locally installed old compiler.
	workspace := checkedIntegerConversionWorkspace(t, "module main; import std::bytes as bytes; fn i32 main() { return 0; }")
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, checked)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	var sdkID hir.PackageID
	for _, p := range metadata {
		if p.Path == sdkstdlib.PackagePath {
			sdkID = p.ID
		}
	}
	descriptor, err := moduleinterface.Extract(checked.Semantic.HIR, sdkID, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	newInterface, err := moduleinterface.Compute(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for m := range descriptor.Modules {
		if descriptor.Modules[m].Path != "std::bytes" {
			continue
		}
		for i, item := range descriptor.Modules[m].Items {
			if item.Name != "equal" {
				continue
			}
			if item.Kind != "function" || item.ResultType != "bool" || len(item.Parameters) != 2 || item.Unsafe || item.ABI != "" || item.LinkName != "" {
				t.Fatalf("unexpected public equality interface: %+v", item)
			}
			descriptor.Modules[m].Items = append(descriptor.Modules[m].Items[:i], descriptor.Modules[m].Items[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("public interface is missing equality")
	}
	oldInterface, err := moduleinterface.Compute(descriptor)
	if err != nil || oldInterface == newInterface {
		t.Fatal("export did not change public interface")
	}

	newDigest := packagecontent.Digest{}
	oldDigest := packagecontent.Digest{}
	for p := range workspace.Packages {
		sdk := &workspace.Packages[p]
		if sdk.Path != sdkstdlib.PackagePath {
			continue
		}
		newDigest = sdk.Digest
		var files []packagecontent.File
		for i := range sdk.Sources {
			file := &sdk.Sources[i]
			if file.Path == "std/bytes.fy" {
				raw := string(file.Bytes)
				start := strings.Index(raw, "/// Compares exact logical bytes")
				end := strings.Index(raw, "/// Borrows the initialized prefix for read-only access.")
				if start < 0 || end <= start {
					t.Fatal("missing added source range")
				}
				file.Bytes = []byte(raw[:start] + raw[end:])
				parsed, err := frontend.AnalyzeSyntax(source.NewFile(file.Path, file.Bytes), source.DefaultLimits())
				if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
					t.Fatal("invalid modeled old SDK")
				}
				file.Module = parsed.AST
			}
			files = append(files, packagecontent.File{Path: file.Path, Bytes: file.Bytes})
		}
		sdk.Digest, err = packagecontent.Compute(files)
		if err != nil {
			t.Fatal(err)
		}
		oldDigest = sdk.Digest
		sdk.Manifest.Package.Version = "0.0.0-stage0.76"
	}
	if oldDigest == newDigest {
		t.Fatal("export did not change SDK source digest")
	}
	for n := range workspace.Graph.Nodes {
		node := &workspace.Graph.Nodes[n]
		if node.Role != packagegraph.RoleStandardLibrary {
			continue
		}
		newID, err := packagegraph.ComputeInstanceID(node.Identity)
		if err != nil {
			t.Fatal(err)
		}
		node.Identity.SourceIdentity = sdkstdlib.SourceIdentity(sdkstdlib.SDKIdentity, sdkstdlib.RuntimeABI, oldDigest)
		node.Identity.VersionIdentity = "0.0.0-stage0.76"
		oldID, err := packagegraph.ComputeInstanceID(node.Identity)
		if err != nil || newID == oldID {
			t.Fatal("SDK lock identity did not change")
		}
	}
	var rejection *packagegraph.Rejection
	workspace.Resolution, rejection, err = packagegraph.Resolve(workspace.Graph)
	if err != nil || rejection != nil {
		t.Fatalf("old bundle graph: %v %+v", err, rejection)
	}
	oldChecked, err := workspacecompiler.Check(workspace)
	if err != nil || !oldChecked.Verified {
		t.Fatalf("old bundle check: %v %+v", err, oldChecked)
	}
	// Real cache reuse must miss on the bundle change even though root bytes stay equal.
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	current := checkedIntegerConversionWorkspace(t, "module main; import std::bytes as bytes; fn i32 main() { return 0; }")
	for _, input := range []struct{ old bool }{{false}, {true}} {
		w := current
		if input.old {
			w = workspace
		}
		result, err := session.Check(w)
		if err != nil || !result.Compiler.Verified || result.Cache.Misses != 2 {
			t.Fatalf("bundle invalidation: %v %+v", err, result.Cache)
		}
	}
	for p := range workspace.Packages {
		if workspace.Packages[p].Path != "." {
			continue
		}
		file := &workspace.Packages[p].Sources[0]
		file.Bytes = []byte("module main; import std::bytes as bytes; fn i32 main() { assert(bytes::equal(b\"a\", b\"a\")); return 0; }")
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(file.Path, file.Bytes), source.DefaultLimits())
		if err != nil || parsed.AST == nil {
			t.Fatal(err)
		}
		file.Module = parsed.AST
	}
	rejected, err := workspacecompiler.Check(workspace)
	if err != nil || rejected.Verified || len(rejected.Semantic.Diagnostics) == 0 {
		t.Fatalf("older bundle resolved new export: %v %+v", err, rejected.Semantic.Diagnostics)
	}
	t.Logf("public interface %s; modeled old interface %s; SDK source and lock/cache identities changed; old bundle rejects equal", newInterface, oldInterface)
}
