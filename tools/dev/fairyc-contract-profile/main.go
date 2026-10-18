// Command fairyc-contract-profile emits one backend-neutral deterministic execution proof report.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/contractprofile"
)

const (
	exitSuccess           = 0
	exitInvalidInvocation = 2
	exitSourceDiagnostics = 3
	exitInternalFailure   = 4
	exitProfileRejected   = 6
)

const usage = `Fairy C deterministic contract-profile prototype

Usage:
  fairyc-contract-profile <file.fy> [<file.fy>...]
  fairyc-contract-profile --help

The command parses one explicit source closure, consumes profile-only @max_iter attributes,
runs the ordinary semantic and verified-MIR pipeline, then emits a JSON decision to stdout.
It does not emit WASM or RISC-V bytecode.
`

func main() {
	os.Exit(run(os.Args[1:], ".", os.Stdout, os.Stderr))
}

func run(args []string, sourceRoot string, stdout, stderr io.Writer) int {
	if stdout == nil || stderr == nil {
		return exitInternalFailure
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		if _, err := io.WriteString(stdout, usage); err != nil {
			return exitInternalFailure
		}
		return exitSuccess
	}
	if len(args) == 0 {
		return invocationError(stderr, "at least one Fairy C source file is required")
	}
	if len(args) > contractprofile.MaxSourceFiles {
		return invocationError(stderr, fmt.Sprintf("at most %d source files are accepted", contractprofile.MaxSourceFiles))
	}
	for _, value := range args {
		if value == "" || strings.HasPrefix(value, "-") {
			return invocationError(stderr, fmt.Sprintf("unsupported source argument %q", value))
		}
	}

	inputs, err := loadInputs(sourceRoot, args)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	report, err := contractprofile.AnalyzeSources(inputs)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "internal contract-profile failure: %v\n", err); writeErr != nil {
			return exitInternalFailure
		}
		return exitInternalFailure
	}
	encoded, err := report.JSON()
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "encode contract-profile report: %v\n", err); writeErr != nil {
			return exitInternalFailure
		}
		return exitInternalFailure
	}
	if _, err := stdout.Write(encoded); err != nil {
		if _, writeErr := io.WriteString(stderr, "write contract-profile report failed\n"); writeErr != nil {
			return exitInternalFailure
		}
		return exitInternalFailure
	}
	switch report.Status {
	case "accepted":
		return exitSuccess
	case "source-diagnostics":
		return exitSourceDiagnostics
	default:
		return exitProfileRejected
	}
}

func loadInputs(sourceRoot string, arguments []string) ([]contractprofile.SourceInput, error) {
	if sourceRoot == "" {
		return nil, fmt.Errorf("source root is empty")
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("open source root: %w", err)
	}
	defer root.Close()

	seen := make(map[string]struct{}, len(arguments))
	inputs := make([]contractprofile.SourceInput, 0, len(arguments))
	maxBytes := int64(8 << 20)
	for _, argument := range arguments {
		if filepath.IsAbs(argument) {
			return nil, fmt.Errorf("source path %q must be relative to the source root", argument)
		}
		clean := filepath.Clean(argument)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("source path %q escapes the source root", argument)
		}
		portable := filepath.ToSlash(clean)
		if filepath.Ext(portable) != ".fy" {
			return nil, fmt.Errorf("source path %q must end in .fy", portable)
		}
		if _, duplicate := seen[portable]; duplicate {
			return nil, fmt.Errorf("source path %q is listed more than once", portable)
		}
		seen[portable] = struct{}{}

		file, err := root.Open(clean)
		if err != nil {
			return nil, fmt.Errorf("open source %q: %w", portable, err)
		}
		bytes, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read source %q: %w", portable, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close source %q: %w", portable, closeErr)
		}
		if int64(len(bytes)) > maxBytes {
			return nil, fmt.Errorf("source %q exceeds the 8 MiB input limit", portable)
		}
		inputs = append(inputs, contractprofile.SourceInput{Path: portable, Bytes: bytes})
	}
	return inputs, nil
}

func invocationError(stderr io.Writer, message string) int {
	if _, err := fmt.Fprintf(stderr, "invalid invocation: %s\n", message); err != nil {
		return exitInternalFailure
	}
	return exitInvalidInvocation
}
