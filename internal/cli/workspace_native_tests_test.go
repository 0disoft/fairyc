package cli

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestNativeWorkspaceTestProgramAddsVerifiedI32Harness(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	native, entry, err := nativeWorkspaceTestProgram(program, "tests::native::passes")
	if err != nil {
		t.Fatal(err)
	}
	if entry != nativeWorkspaceTestEntry || len(native.Functions) != len(program.Functions)+1 {
		t.Fatalf("native test program entry=%q functions=%d", entry, len(native.Functions))
	}
	if failures := mir.Verify(native); len(failures) != 0 {
		t.Fatalf("native test MIR failures = %v", failures)
	}
	unit, err := cbackend.Emit(native, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if unit.Entry != nativeWorkspaceTestEntry || !strings.Contains(unit.Source, "int main(void)") ||
		!strings.Contains(unit.Source, "fy_fn_0();") {
		t.Fatalf("native test C unit = entry %q, functions %v", unit.Entry, unit.Functions)
	}
}

func TestNativeWorkspaceTestsContinueAfterTrapsAndPreserveOrder(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{
		{display: "tests::native::fails", entry: "tests::native::fails"},
		{display: "tests::native::passes", entry: "tests::native::passes"},
	}
	compiled := 0
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		if entry != nativeWorkspaceTestEntry || profile != cdriver.OptimizedProfile || len(mir.Verify(program)) != 0 {
			t.Fatalf("compile native test entry=%q profile=%+v", entry, profile)
		}
		compiled++
		return cdriver.WindowsHostArtifact{Bytes: []byte{byte(compiled)}}, ExitSuccess
	}
	programArguments := []string{"--flag", "", "한글λ"}
	execute := func(artifact cdriver.WindowsHostArtifact, stdin io.Reader, arguments []string, timeout time.Duration) (cdriver.WindowsHostOutcome, error) {
		if timeout != time.Minute {
			t.Fatalf("timeout = %s", timeout)
		}
		input, err := io.ReadAll(stdin)
		if err != nil || len(input) != 0 {
			t.Fatalf("native test default input = %q, %v", input, err)
		}
		if !slices.Equal(arguments, programArguments) {
			t.Fatalf("native test program arguments = %q, want %q", arguments, programArguments)
		}
		if artifact.Bytes[0] == 1 {
			code, found := cbackend.TrapCode("assertion-failed")
			if !found {
				t.Fatal("assertion-failed trap code is missing")
			}
			return cdriver.WindowsHostOutcome{
				ExitCode:    int32(productTrapExitBase + code),
				TrapControl: string([]byte{byte(code)}),
			}, nil
		}
		return cdriver.WindowsHostOutcome{}, nil
	}
	var stderr bytes.Buffer
	outcomes, code := runNativeWorkspaceTestsWith(
		program, tests, cdriver.OptimizedProfile, nativeWorkspaceTestInput{}, programArguments, time.Minute, false, &stderr, compile, execute,
	)
	if code != ExitSuccess || stderr.String() != "" || compiled != 2 || len(outcomes) != 2 ||
		outcomes[0].display != tests[0].display || outcomes[0].trapCategory != "assertion-failed" ||
		outcomes[1].display != tests[1].display || outcomes[1].trapCategory != "" {
		t.Fatalf("native outcomes=%+v code=%d stderr=%q compiled=%d", outcomes, code, stderr.String(), compiled)
	}
}

func TestNativeWorkspaceTestForwardsCallerInputToSelectedProcess(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{{display: "tests::native::passes", entry: "tests::native::passes"}}
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		return cdriver.WindowsHostArtifact{Bytes: []byte{1}}, ExitSuccess
	}
	programArguments := []string{"--flag"}
	execute := func(_ cdriver.WindowsHostArtifact, stdin io.Reader, arguments []string, _ time.Duration) (cdriver.WindowsHostOutcome, error) {
		input, err := io.ReadAll(stdin)
		if err != nil {
			return cdriver.WindowsHostOutcome{}, err
		}
		if string(input) != "Fairy input\n" || !slices.Equal(arguments, programArguments) {
			t.Fatalf("native process input=%q arguments=%q", input, arguments)
		}
		return cdriver.WindowsHostOutcome{}, nil
	}
	var stderr bytes.Buffer
	outcomes, code := runNativeWorkspaceTestsWith(
		program,
		tests,
		cdriver.LowProfile,
		nativeWorkspaceTestInput{caller: strings.NewReader("Fairy input\n")},
		programArguments,
		time.Minute,
		false,
		&stderr,
		compile,
		execute,
	)
	if code != ExitSuccess || stderr.String() != "" || len(outcomes) != 1 || outcomes[0].display != tests[0].display {
		t.Fatalf("native input outcomes=%+v code=%d stderr=%q", outcomes, code, stderr.String())
	}
}

func TestNativeWorkspaceTestsReplayFixtureInputFromByteZero(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{
		{display: "tests::native::fails", entry: "tests::native::fails"},
		{display: "tests::native::passes", entry: "tests::native::passes"},
	}
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		return cdriver.WindowsHostArtifact{Bytes: []byte(entry)}, ExitSuccess
	}
	executions := 0
	execute := func(_ cdriver.WindowsHostArtifact, stdin io.Reader, _ []string, _ time.Duration) (cdriver.WindowsHostOutcome, error) {
		contents, err := io.ReadAll(stdin)
		if err != nil || string(contents) != "fixture\x00bytes" {
			t.Fatalf("execution %d stdin = %q, %v", executions, contents, err)
		}
		executions++
		return cdriver.WindowsHostOutcome{}, nil
	}
	var stderr bytes.Buffer
	outcomes, code := runNativeWorkspaceTestsWith(
		program, tests, cdriver.LowProfile,
		nativeWorkspaceTestInput{replay: []byte("fixture\x00bytes"), replayEach: true},
		nil, time.Minute, false, &stderr, compile, execute,
	)
	if code != ExitSuccess || stderr.String() != "" || len(outcomes) != 2 || executions != 2 {
		t.Fatalf("replayed native outcomes=%+v executions=%d code=%d stderr=%q", outcomes, executions, code, stderr.String())
	}
}

func TestNativeWorkspaceTestsCaptureExactProgramStreamsApartFromTrapControl(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{
		{display: "tests::native::fails", entry: "tests::native::fails"},
		{display: "tests::native::passes", entry: "tests::native::passes"},
	}
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		return cdriver.WindowsHostArtifact{Bytes: []byte(entry)}, ExitSuccess
	}
	executions := 0
	execute := func(cdriver.WindowsHostArtifact, io.Reader, []string, time.Duration) (cdriver.WindowsHostOutcome, error) {
		executions++
		if executions == 1 {
			code, found := cbackend.TrapCode("assertion-failed")
			if !found {
				t.Fatal("assertion-failed trap code is missing")
			}
			return cdriver.WindowsHostOutcome{
				ExitCode:    int32(productTrapExitBase + code),
				Stdout:      string([]byte{0, 0xff}),
				Stderr:      string([]byte{0x80}),
				TrapControl: string([]byte{byte(code)}),
			}, nil
		}
		return cdriver.WindowsHostOutcome{Stdout: "pass", Stderr: "warn"}, nil
	}
	var stderr bytes.Buffer
	outcomes, code := runNativeWorkspaceTestsWith(
		program, tests, cdriver.LowProfile, nativeWorkspaceTestInput{}, nil, time.Minute, true, &stderr, compile, execute,
	)
	if code != ExitSuccess || stderr.String() != "" || len(outcomes) != 2 ||
		outcomes[0].trapCategory != "assertion-failed" || outcomes[0].stdout != string([]byte{0, 0xff}) ||
		outcomes[0].stderr != string([]byte{0x80}) || outcomes[1].stdout != "pass" || outcomes[1].stderr != "warn" {
		t.Fatalf("captured native outcomes=%+v code=%d stderr=%q", outcomes, code, stderr.String())
	}
}

func TestNativeWorkspaceTestsRejectProgramOutputAndUnreceiptedExit(t *testing.T) {
	t.Parallel()
	program := nativeTestFixtureProgram()
	tests := []workspaceTest{{display: "tests::native::passes", entry: "tests::native::passes"}}
	compile := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
		return cdriver.WindowsHostArtifact{Bytes: []byte{1}}, ExitSuccess
	}
	for _, test := range []struct {
		name     string
		outcome  cdriver.WindowsHostOutcome
		wantCode int
		contains string
	}{
		{name: "stdout", outcome: cdriver.WindowsHostOutcome{Stdout: "unexpected"}, wantCode: ExitInvalidInvocation, contains: "produced program output"},
		{name: "uncontrolled exit", outcome: cdriver.WindowsHostOutcome{ExitCode: 42}, wantCode: ExitBuildFailure, contains: "without valid trap control"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			outcomes, code := runNativeWorkspaceTestsWith(
				program,
				tests,
				cdriver.LowProfile,
				nativeWorkspaceTestInput{},
				nil,
				time.Minute,
				false,
				&stderr,
				compile,
				func(cdriver.WindowsHostArtifact, io.Reader, []string, time.Duration) (cdriver.WindowsHostOutcome, error) {
					return test.outcome, nil
				},
			)
			if outcomes != nil || code != test.wantCode || !strings.Contains(stderr.String(), test.contains) {
				t.Fatalf("native failure outcomes=%+v code=%d stderr=%q", outcomes, code, stderr.String())
			}
		})
	}
}

func nativeTestFixtureProgram() mir.Program {
	makeTest := func(symbol hir.SymbolID, name string) mir.Function {
		return mir.Function{
			Symbol: symbol, Name: name, ResultType: hir.TypeVoid, Entry: 0,
			Blocks: []mir.Block{{
				ID: 0,
				Terminator: mir.Terminator{
					Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock, DestinationLocal: mir.NoLocal,
				},
			}},
		}
	}
	return mir.Program{Functions: []mir.Function{
		makeTest(1, "tests::native::passes"),
		makeTest(2, "tests::native::fails"),
	}}
}
