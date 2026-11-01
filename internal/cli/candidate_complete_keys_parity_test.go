package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Use the real resolved sources, complete checked candidate IR, compiler-image
// identity and specialization producer, rather than supplying hashes to a codec.
func TestCandidateCompleteCacheKeysV5(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("complete candidate key profile currently requires Windows AMD64; Linux remains deferred")
	}
	if os.Getenv("FAIRYC_RUN_PARITY_INTEGRATION") != "1" {
		t.Skip("set FAIRYC_RUN_PARITY_INTEGRATION=1 for the bounded private compiler API replay")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	fixture := filepath.Join(root, "fixtures")
	command := exec.CommandContext(t.Context(), "go", "run", "-mod=readonly", "tools/verify-candidate-complete-keys.go", "--output", fixture)
	command.Dir = filepath.Join("..", "..")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("complete Go key oracle: %v: %s", err, output)
	}
	testSource, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "project_cache_keys.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(testSource), "module tests::project_cache_keys;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test ", "")
	program += `
@effects(filesystem,environment,console) fn i32 main(){
    let base=root();let env=current();var bytes::Buffer output=bytes::empty();
    assert(bytes::append(ref mut output,b"FYCKOUT1").is_ok);var usize work=0;
    append(ref mut output,environment::compiler_build(ref env),ref mut work);
    assert(wire::length(ref mut output,2,ref mut work)==0);
    var usize i=0;while(i<2){let leaf=if(i==0){b"Plain"}else{b"Aliases"};
        let actual=detached(bytes::view(ref base),leaf,ref env);
        oracle(bytes::view(ref base),leaf,ref actual,environment::compiler_build(ref env),ref mut output);i+=1;}
    assert(switch(io::write_bytes(bytes::view(ref output))){case .ok(n):{n==bytes::len(ref output)}case .err(_):{false}});
    return 0;
}
`
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "project_cache_keys", "project_cache_environment", "project_compiler_core", "project_compiler_input", "project_coordinate_graph", "project_module_visibility", "project_source_set", "project_cache_key", "project_generic_cache_key", "project_cache_encoding", "file_source")
	build, support := candidateParityBuildParts(t, root)
	manifest := filepath.Join(root, "fixture-root.txt")
	if err := os.WriteFile(manifest, []byte(filepath.ToSlash(fixture)), 0o600); err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	// Use the existing private compiler-host builder and explicit 16MiB reserve.
	// Public run/link admission and the default native stack are unchanged.
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Logf("build private %s: %d C parts, header=%d bytes", profile.Name, len(build.Sources), len(build.Header))
		artifact, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		imagePath := filepath.Join(root, "private-"+profile.Name+".exe")
		if err := os.WriteFile(imagePath, artifact.Bytes, 0o700); err != nil {
			t.Fatal(err)
		}
		// This explicit private API replay has its own bounded allowance. Product
		// compiler/run deadlines and the normal CLI linker contract do not change.
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		command := exec.CommandContext(ctx, imagePath, manifest)
		var out, diagnostics bytes.Buffer
		command.Stdout, command.Stderr = &out, &diagnostics
		started := time.Now()
		err = command.Run()
		cancel()
		if err != nil || diagnostics.Len() != 0 {
			t.Fatalf("%s after %s: %v; stderr=%s", profile.Name, time.Since(started), err, &diagnostics)
		}
		validateCandidateKeyLedger(t, out.String(), artifact.Bytes)
		t.Logf("private %s execution: %s", profile.Name, time.Since(started))
	}
}

func candidateParityBuildParts(t *testing.T, root string) (cbackend.BuildUnit, string) {
	t.Helper()
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "candidate parity probe", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower parity probe: %d %s", code, &diagnostics)
	}
	build, err := cbackend.EmitBuildContext(t.Context(), lowered.program, lowered.entryName, cbackend.Stage0WindowsMSVCProfile(), true, 256<<10)
	if err != nil {
		t.Fatal(err)
	}
	if build.Header == "" {
		t.Fatal("expected a partitioned private compiler API probe")
	}
	for i, part := range build.Sources {
		const include = "#include \"fy_bootstrap.h\"\n"
		if !strings.HasPrefix(part, include) {
			t.Fatal("unexpected C partition header")
		}
		build.Sources[i] = "#include \"fy_project.h\"\n" + strings.TrimPrefix(part, include)
	}
	var standard []string
	for _, symbol := range build.Unit.ForeignSymbols {
		if !strings.HasPrefix(symbol, "fairyc_private_windows_") {
			standard = append(standard, symbol)
		}
	}
	support, err := nativesupport.Source(target.X8664WindowsMSVC, standard, nativesupport.Options{SDKStorage: true})
	if err != nil {
		t.Fatal(err)
	}
	host, err := candidatehost.WindowsSource(strings.Join(build.Unit.ForeignSymbols, "(\n") + "(")
	if err != nil {
		t.Fatal(err)
	}
	return build, support + "\n" + host
}

func validateCandidateKeyLedger(t *testing.T, ledger string, image []byte) {
	t.Helper()
	if !strings.HasPrefix(ledger, "FYCKOUT1") {
		t.Fatal("missing key ledger marker")
	}
	reader := bytes.NewReader([]byte(ledger[8:]))
	number := func() uint64 {
		var n uint64
		if err := binary.Read(reader, binary.BigEndian, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	field := func() []byte {
		n := number()
		if n > uint64(reader.Len()) {
			t.Fatal("invalid key ledger field")
		}
		data := make([]byte, int(n))
		if _, err := reader.Read(data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	imageHash := sha256.Sum256(image)
	expected := "fairyc.compiler-executable-sha256/v1:" + hex.EncodeToString(imageHash[:])
	if string(field()) != expected {
		t.Fatal("ledger is not bound to the executed image")
	}
	if cases := number(); cases != 2 {
		t.Fatalf("ledger cases=%d", cases)
	}
	packages := 0
	for range 2 {
		_ = field() // exact case order/inputs are checked by the authored fixture
		n := number()
		if n < 1 || n > 512 {
			t.Fatal("invalid package count")
		}
		for range n {
			_ = field() // package instance
			_ = field() // package coordinate
			for _, prefix := range []string{"fycache5-sha256-", "fycacheg0-sha256-"} {
				canonical, key := field(), field()
				digest := sha256.Sum256(canonical)
				if string(key) != prefix+hex.EncodeToString(digest[:]) || bytes.Count(canonical, []byte(expected)) != 1 {
					t.Fatal("key contradicts preimage/compiler authority")
				}
			}
			packages++
		}
	}
	if reader.Len() != 0 {
		t.Fatal("trailing key ledger bytes")
	}
	t.Logf("verified %d package ordinary/generic pairs across two source graphs; ledger=%d bytes", packages, len(ledger))
}
