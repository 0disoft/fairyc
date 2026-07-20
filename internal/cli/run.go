// Package cli owns the experimental Stage 0 command-line boundary.
package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	ExitSuccess           = 0
	ExitInvalidInvocation = 2
	ExitSourceDiagnostics = 3
	ExitInternalFailure   = 4
	ExitProgramTrap       = 5
	ExitBuildFailure      = 6
	maxSemanticInputFiles = 64
)

const usage = `Fairy C Stage 0 compiler prototype

Usage:
  fairyc explain <diagnostic-code>
  fairyc targets
  fairyc experimental parse <file.fy>
  fairyc experimental check-syntax <file.fy>
  fairyc experimental check-semantics <file.fy> [<file.fy>...]
  fairyc experimental target-spec <target-id>
  fairyc experimental emit-c <file.fy>
  fairyc experimental build <file.fy> --cc <clang|gcc> --profile <low|optimized> --output <build/path>
  fairyc experimental build-target <file.fy> --target x86_64-windows-msvc --profile <low|optimized> --output <build/path.exe>
  fairyc experimental run-target <file.fy> --target x86_64-windows-msvc --profile <low|optimized>
  fairyc experimental build-host <file.fy> --profile <low|optimized> --output <build/path.exe>
  fairyc experimental run-host <file.fy> --profile <low|optimized>
  fairyc experimental run <file.fy>
  fairyc --help

The experimental commands are unstable. The reserved "fairyc check" command is not implemented
until effect and later semantic analysis groups exist.
`

// Run executes one bounded command without reading global process arguments or exiting the process.
func Run(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	if stdout == nil || stderr == nil {
		return ExitInternalFailure
	}
	if len(args) == 1 && isHelp(args[0]) {
		return writeHelp(stdout, stderr)
	}
	if len(args) == 0 {
		return invocationError(stderr, "a command is required")
	}
	if args[0] == "explain" {
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			return writeHelp(stdout, stderr)
		}
		return runExplain(args[1:], stdout, stderr)
	}
	if args[0] == "targets" {
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			return writeHelp(stdout, stderr)
		}
		return runTargets(args[1:], stdout, stderr)
	}
	if args[0] == "check" {
		return invocationError(stderr, `"check" is reserved for the full semantic pipeline; use "experimental check-syntax" for parser-only validation`)
	}
	if args[0] != "experimental" {
		return invocationError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
	return runExperimental(args[1:], sourceRoot, stdout, stderr)
}

func runExplain(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return invocationError(stderr, `command "explain" expects exactly one diagnostic code`)
	}
	code := args[0]
	if !validDiagnosticCode(code) {
		return invocationError(stderr, fmt.Sprintf("diagnostic code %q is not in canonical E0001, A2001, or V1001 form", code))
	}
	entry, found := diagnostic.Lookup(code)
	if !found {
		return invocationError(stderr, fmt.Sprintf("unknown diagnostic code %q", code))
	}
	if _, err := fmt.Fprintf(
		stdout,
		"%s: %s\nCategory: %s\nCatalog: %s (provisional Stage 0)\n",
		entry.Code,
		entry.Meaning,
		entry.Category,
		diagnostic.CatalogVersion,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func validDiagnosticCode(code string) bool {
	if len(code) != 5 || (code[0] != 'E' && code[0] != 'A' && code[0] != 'V') {
		return false
	}
	for _, value := range code[1:] {
		if value < '0' || value > '9' {
			return false
		}
	}
	return true
}

func runExperimental(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	if len(args) == 1 && isHelp(args[0]) {
		return writeHelp(stdout, stderr)
	}
	if len(args) == 0 {
		return invocationError(stderr, "an experimental command is required")
	}
	command := args[0]
	if len(args) == 2 && isHelp(args[1]) {
		return writeHelp(stdout, stderr)
	}
	switch command {
	case "parse":
		if len(args) != 2 {
			return invocationError(stderr, fmt.Sprintf("experimental command %q expects exactly one source file", command))
		}
		return runParse(args[1], sourceRoot, stdout, stderr)
	case "check-syntax":
		if len(args) != 2 {
			return invocationError(stderr, fmt.Sprintf("experimental command %q expects exactly one source file", command))
		}
		return runCheckSyntax(args[1], sourceRoot, stdout, stderr)
	case "check-semantics":
		if len(args) < 2 {
			return invocationError(stderr, "experimental command \"check-semantics\" expects one or more source files")
		}
		return runCheckSemantics(args[1:], sourceRoot, stdout, stderr)
	case "run":
		if len(args) != 2 {
			return invocationError(stderr, "experimental command \"run\" expects exactly one source file")
		}
		return runInterpreter(args[1], sourceRoot, stdout, stderr)
	case "target-spec":
		return runTargetSpec(args[1:], stdout, stderr)
	case "emit-c":
		if len(args) != 2 {
			return invocationError(stderr, "experimental command \"emit-c\" expects exactly one source file")
		}
		return runEmitC(args[1], sourceRoot, stdout, stderr)
	case "build":
		return runBuild(args[1:], sourceRoot, stdout, stderr)
	case "build-target":
		return runTargetBuild(args[1:], sourceRoot, stdout, stderr)
	case "run-target":
		return runTarget(args[1:], sourceRoot, stdout, stderr)
	case "build-host":
		return runHostBuild(args[1:], sourceRoot, stdout, stderr)
	case "run-host":
		return runHost(args[1:], sourceRoot, stdout, stderr)
	default:
		return invocationError(stderr, fmt.Sprintf("unknown experimental command %q", command))
	}
}

func runInterpreter(argument, sourceRoot string, stdout, stderr io.Writer) int {
	lowered, code := lowerSingleProgram(argument, sourceRoot, stderr, "interpreter")
	if code != ExitSuccess {
		return code
	}
	outcome, err := interpreter.Execute(lowered.program, lowered.entryName)
	if err != nil {
		var capability interpreter.CapabilityError
		if errors.As(err, &capability) {
			return subsetCapabilityError(stderr, "interpreter", capability)
		}
		return internalError(stderr)
	}
	if outcome.Kind == interpreter.OutcomeTrap {
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", outcome.TrapCategory); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if outcome.Kind != interpreter.OutcomeReturn {
		return internalError(stderr)
	}
	if _, err := fmt.Fprintf(stdout, "result: %d\n", outcome.Value); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func runEmitC(argument, sourceRoot string, stdout, stderr io.Writer) int {
	lowered, code := lowerSingleProgram(argument, sourceRoot, stderr, "C-emission")
	if code != ExitSuccess {
		return code
	}
	unit, err := cbackend.Emit(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return subsetCapabilityError(stderr, "C-emission", capability)
		}
		return internalError(stderr)
	}
	if _, err := io.WriteString(stdout, unit.Source); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

type loweredProgram struct {
	program   mir.Program
	entryName string
}

func lowerSingleProgram(argument, sourceRoot string, stderr io.Writer, subset string) (loweredProgram, int) {
	limits := source.DefaultLimits()
	loaded, err := loadSource(sourceRoot, argument, limits.MaxBytes)
	if err != nil {
		var tooLarge sourceTooLargeError
		if errors.As(err, &tooLarge) {
			value := source.ExhaustionDiagnostic(0)
			if writeErr := writeDiagnostics(stderr, tooLarge.relativePath, nil, []diagnostic.Diagnostic{value}); writeErr != nil {
				return loweredProgram{}, ExitInternalFailure
			}
			return loweredProgram{}, ExitSourceDiagnostics
		}
		return loweredProgram{}, inputError(stderr, err)
	}
	frontendResult, err := frontend.AnalyzeSyntax(loaded.file, limits)
	if err != nil {
		return loweredProgram{}, internalError(stderr)
	}
	if len(frontendResult.Tree.Diagnostics) > 0 {
		if err := writeDiagnostics(stderr, loaded.file.RelativePath, frontendResult.Tree.Source, frontendResult.Tree.Diagnostics); err != nil {
			return loweredProgram{}, ExitInternalFailure
		}
		return loweredProgram{}, ExitSourceDiagnostics
	}
	semanticResult := semantic.Analyze([]semantic.File{{Path: loaded.file.RelativePath, Module: frontendResult.AST}})
	if len(semanticResult.Diagnostics) > 0 {
		if err := writeSemanticDiagnostics(stderr, map[string][]byte{loaded.file.RelativePath: frontendResult.Tree.Source}, semanticResult.Diagnostics); err != nil {
			return loweredProgram{}, ExitInternalFailure
		}
		return loweredProgram{}, ExitSourceDiagnostics
	}
	if len(semanticResult.Unsupported) > 0 {
		if err := writeUnsupportedSemantics(stderr, semanticResult.Unsupported); err != nil {
			return loweredProgram{}, ExitInternalFailure
		}
		return loweredProgram{}, ExitInvalidInvocation
	}
	program, err := mir.Lower(semanticResult.HIR)
	if err != nil {
		return loweredProgram{}, subsetCapabilityError(stderr, subset, err)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return loweredProgram{}, internalError(stderr)
	}
	entryName := frontendResult.AST.Path + "::main"
	entry, found := programEntry(program, entryName)
	if !found || len(entry.Parameters) != 0 || entry.ResultType != hir.TypeI32 {
		return loweredProgram{}, subsetCapabilityError(stderr, subset, fmt.Errorf("entry %s must have type fn() -> i32", entryName))
	}
	return loweredProgram{program: program, entryName: entryName}, ExitSuccess
}

func programEntry(program mir.Program, name string) (mir.Function, bool) {
	for _, function := range program.Functions {
		if function.Name == name {
			return function, true
		}
	}
	return mir.Function{}, false
}

func subsetCapabilityError(stderr io.Writer, subset string, err error) int {
	message := fmt.Sprintf("requested program is outside the experimental %s subset", subset)
	if err != nil {
		message += ": " + strings.TrimSpace(err.Error())
	}
	if _, writeErr := fmt.Fprintf(stderr, "fairyc: %s\n", message); writeErr != nil {
		return ExitInternalFailure
	}
	return ExitInvalidInvocation
}

func runCheckSemantics(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	limits := source.DefaultLimits()
	if len(arguments) > maxSemanticInputFiles {
		return invocationError(stderr, fmt.Sprintf("experimental command \"check-semantics\" accepts at most %d source files", maxSemanticInputFiles))
	}
	aggregateLimit := limits.MaxBytes * 4
	aggregateBytes := 0
	loadedFiles := make([]loadedSource, 0, len(arguments))
	seen := make(map[string]struct{}, len(arguments))
	for _, argument := range arguments {
		loaded, err := loadSource(sourceRoot, argument, limits.MaxBytes)
		if err != nil {
			var tooLarge sourceTooLargeError
			if errors.As(err, &tooLarge) {
				value := source.ExhaustionDiagnostic(0)
				if writeErr := writeDiagnostics(stderr, tooLarge.relativePath, nil, []diagnostic.Diagnostic{value}); writeErr != nil {
					return ExitInternalFailure
				}
				return ExitSourceDiagnostics
			}
			return inputError(stderr, err)
		}
		if _, duplicate := seen[loaded.file.RelativePath]; duplicate {
			return invocationError(stderr, fmt.Sprintf("semantic input %q is listed more than once", safeDisplayPath(loaded.file.RelativePath)))
		}
		seen[loaded.file.RelativePath] = struct{}{}
		aggregateBytes += len(loaded.file.Bytes)
		if aggregateBytes > aggregateLimit {
			return invocationError(stderr, "semantic input set exceeds the experimental aggregate byte limit")
		}
		loadedFiles = append(loadedFiles, loaded)
	}

	semanticFiles := make([]semantic.File, 0, len(loadedFiles))
	sources := make(map[string][]byte, len(loadedFiles))
	syntaxFailed := false
	for _, loaded := range loadedFiles {
		frontendResult, err := frontend.AnalyzeSyntax(loaded.file, limits)
		if err != nil {
			return internalError(stderr)
		}
		sources[loaded.file.RelativePath] = frontendResult.Tree.Source
		if len(frontendResult.Tree.Diagnostics) > 0 {
			syntaxFailed = true
			if err := writeDiagnostics(stderr, loaded.file.RelativePath, frontendResult.Tree.Source, frontendResult.Tree.Diagnostics); err != nil {
				return ExitInternalFailure
			}
			continue
		}
		semanticFiles = append(semanticFiles, semantic.File{Path: loaded.file.RelativePath, Module: frontendResult.AST})
	}
	if syntaxFailed {
		return ExitSourceDiagnostics
	}
	result := semantic.Analyze(semanticFiles)
	if len(result.Diagnostics) > 0 {
		if err := writeSemanticDiagnostics(stderr, sources, result.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(result.Unsupported) > 0 {
		if err := writeUnsupportedSemantics(stderr, result.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitInvalidInvocation
	}
	functions := 0
	for _, module := range result.HIR.Modules {
		functions += len(module.Functions)
	}
	if len(loadedFiles) == 1 {
		if _, err := fmt.Fprintf(stdout, "semantics ok: %s (typed HIR: %d function(s))\n", safeDisplayPath(loadedFiles[0].file.RelativePath), functions); err != nil {
			return outputError(stderr)
		}
		return ExitSuccess
	}
	if _, err := fmt.Fprintf(stdout, "semantics ok: %d file(s), %d module(s) (typed HIR: %d function(s))\n", len(loadedFiles), len(result.HIR.Modules), functions); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func runParse(argument, sourceRoot string, stdout, stderr io.Writer) int {
	limits := source.DefaultLimits()
	loaded, err := loadSource(sourceRoot, argument, limits.MaxBytes)
	if err != nil {
		var tooLarge sourceTooLargeError
		if errors.As(err, &tooLarge) {
			if writeErr := writeParseReport(stdout, buildTooLargeReport(tooLarge.relativePath, tooLarge.limit)); writeErr != nil {
				return outputError(stderr)
			}
			return ExitSourceDiagnostics
		}
		return inputError(stderr, err)
	}
	result, err := frontend.AnalyzeSyntax(loaded.file, limits)
	if err != nil {
		return internalError(stderr)
	}
	report := buildParseReport(loaded.file.RelativePath, result.Tree, result.AST != nil)
	if err := writeParseReport(stdout, report); err != nil {
		return outputError(stderr)
	}
	if len(result.Tree.Diagnostics) > 0 {
		return ExitSourceDiagnostics
	}
	return ExitSuccess
}

func runCheckSyntax(argument, sourceRoot string, stdout, stderr io.Writer) int {
	limits := source.DefaultLimits()
	loaded, err := loadSource(sourceRoot, argument, limits.MaxBytes)
	if err != nil {
		var tooLarge sourceTooLargeError
		if errors.As(err, &tooLarge) {
			value := source.ExhaustionDiagnostic(0)
			if writeErr := writeDiagnostics(stderr, tooLarge.relativePath, nil, []diagnostic.Diagnostic{value}); writeErr != nil {
				return ExitInternalFailure
			}
			return ExitSourceDiagnostics
		}
		return inputError(stderr, err)
	}
	result, err := frontend.AnalyzeSyntax(loaded.file, limits)
	if err != nil {
		return internalError(stderr)
	}
	if len(result.Tree.Diagnostics) > 0 {
		if err := writeDiagnostics(stderr, loaded.file.RelativePath, result.Tree.Source, result.Tree.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if _, err := fmt.Fprintf(stdout, "syntax ok: %s\n", safeDisplayPath(loaded.file.RelativePath)); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func isHelp(value string) bool {
	return value == "-h" || value == "--help" || value == "help"
}

func writeHelp(stdout, stderr io.Writer) int {
	if _, err := io.WriteString(stdout, usage); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func invocationError(stderr io.Writer, message string) int {
	if _, err := fmt.Fprintf(stderr, "fairyc: %s\n\n%s", message, usage); err != nil {
		return ExitInternalFailure
	}
	return ExitInvalidInvocation
}

func inputError(stderr io.Writer, err error) int {
	message := "cannot load input"
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	if _, writeErr := fmt.Fprintf(stderr, "fairyc: %s\n", message); writeErr != nil {
		return ExitInternalFailure
	}
	return ExitInvalidInvocation
}

func internalError(stderr io.Writer) int {
	if _, err := io.WriteString(stderr, "fairyc: internal compiler invariant failed\n"); err != nil {
		return ExitInternalFailure
	}
	return ExitInternalFailure
}

func outputError(stderr io.Writer) int {
	if stderr != nil {
		_, _ = io.WriteString(stderr, "fairyc: write output failed\n")
	}
	return ExitInternalFailure
}
