package cdriver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/0disoft/fairyc/internal/target"
)

// LinuxHostOutcome keeps the raw process status and the private trap receipt separate.
// Linux process status preserves only the low eight bits of an ordinary i32 return.
type LinuxHostOutcome struct {
	ExitCode     int32
	Stdout       string
	Stderr       string
	TrapCategory uint8
}

// ExecuteLinuxTarget runs one temporary Linux ELF with bounded time and output.
// fd 3 is reserved for the compiler-private one-byte trap receipt.
func ExecuteLinuxTarget(artifact Artifact) (LinuxHostOutcome, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return LinuxHostOutcome{}, fmt.Errorf("Linux target execution requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if artifact.Toolchain.Target != target.X8664LinuxGNU || artifact.ConsumerMode != C11ConsumerMode ||
		len(artifact.Bytes) < 4 || len(artifact.Bytes) > maxArtifactBytes || string(artifact.Bytes[:4]) != "\x7fELF" {
		return LinuxHostOutcome{}, fmt.Errorf("Linux target executable has invalid identity or image")
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-linux-run-")
	if err != nil {
		return LinuxHostOutcome{}, fmt.Errorf("create Linux execution temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	executablePath := filepath.Join(temporaryRoot, "program")
	if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
		return LinuxHostOutcome{}, fmt.Errorf("materialize Linux executable: %w", err)
	}
	receiptRead, receiptWrite, err := os.Pipe()
	if err != nil {
		return LinuxHostOutcome{}, fmt.Errorf("create Linux trap channel: %w", err)
	}
	defer receiptRead.Close()
	defer receiptWrite.Close()
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, executablePath)
	command.Dir = temporaryRoot
	command.Stdin = nil
	command.WaitDelay = time.Second
	command.ExtraFiles = []*os.File{receiptWrite}
	output := newBoundedProcessOutput(maxProcessOutput)
	command.Stdout = output.stdoutWriter()
	command.Stderr = output.stderrWriter()
	if err := command.Start(); err != nil {
		return LinuxHostOutcome{}, fmt.Errorf("start Linux executable: %w", err)
	}
	if err := receiptWrite.Close(); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return LinuxHostOutcome{}, fmt.Errorf("close parent trap channel: %w", err)
	}
	runErr := command.Wait()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return LinuxHostOutcome{}, fmt.Errorf("Linux executable exceeded %s", commandTimeout)
	}
	stdout, stderr, truncated := output.values()
	if truncated {
		return LinuxHostOutcome{}, fmt.Errorf("Linux executable exceeded %d output bytes", maxProcessOutput)
	}
	exitCode := 0
	if runErr != nil {
		var exitError *exec.ExitError
		if !errors.As(runErr, &exitError) || exitError.ExitCode() < 0 {
			return LinuxHostOutcome{}, fmt.Errorf("execute Linux artifact: %w", runErr)
		}
		exitCode = exitError.ExitCode()
	}
	if err := receiptRead.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		return LinuxHostOutcome{}, fmt.Errorf("bound Linux trap channel: %w", err)
	}
	receipt, err := io.ReadAll(io.LimitReader(receiptRead, 2))
	if err != nil {
		return LinuxHostOutcome{}, fmt.Errorf("read Linux trap channel: %w", err)
	}
	if len(receipt) > 1 || len(receipt) == 1 && (receipt[0] < 1 || receipt[0] > 7 || exitCode != 128+int(receipt[0])) {
		return LinuxHostOutcome{}, fmt.Errorf("Linux executable produced invalid trap control data")
	}
	category := uint8(0)
	if len(receipt) == 1 {
		category = receipt[0]
	}
	return LinuxHostOutcome{ExitCode: int32(exitCode), Stdout: stdout, Stderr: stderr, TrapCategory: category}, nil
}
