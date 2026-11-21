package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

const nativeWorkspaceTestEntry = "$fairyc::native-test-harness::main"

type nativeWorkspaceTestPrepare func(
	program mir.Program,
	entry string,
	profile cdriver.Profile,
	stderr io.Writer,
) (nativeWorkspaceTestBuild, int)

type nativeWorkspaceTestBuild func(io.Writer) (nativeWorkspaceTestArtifact, int)

type nativeWorkspaceTestArtifact = nativeCompilation

type nativeWorkspaceTestOutcome = cdriver.NativeOutcome

type nativeWorkspaceTestExecutor func(nativeWorkspaceTestArtifact, io.Reader, []string, time.Duration) (nativeWorkspaceTestOutcome, error)

type nativeWorkspaceTestInput struct {
	caller     io.Reader
	replay     []byte
	replayEach bool
}

func (input nativeWorkspaceTestInput) reader() io.Reader {
	if input.replayEach {
		return bytes.NewReader(input.replay)
	}
	if input.caller != nil {
		return input.caller
	}
	return strings.NewReader("")
}

func runNativeWorkspaceTests(
	program mir.Program,
	tests []workspaceTest,
	profile cdriver.Profile,
	input nativeWorkspaceTestInput,
	programArguments []string,
	timeout time.Duration,
	captureOutput bool,
	sdkStorage bool,
	jobs int,
	stderr io.Writer,
) ([]workspaceTestOutcome, int) {
	prepare := func(program mir.Program, entry string, profile cdriver.Profile, stderr io.Writer) (nativeWorkspaceTestBuild, int) {
		return prepareNativeWorkspaceTest(program, entry, profile, sdkStorage, stderr)
	}
	return runNativeWorkspaceTestsWith(
		program,
		tests,
		profile,
		input,
		programArguments,
		timeout,
		captureOutput,
		sdkStorage,
		jobs,
		stderr,
		prepare,
		executeNativeWorkspaceTest,
	)
}

func runNativeWorkspaceTestsWith(
	program mir.Program,
	tests []workspaceTest,
	profile cdriver.Profile,
	input nativeWorkspaceTestInput,
	programArguments []string,
	timeout time.Duration,
	captureOutput bool,
	sdkStorage bool,
	jobs int,
	stderr io.Writer,
	prepare nativeWorkspaceTestPrepare,
	execute nativeWorkspaceTestExecutor,
) ([]workspaceTestOutcome, int) {
	outcomes := make([]workspaceTestOutcome, 0, len(tests))
	// Each slot owns one build until its result is consumed. In particular,
	// wait for speculative builds to finish their scratch cleanup on failure.
	var pending [maxNativeTestJobs]<-chan nativeWorkspaceTestBuildResult
	defer func() {
		for _, done := range pending {
			if done != nil {
				<-done
			}
		}
	}()
	start := func(test workspaceTest) <-chan nativeWorkspaceTestBuildResult {
		return startNativeWorkspaceTestBuild(program, test, profile, sdkStorage, jobs > 1, prepare)
	}
	for index := range min(jobs, len(tests)) {
		pending[index] = start(tests[index])
	}
	for index, test := range tests {
		slot := index % jobs
		result := <-pending[slot]
		pending[slot] = nil
		if len(result.diagnostics) != 0 {
			_, _ = stderr.Write(result.diagnostics)
		}
		if result.code != ExitSuccess {
			return nil, result.code
		}
		outcome, err := execute(result.artifact, input.reader(), programArguments, timeout)
		if err != nil {
			return nil, buildFailure(stderr, fmt.Errorf("execute native test %s: %w", test.display, err))
		}
		programStderr, category := outcome.Stderr, outcome.TrapCategory
		if !captureOutput && (outcome.Stdout != "" || programStderr != "") {
			return nil, workspaceSubsetCapabilityError(
				workspaceCompilePolicy{stableDiagnostics: true},
				stderr,
				"native test",
				fmt.Errorf("%s produced program output", test.display),
			)
		}
		if category == "" && outcome.ExitCode != 0 {
			return nil, buildFailure(
				stderr,
				fmt.Errorf("native test %s exited with status %d without valid trap control", test.display, outcome.ExitCode),
			)
		}
		outcomes = append(outcomes, workspaceTestOutcome{
			display: test.display, trapCategory: category, source: test.source,
			stdout: outcome.Stdout, stderr: programStderr,
		})
		if next := index + jobs; next < len(tests) {
			pending[slot] = start(tests[next])
		}
	}
	return outcomes, ExitSuccess
}

type nativeWorkspaceTestBuildResult struct {
	artifact    nativeWorkspaceTestArtifact
	code        int
	diagnostics []byte
}

func startNativeWorkspaceTestBuild(program mir.Program, test workspaceTest, profile cdriver.Profile, sdkStorage, asynchronous bool, prepare nativeWorkspaceTestPrepare) <-chan nativeWorkspaceTestBuildResult {
	done := make(chan nativeWorkspaceTestBuildResult, 1)
	var diagnostics bytes.Buffer
	testProgram, entry, err := nativeWorkspaceTestProgram(program, test.entry, sdkStorage)
	if err != nil {
		code := internalError(&diagnostics)
		done <- nativeWorkspaceTestBuildResult{code: code, diagnostics: diagnostics.Bytes()}
		return done
	}
	build, code := prepare(testProgram, entry, profile, &diagnostics)
	if code != ExitSuccess {
		done <- nativeWorkspaceTestBuildResult{code: code, diagnostics: diagnostics.Bytes()}
		return done
	}
	run := func() {
		artifact, code := build(&diagnostics)
		done <- nativeWorkspaceTestBuildResult{artifact: artifact, code: code, diagnostics: diagnostics.Bytes()}
	}
	if asynchronous {
		go run()
	} else {
		run()
	}
	return done
}

func prepareNativeWorkspaceTest(program mir.Program, entry string, profile cdriver.Profile, sdkStorage bool, stderr io.Writer) (nativeWorkspaceTestBuild, int) {
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		return nil, buildFailure(stderr, err)
	}
	plan, code := prepareNativeProgram(program, entry, workspaceTargetOptions{targetID: id, profile: profile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, sdkStorage, stderr)
	if code != ExitSuccess {
		return nil, code
	}
	return plan.compile, ExitSuccess
}

func executeNativeWorkspaceTest(artifact nativeWorkspaceTestArtifact, stdin io.Reader, programArguments []string, timeout time.Duration) (nativeWorkspaceTestOutcome, error) {
	return artifact.toolchain.Execute(artifact.artifact, cdriver.NativeExecutionOptions{Input: stdin, Arguments: programArguments, Timeout: timeout})
}

func nativeWorkspaceTarget() target.ID {
	id, _ := cdriver.HostNativeTarget()
	return id
}

func nativeWorkspaceTestProgram(input mir.Program, testEntry string, sdkStorage bool) (mir.Program, string, error) {
	test, found := programEntry(input, testEntry)
	if !found || len(test.Parameters) != 0 || test.ResultType != hir.TypeVoid {
		return mir.Program{}, "", fmt.Errorf("native test entry %q must have type fn() -> void", testEntry)
	}
	maximumSymbol := hir.SymbolID(0)
	for _, function := range input.Functions {
		if function.Name == nativeWorkspaceTestEntry {
			return mir.Program{}, "", fmt.Errorf("native test harness name collides with MIR function")
		}
		maximumSymbol = max(maximumSymbol, function.Symbol)
	}
	for _, foreign := range input.ForeignFunctions {
		maximumSymbol = max(maximumSymbol, foreign.Symbol)
	}
	if maximumSymbol == hir.SymbolID(int(^uint(0)>>1)) {
		return mir.Program{}, "", fmt.Errorf("native test harness symbol space is exhausted")
	}
	harnessSymbol := maximumSymbol + 1
	// Verification and emission read the existing MIR. Own the function table
	// so each harness can append its entry without changing the checked program.
	result := input
	result.Functions = make([]mir.Function, len(input.Functions), len(input.Functions)+1)
	copy(result.Functions, input.Functions)
	result.Functions = append(result.Functions, mir.Function{
		Symbol:       harnessSymbol,
		Name:         nativeWorkspaceTestEntry,
		Span:         test.Span,
		ResultType:   hir.TypeI32,
		Effects:      append([]hir.Effect(nil), test.Effects...),
		Entry:        0,
		ReturnEntity: "terminator:native-test-return",
		Entities: []string{
			"block:native-test-call",
			"block:native-test-return",
			"terminator:native-test-call",
			"terminator:native-test-return",
		},
		Blocks: []mir.Block{
			{
				ID:     0,
				Entity: "block:native-test-call",
				Terminator: mir.Terminator{
					Kind:             mir.TerminatorCall,
					Entity:           "terminator:native-test-call",
					Span:             test.Span,
					Successors:       []mir.BlockID{1},
					UnwindSuccessor:  mir.NoBlock,
					Callee:           test.Symbol,
					DestinationLocal: mir.NoLocal,
				},
			},
			{
				ID:     1,
				Entity: "block:native-test-return",
				Terminator: mir.Terminator{
					Kind:             mir.TerminatorReturn,
					Entity:           "terminator:native-test-return",
					Span:             test.Span,
					UnwindSuccessor:  mir.NoBlock,
					DestinationLocal: mir.NoLocal,
					ReturnValue:      mir.Operand{Kind: mir.OperandI32, I32: 0},
				},
			},
		},
	})
	verify := mir.Verify
	if sdkStorage {
		verify = mir.VerifyStorageTransfers
	}
	if failures := verify(result); len(failures) != 0 {
		return mir.Program{}, "", fmt.Errorf("native test harness failed MIR verification: %v", failures[0])
	}
	return result, nativeWorkspaceTestEntry, nil
}
