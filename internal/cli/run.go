// Package cli owns the experimental Stage 0 command-line boundary.
package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const (
	ExitSuccess           = 0
	ExitInvalidInvocation = 2
	ExitSourceDiagnostics = 3
	ExitInternalFailure   = 4
)

const usage = `Fairy C Stage 0 frontend prototype

Usage:
  fairyc experimental parse <file.fy>
  fairyc experimental check-syntax <file.fy>
  fairyc experimental check-semantics <file.fy>
  fairyc --help

The experimental commands are unstable. The reserved "fairyc check" command is not implemented
until resource and effect analysis exist.
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
	if args[0] == "check" {
		return invocationError(stderr, `"check" is reserved for the full semantic pipeline; use "experimental check-syntax" for parser-only validation`)
	}
	if args[0] != "experimental" {
		return invocationError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
	return runExperimental(args[1:], sourceRoot, stdout, stderr)
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
	if len(args) != 2 {
		return invocationError(stderr, fmt.Sprintf("experimental command %q expects exactly one source file", command))
	}
	switch command {
	case "parse":
		return runParse(args[1], sourceRoot, stdout, stderr)
	case "check-syntax":
		return runCheckSyntax(args[1], sourceRoot, stdout, stderr)
	case "check-semantics":
		return runCheckSemantics(args[1], sourceRoot, stdout, stderr)
	default:
		return invocationError(stderr, fmt.Sprintf("unknown experimental command %q", command))
	}
}

func runCheckSemantics(argument, sourceRoot string, stdout, stderr io.Writer) int {
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
	frontendResult, err := frontend.AnalyzeSyntax(loaded.file, limits)
	if err != nil {
		return internalError(stderr)
	}
	if len(frontendResult.Tree.Diagnostics) > 0 {
		if err := writeDiagnostics(stderr, loaded.file.RelativePath, frontendResult.Tree.Source, frontendResult.Tree.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	result := semantic.Analyze([]semantic.File{{Path: loaded.file.RelativePath, Module: frontendResult.AST}})
	if len(result.Diagnostics) > 0 {
		sources := map[string][]byte{loaded.file.RelativePath: frontendResult.Tree.Source}
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
	if _, err := fmt.Fprintf(stdout, "semantics ok: %s (typed HIR: %d function(s))\n", safeDisplayPath(loaded.file.RelativePath), functions); err != nil {
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
	if _, err := io.WriteString(stderr, "fairyc: internal frontend invariant failed\n"); err != nil {
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
