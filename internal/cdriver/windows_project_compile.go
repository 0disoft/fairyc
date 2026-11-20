package cdriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Windows has no /proc memory admission. Without a measured memory budget, admit
// at most two CL processes, each with the existing <=2MiB literal/body source
// and shared <=8MiB header. This is a concurrency/source bound, not a CL RSS
// bound. The separately admitted <=8MiB runtime always compiles alone.
func windowsProjectCompileWorkers(cpus int) int { return max(1, min(2, cpus)) }

// The coordinator owns all scheduling/error state. Results arrive through the
// channel only after the callback's command has completed Run (including Wait).
// A later failure stops dispatch and cancels later active jobs, but lets earlier
// jobs finish under their unchanged command deadlines: cancelling those would
// hide the first source-order error behind a scheduling-dependent cancellation.
func compileWindowsProjectOrdered(parent context.Context, workers, count int, compile func(context.Context, int) error) error {
	if err := parent.Err(); err != nil {
		return err
	}
	type result struct {
		index int
		err   error
	}
	workers = min(windowsProjectCompileWorkers(workers), count)
	done := make(chan result, workers)
	active := make(map[int]context.CancelFunc, workers)
	first, next := count, 0
	var failure error
	launch := func() {
		for len(active) < workers && next < count && first == count && parent.Err() == nil {
			index := next
			next++
			ctx, cancel := context.WithCancel(parent)
			active[index] = cancel
			go func() { done <- result{index, compile(ctx, index)} }()
		}
	}
	launch()
	for len(active) > 0 {
		completed := <-done
		active[completed.index]()
		delete(active, completed.index)
		if completed.err != nil && completed.index < first {
			first, failure = completed.index, completed.err
			for index, cancel := range active {
				if index > first {
					cancel()
				}
			}
		}
		launch()
	}
	if err := parent.Err(); err != nil {
		return err
	}
	return failure
}

func compileWindowsCompilerHostProject(ctx context.Context, toolchain WindowsHostToolchain, directory string, sources []windowsMSVCSource, optimization string, timeout time.Duration, workers int) ([]string, error) {
	// Only the immutable source vector is shared. Each job writes its own source
	// and object, inspects one bounded object, and releases its bytes on return.
	compile := func(ctx context.Context, index int) error {
		source := sources[index]
		sourcePath := filepath.Join(directory, source.Name)
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.WriteFile(sourcePath, []byte(source.Contents), 0o600); err != nil {
			return fmt.Errorf("write project source %s: %w", source.Name, err)
		}
		objectName := strings.TrimSuffix(source.Name, ".c") + ".obj"
		objectPath := filepath.Join(directory, objectName)
		if err := runWindowsProjectCompileCommand(ctx, toolchain.Path, toolchain.Environment, directory, maxProcessOutput, timeout,
			"/nologo", "/std:c11", "/TC", "/MT", "/Brepro", optimization, "/c", sourcePath, "/Fo:"+objectPath); err != nil {
			return windowsProjectSourceError{source.Name, directory, err}
		}
		contents, err := readArtifact(objectPath)
		if err != nil {
			return fmt.Errorf("read project object %s: %w", objectName, err)
		}
		if err := inspectProjectObject(contents); err != nil {
			return fmt.Errorf("inspect project object %s: %w", objectName, err)
		}
		return nil
	}
	// The final runtime source has its separate pre-existing 8MiB admission.
	last := len(sources) - 1
	if err := compileWindowsProjectOrdered(ctx, workers, last, compile); err != nil {
		return nil, err
	}
	if err := compile(ctx, last); err != nil {
		return nil, err
	}
	names := make([]string, len(sources))
	for i, source := range sources {
		names[i] = strings.TrimSuffix(source.Name, ".c") + ".obj"
	}
	return names, nil
}

// Compiler-host-only context adapter. Keep each tool tree in an owned Windows
// job so a worker finishes only after the tool and its descendants exit.
func runWindowsProjectCompileCommand(parent context.Context, name string, environment []string, directory string, limit int, timeout time.Duration, arguments ...string) error {
	if timeout <= 0 || timeout > 10*time.Minute {
		return fmt.Errorf("MSVC command timeout must be in (0,10m]")
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.WaitDelay = 2 * time.Second
	command.Env = environment
	command.Dir = directory
	output := boundedBuffer{limit: limit}
	command.Stdout, command.Stderr = &output, &output
	_, _, err := runOwnedWindowsProcess(command, "")
	return windowsProjectCommandResult(ctx.Err(), err, output, timeout)
}

type windowsJobCleanupError struct {
	wait, cleanup error
}

func (err *windowsJobCleanupError) Error() string {
	return fmt.Sprintf("execution result %v; clean up execution job: %v", err.wait, err.cleanup)
}

func (err *windowsJobCleanupError) Unwrap() error { return err.cleanup }

type windowsProjectSourceError struct {
	name, directory string
	cause           error
}

func (err windowsProjectSourceError) Error() string {
	return "compile project source " + err.name + ": " + strings.ReplaceAll(err.cause.Error(), err.directory, "<temporary>")
}

func (err windowsProjectSourceError) Unwrap() error { return err.cause }

func windowsProjectCommandResult(contextErr, runErr error, output boundedBuffer, timeout time.Duration) error {
	if contextErr != nil {
		if errors.Is(contextErr, context.DeadlineExceeded) {
			contextErr = fmt.Errorf("MSVC host command exceeded %s: %w", timeout, contextErr)
		}
		var cleanup *windowsJobCleanupError
		if errors.As(runErr, &cleanup) {
			return errors.Join(contextErr, fmt.Errorf("clean up execution job: %w", cleanup.cleanup))
		}
		return contextErr
	}
	if output.truncated {
		return fmt.Errorf("MSVC host command exceeded %d output bytes", output.limit)
	}
	if runErr != nil {
		return fmt.Errorf("MSVC host command failed: %w: %s", runErr, strings.TrimSpace(output.String()))
	}
	return nil
}
