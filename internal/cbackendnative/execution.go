package cbackendnative

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func execute(caseID, executablePath string, expectedExit int, expectedDrops []uint32) error {
	return executeInDirectory(caseID, executablePath, "", expectedExit, expectedDrops)
}

func executeInDirectory(caseID, executablePath, workingDirectory string, expectedExit int, expectedDrops []uint32) error {
	result, err := executeProcessInDirectory(executablePath, workingDirectory, nil)
	if err != nil {
		return err
	}
	if len(result.stdout) != 0 {
		return fmt.Errorf("native executable produced unexpected stdout %q", string(result.stdout))
	}
	actualDrops, err := decodeDropTrace(result.stderr)
	if err != nil {
		return fmt.Errorf("native drop trace for %s: %w", caseID, err)
	}
	if !slices.Equal(actualDrops, expectedDrops) {
		return fmt.Errorf("native drop trace for %s: got %v, want %v", caseID, actualDrops, expectedDrops)
	}
	if result.exitCode != expectedExit {
		return fmt.Errorf("native exit for %s: got %d, want %d", caseID, result.exitCode, expectedExit)
	}
	return nil
}

func executeSanitized(caseID, executablePath string, expectedExit int) error {
	result, err := executeProcess(executablePath, sanitizerEnvironment(os.Environ()))
	if err != nil {
		return err
	}
	if len(result.stdout) != 0 {
		return fmt.Errorf("sanitized executable produced unexpected stdout %q", string(result.stdout))
	}
	if len(result.stderr) != 0 {
		diagnostic := strings.ReplaceAll(strings.TrimSpace(string(result.stderr)), filepath.Dir(executablePath), "<temporary>")
		return fmt.Errorf("sanitizer reported for %s: %s", caseID, diagnostic)
	}
	if result.exitCode != expectedExit {
		return fmt.Errorf("sanitized exit for %s: got %d, want %d", caseID, result.exitCode, expectedExit)
	}
	return nil
}

type processResult struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func executeProcess(executablePath string, environment []string) (processResult, error) {
	return executeProcessInDirectory(executablePath, "", environment)
}

func executeProcessInDirectory(executablePath, workingDirectory string, environment []string) (processResult, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	command.Stdin = nil
	if workingDirectory != "" {
		command.Dir = workingDirectory
	}
	if environment != nil {
		command.Env = environment
	}
	stdout := boundedBuffer{limit: maxProcessOutput}
	stderr := boundedBuffer{limit: maxProcessOutput}
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return processResult{}, fmt.Errorf("native executable exceeded %s", executionTimeout)
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return processResult{}, fmt.Errorf("execute native program: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	if stdout.truncated || stderr.truncated || len(stdout.bytes)+len(stderr.bytes) > maxProcessOutput {
		return processResult{}, fmt.Errorf("native executable exceeded %d output bytes", maxProcessOutput)
	}
	return processResult{exitCode: exitCode, stdout: bytes.Clone(stdout.bytes), stderr: bytes.Clone(stderr.bytes)}, nil
}

func sanitizerEnvironment(base []string) []string {
	result := make([]string, 0, len(base)+3)
	for _, entry := range base {
		name, _, _ := strings.Cut(entry, "=")
		if name == "ASAN_OPTIONS" || name == "LSAN_OPTIONS" || name == "UBSAN_OPTIONS" {
			continue
		}
		result = append(result, entry)
	}
	return append(
		result,
		"ASAN_OPTIONS=abort_on_error=1:color=never:detect_leaks=1:halt_on_error=1:symbolize=0",
		"LSAN_OPTIONS=exitcode=23:report_objects=0",
		"UBSAN_OPTIONS=color=never:halt_on_error=1:print_stacktrace=0",
	)
}
