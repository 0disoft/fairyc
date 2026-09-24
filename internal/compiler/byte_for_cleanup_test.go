package compiler_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

const byteForCleanupSource = `module main;
OBSERVERS
fn void emit(i32 value) { discard observe(value); }
resource Mark { i32 id; cleanup { let i32 marked = observe(this.id); } }
fn i32 route(ref u8[] view, bool early) {
    for (let u8 byte in view) {
        let Mark mark = Mark { .id = byte as i32 };
        defer emit(byte as i32 + 32);
        if (byte == 65) { continue; }
        if (byte == 66) {
            if (early) { return 41; }
            break;
        }
    }
    return 41;
}
fn i32 main() {
    let ref u8[] view = b"AB"[..];
    assert(route(view, false) == 41);
    assert(route(view, true) == 41);
    return 41 + finish();
}`

const byteForResultCleanupSource = `module main;
OBSERVERS
fn void emit(i32 value) { discard observe(value); }
resource Mark { i32 id; cleanup { let i32 marked = observe(this.id); } }
fn Result<i32, i32> step(u8 byte) {
    if (byte == 66) { return .err(5); }
    return .ok(1);
}
fn Result<i32, i32> route(ref u8[] view) {
    for (let u8 byte in view) {
        let Mark mark = Mark { .id = byte as i32 };
        defer emit(byte as i32 + 32);
        let i32 value = step(byte)?;
    }
    return .ok(41);
}
fn i32 main() {
    let ref u8[] view = b"AB"[..];
    let Result<i32, i32> result = route(view);
    assert(switch (result) { case .ok(_): { false } case .err(code): { code == 5 } });
    return 41 + finish();
}`

func TestByteForCleanupOnContinueBreakAndReturn(t *testing.T) {
	runByteForCleanupProof(t, byteForCleanupSource, "aAbBaAbB", 4)
}

func TestByteForCleanupOnResultPropagation(t *testing.T) {
	runByteForCleanupProof(t, byteForResultCleanupSource, "aAbB", 2)
}

func runByteForCleanupProof(t *testing.T, source, expected string, drops int) {
	t.Helper()
	const interpreterObservers = `fn i32 observe(i32 value) { return value; }
fn i32 finish() { return 1; }`
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(source, "OBSERVERS", interpreterObservers, 1))
	result, err := compiler.CheckExplicitFiles([]semantic.File{file})
	if err != nil || !result.Verified {
		t.Fatalf("compile byte-for cleanup: %v diagnostics=%+v unsupported=%+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	outcome, err := interpreter.ExecuteWithOptions(result.MIR, "main::main", interpreter.Options{
		TestIntrinsics: map[string]interpreter.TestIntrinsicKind{"main::observe": interpreter.TestIntrinsicWriteStdoutByte},
		OutputLimit:    len(expected),
	})
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || string(outcome.Stdout) != expected || len(outcome.DropTrace) != drops {
		t.Fatalf("byte-for cleanup interpreter: %+v %v", outcome, err)
	}

	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return
	}
	const nativeObservers = `@link_name("byte_for_record") unsafe extern C fn i32 native_record(i32 value);
@link_name("byte_for_finished") unsafe extern C fn i32 native_finished();
fn i32 observe(i32 value) { return unsafe { native_record(value) }; }
fn i32 finish() { return unsafe { native_finished() }; }`
	nativeFile := parseCompilerFile(t, hir.DefaultPackage, "main.fy", strings.Replace(source, "OBSERVERS", nativeObservers, 1))
	nativeResult, err := compiler.CheckExplicitFiles([]semantic.File{nativeFile})
	if err != nil || !nativeResult.Verified {
		t.Fatalf("compile native byte-for cleanup: %v diagnostics=%+v unsupported=%+v", err, nativeResult.Semantic.Diagnostics, nativeResult.Semantic.Unsupported)
	}
	unit, err := cbackend.Emit(nativeResult.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalarRuntime, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	var expectedValues strings.Builder
	for index, value := range []byte(expected) {
		if index > 0 {
			expectedValues.WriteString(", ")
		}
		fmt.Fprintf(&expectedValues, "%d", value)
	}
	observer := fmt.Sprintf(`
static const int32_t byte_for_expected[] = {%s};
static size_t byte_for_position = 0;
static int byte_for_failed = 0;
int32_t byte_for_record(int32_t value) {
    if (byte_for_position >= %d || byte_for_expected[byte_for_position] != value) {
        byte_for_failed = 1;
        return 0;
    }
    byte_for_position++;
    return value;
}
int32_t byte_for_finished(void) { return !byte_for_failed && byte_for_position == %d ? 1 : 0; }
`, expectedValues.String(), len(expected), len(expected))
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime+observer)
			if err != nil {
				t.Fatal(err)
			}
			native, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || native.ExitCode != 42 || native.Stdout != "" || native.Stderr != "" {
				t.Fatalf("native byte-for cleanup: %+v %v", native, err)
			}
		})
	}
}
