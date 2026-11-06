package cdriver

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"
)

const maxWindowsExecutionOutputBytes = 64 << 20

// ExecuteWindowsMSVCTargetWithContext retains the ordinary owned-job, bounded
// pipes and cleanup policy while allowing the caller to cancel native execution.
func ExecuteWindowsMSVCTargetWithContext(ctx context.Context, artifact WindowsHostArtifact, input io.Reader, timeout time.Duration) (WindowsHostOutcome, error) {
	return ExecuteWindowsMSVCTargetWithContextAndOutputLimit(ctx, artifact, input, timeout, maxProcessOutput)
}

// ExecuteWindowsMSVCTargetWithContextAndOutputLimit permits an explicit bounded
// binary response without changing ordinary execution's 64KiB combined capture.
// Stdout/stderr share the limit; owned-job, cancellation and pipe cleanup remain.
func ExecuteWindowsMSVCTargetWithContextAndOutputLimit(ctx context.Context, artifact WindowsHostArtifact, input io.Reader, timeout time.Duration, outputLimit int) (WindowsHostOutcome, error) {
	if ctx == nil || input == nil {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution requires a context and input")
	}
	if timeout <= 0 || timeout > commandTimeout {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution timeout must be greater than zero and at most %s", commandTimeout)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifactWithOutputLimit(ctx, artifact, "MSVC target", input, nil, timeout, "", outputLimit)
}
