package cli

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

func TestPracticalFieldCompound(t *testing.T) {
	toolchainAvailable := runtime.GOOS == "windows" && runtime.GOARCH == "amd64"
	var toolchain cdriver.WindowsHostToolchain
	if toolchainAvailable {
		var err error
		toolchain, err = cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct{ name, source, trap string }{
		{"checked-and-order", practicalFieldProgram, ""},
		{"unsigned-overflow", `module main; struct S { u8 n; } fn i32 main(){ var S s=S{.n=255}; s.n+=1; return 0; }`, "integer-overflow"},
		{"divide-zero", `module main; struct S { i32 n; } fn i32 main(){ var S s=S{.n=42}; s.n/=0; return 0; }`, "division-by-zero"},
		{"signed-overflow", `module main; struct S { i32 n; } fn i32 main(){ var S s=S{.n=2147483647}; s.n+=1; return 0; }`, "integer-overflow"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.field-compound\"\nname = \"fields\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", item.source)
			var diagnostics strings.Builder
			lowered, code := lowerWorkspaceExecutable(root, false, workspaceCompilePolicy{stableDiagnostics: true}, "field compound", &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("lower: %d %s", code, diagnostics.String())
			}
			encoded, err := mircodec.Encode(lowered.program, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := interpreter.Execute(decoded, lowered.entryName)
			if err != nil || outcome.TrapCategory != item.trap || item.trap == "" && (outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0) {
				t.Fatalf("interpreter round trip: %+v %v", outcome, err)
			}
			if !toolchainAvailable {
				t.Log("native Windows profiles not exercised on this host")
				return
			}
			unit, err := cbackend.EmitStorageTransfers(decoded, lowered.entryName, cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			support, err := windowsVectorSupportSource(unit.ForeignSymbols, true)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				artifact.TrapControl = true
				result, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				_, trap, classificationErr := classifyStableRunOutcome(result)
				if err != nil || classificationErr != nil || trap != item.trap || result.Stdout != "" || result.Stderr != "" || item.trap == "" && result.ExitCode != 0 {
					t.Fatalf("native %s: %+v %v %v", profile.Name, result, err, classificationErr)
				}
			}
		})
	}
}

const practicalFieldProgram = `module main;
struct State { usize position; i32 calls; i32 n; u8 byte; u32 word; }
resource Owner { usize position; cleanup {} }
fn usize next(ref mut State s) { s.calls += 1; s.position = 99; return 2; }
fn void advance(ref mut State s) { s.position += next(s); }
fn i32 main() {
 var State s=State{.position=3,.calls=0,.n=10,.byte=2,.word=10};
 var State copy=s;
 advance(ref mut s);
 assert(s.position==5 && s.calls==1 && copy.position==3 && copy.calls==0);
 s.n -= 2; s.n *= 3; s.n /= 4; s.n %= 5;
 assert(s.n==1);
 s.n <<= 3; s.n >>= 1; s.n |= 3; s.n ^= 1; s.n &= 6;
 assert(s.n==6);
 s.byte += 2; s.byte *= 3; s.word <<= 2; s.word /= 4;
 assert(s.byte==12 && s.word==10);
 var Owner owner=Owner{.position=0}; owner.position+=s.position;
 assert(owner.position==5);
 return 0;
}`
