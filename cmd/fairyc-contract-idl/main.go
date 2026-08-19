// Command fairyc-contract-idl validates the experimental contract-interface profile and emits
// deterministic, chain-neutral IDL or typed client bindings.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/contractabi"
)

const (
	exitSuccess           = 0
	exitInvalidInvocation = 2
	exitSourceDiagnostics = 3
	exitInternalFailure   = 4
)

const usage = `Fairy C experimental contract IDL generator

Usage:
  fairyc-contract-idl [--format <json|typescript|go>] [--package <go-package>] <interface.fy>

The JSON IDL is chain-neutral. Generated clients call a ContractTransport supplied by the consumer;
that transport owns the concrete chain codec, address format, signing, submission, and decoding.
Output is written to stdout.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if stdout == nil || stderr == nil {
		return exitInternalFailure
	}
	flags := flag.NewFlagSet("fairyc-contract-idl", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		_, _ = io.WriteString(stderr, usage)
	}
	formatName := flags.String("format", "json", "output format: json, typescript, or go")
	packageName := flags.String("package", "contractbindings", "package name used by generated Go bindings")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitSuccess
		}
		return exitInvalidInvocation
	}
	if flags.NArg() != 1 {
		return invocationError(stderr, "exactly one contract-interface source file is required")
	}
	inputPath := flags.Arg(0)
	if strings.ToLower(filepath.Ext(inputPath)) != ".fy" {
		return invocationError(stderr, "contract-interface source must end in .fy")
	}

	format := canonicalFormat(*formatName)
	if format == "" {
		return invocationError(stderr, fmt.Sprintf("unknown output format %q", *formatName))
	}
	if format == "go" {
		if err := contractabi.ValidateGoPackageName(*packageName); err != nil {
			return invocationError(stderr, err.Error())
		}
	}

	sourceBytes, err := readBoundedFile(inputPath)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "fairyc-contract-idl: %v\n", err); writeErr != nil {
			return exitInternalFailure
		}
		return exitInvalidInvocation
	}
	document, diagnostics := contractabi.Parse(sourceBytes)
	if len(diagnostics) != 0 {
		for _, diagnostic := range diagnostics {
			if _, err := fmt.Fprintf(
				stderr,
				"%s:%d:%d: %s: %s\n",
				inputPath,
				diagnostic.Line,
				diagnostic.Column,
				diagnostic.Code,
				diagnostic.Message,
			); err != nil {
				return exitInternalFailure
			}
		}
		return exitSourceDiagnostics
	}

	var output []byte
	switch format {
	case "json":
		output, err = contractabi.RenderJSON(document)
	case "typescript":
		output, err = contractabi.RenderTypeScript(document)
	case "go":
		output, err = contractabi.RenderGo(document, *packageName)
	default:
		return exitInternalFailure
	}
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "%s:1:1: CIDL1101: generate %s output: %v\n", inputPath, format, err); writeErr != nil {
			return exitInternalFailure
		}
		return exitSourceDiagnostics
	}
	if _, err := stdout.Write(output); err != nil {
		return exitInternalFailure
	}
	return exitSuccess
}

func canonicalFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "json", "idl":
		return "json"
	case "typescript", "ts":
		return "typescript"
	case "go":
		return "go"
	default:
		return ""
	}
}

func readBoundedFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open input: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect input: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input is not a regular file")
	}
	if info.Size() > contractabi.MaxSourceBytes {
		return nil, fmt.Errorf("input exceeds %d bytes", contractabi.MaxSourceBytes)
	}
	contents, err := io.ReadAll(io.LimitReader(file, contractabi.MaxSourceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	if len(contents) > contractabi.MaxSourceBytes {
		return nil, fmt.Errorf("input exceeds %d bytes", contractabi.MaxSourceBytes)
	}
	return contents, nil
}

func invocationError(stderr io.Writer, message string) int {
	if _, err := fmt.Fprintf(stderr, "fairyc-contract-idl: %s\n", message); err != nil {
		return exitInternalFailure
	}
	return exitInvalidInvocation
}
