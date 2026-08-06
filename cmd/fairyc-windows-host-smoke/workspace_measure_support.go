package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func observeMonotonicClockResolution() (int64, error) {
	const (
		maximumObservations      = 10_000_000
		requiredTransitions      = 32
		cooperativeYieldInterval = 4_096
	)
	var minimumPositive int64
	transitions := 0
	previous := time.Now()
	for attempts := 0; attempts < maximumObservations && transitions < requiredTransitions; attempts++ {
		if attempts != 0 && attempts%cooperativeYieldInterval == 0 {
			runtime.Gosched()
		}
		current := time.Now()
		if elapsed := current.Sub(previous).Nanoseconds(); elapsed > 0 {
			if minimumPositive == 0 || elapsed < minimumPositive {
				minimumPositive = elapsed
			}
			transitions++
		}
		previous = current
	}
	if transitions < requiredTransitions {
		return 0, fmt.Errorf("monotonic clock produced %d of %d required positive transitions during %d resolution observations", transitions, requiredTransitions, maximumObservations)
	}
	return minimumPositive, nil
}

func copyWorkspaceMeasurementFixture(sourceRoot string) (string, error) {
	root, err := os.MkdirTemp("", "fairyc-process-cache-edit-")
	if err != nil {
		return "", err
	}
	for _, name := range []string{workspaceManifestPath, workspaceMainPath, workspaceMemberManifestPath, workspaceMemberSourcePath} {
		contents, readErr := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(name)))
		if readErr != nil {
			os.RemoveAll(root)
			return "", readErr
		}
		destination := filepath.Join(root, filepath.FromSlash(name))
		if mkdirErr := os.MkdirAll(filepath.Dir(destination), 0o700); mkdirErr != nil {
			os.RemoveAll(root)
			return "", mkdirErr
		}
		if writeErr := os.WriteFile(destination, contents, 0o600); writeErr != nil {
			os.RemoveAll(root)
			return "", writeErr
		}
	}
	return root, nil
}

func mutateWorkspaceMeasurementFixture(root, condition string) error {
	path := workspaceMainPath
	needle := "\nfn i32 main()"
	replacement := "\nstruct cache_private_root { i32 value; }\n\nfn i32 main()"
	if condition == buildmeasure.CacheEditPrivateDependency || condition == buildmeasure.CacheEditPublicDependency {
		path = workspaceMemberSourcePath
		needle = "\npub fn i32 value()"
		replacement = "\nstruct cache_private_dependency { i32 value; }\n\npub fn i32 value()"
	}
	absolute := filepath.Join(root, filepath.FromSlash(path))
	contents, err := os.ReadFile(absolute)
	if err != nil {
		return err
	}
	updated := string(contents)
	switch condition {
	case buildmeasure.CacheEditPrivateRoot, buildmeasure.CacheEditPrivateDependency:
		updated = strings.Replace(updated, needle, replacement, 1)
	case buildmeasure.CacheEditPublicDependency:
		updated += "\npub fn i32 cache_public_probe() {\n    return 7;\n}\n"
	default:
		return fmt.Errorf("unknown cache edit condition %q", condition)
	}
	if updated == string(contents) {
		return fmt.Errorf("cache edit condition %q did not change its source", condition)
	}
	return os.WriteFile(absolute, []byte(updated), 0o600)
}

func runStandaloneCommand(directory, executablePath string, arguments ...string) (int, string, string, error) {
	return runStandaloneCommandWithInput(directory, executablePath, "", arguments...)
}

func runStandaloneCommandWithInput(directory, executablePath, input string, arguments ...string) (int, string, string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath, arguments...)
	command.Dir = directory
	command.Stdin = strings.NewReader(input)
	var stdout boundedBuffer
	var stderr boundedBuffer
	stdout.limit = subprocessOutputLimit
	stderr.limit = subprocessOutputLimit
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return -1, stdout.String(), stderr.String(), fmt.Errorf("command exceeded timeout")
	}
	if err == nil {
		return 0, stdout.String(), stderr.String(), nil
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return -1, stdout.String(), stderr.String(), err
	}
	return exitError.ExitCode(), stdout.String(), stderr.String(), nil
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	originalLength := len(value)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = buffer.buffer.Write(value)
	}
	if originalLength > remaining {
		buffer.truncated = true
	}
	return originalLength, nil
}

func (buffer *boundedBuffer) Len() int {
	return buffer.buffer.Len()
}

func (buffer *boundedBuffer) String() string {
	if buffer.truncated {
		return buffer.buffer.String() + "\n[output truncated]"
	}
	return buffer.buffer.String()
}
