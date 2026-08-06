package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cli"
)

func measureStandaloneWorkspaceSession(executablePath, sourceRoot string) (buildmeasure.SessionProcessReport, error) {
	workspaceRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("copy standalone workspace-session fixture: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)

	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(
		contextValue,
		executablePath,
		"experimental", "check-workspace-session",
		"--target", "x86_64-windows-msvc", "--max-requests", "8",
		"--cache-max-entries", "8", "--cache-max-bytes", "33554432", "--cache-max-artifact-bytes", "8388608",
	)
	command.Dir = workspaceRoot
	stdin, err := command.StdinPipe()
	if err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("open standalone workspace-session stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("open standalone workspace-session stdout: %w", err)
	}
	var stderr boundedBuffer
	stderr.limit = subprocessOutputLimit
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("start standalone workspace session: %w", err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = stdin.Close()
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	}()
	probe, err := openProcessMemoryProbe(command.Process.Pid)
	if err != nil {
		return buildmeasure.SessionProcessReport{}, err
	}
	defer probe.close()

	report := buildmeasure.SessionProcessReport{
		Schema: buildmeasure.SessionProcessSchema, Workload: "workspace_quickstart-editor-sequence",
		Scope: "request-write-through-response-read;mutation-excluded", Clock: "caller-monotonic-wall-nanoseconds",
		Setup: "one-standalone-fairyc-process;sequential-requests", RetryPolicy: "none;preserve-first-observation",
		MemoryMetric:        "windows-process-memory-counters-peak-working-set-and-pagefile-bytes",
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "local-observation-only-no-reference-budget-or-speedup-claim",
		Environment: buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc"},
		Policy:      buildmeasure.CachePolicy{MaxEntries: 8, MaxBytes: 33554432, MaxArtifactBytes: 8388608},
	}
	conditions := []string{
		buildmeasure.SessionProcessCold,
		buildmeasure.SessionProcessNoOp,
		buildmeasure.SessionProcessPrivateRoot,
		buildmeasure.SessionProcessNoOpAfterRoot,
		buildmeasure.SessionProcessPrivateDependency,
		buildmeasure.SessionProcessNoOpAfterPrivate,
		buildmeasure.SessionProcessPublicDependency,
		buildmeasure.SessionProcessNoOpAfterPublic,
	}
	reader := bufio.NewReaderSize(stdout, subprocessOutputLimit)
	for index, condition := range conditions {
		switch condition {
		case buildmeasure.SessionProcessPrivateRoot:
			err = mutateWorkspaceMeasurementFixture(workspaceRoot, buildmeasure.CacheEditPrivateRoot)
		case buildmeasure.SessionProcessPrivateDependency:
			err = mutateWorkspaceMeasurementFixture(workspaceRoot, buildmeasure.CacheEditPrivateDependency)
		case buildmeasure.SessionProcessPublicDependency:
			err = mutateWorkspaceMeasurementFixture(workspaceRoot, buildmeasure.CacheEditPublicDependency)
		default:
			err = nil
		}
		if err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("prepare standalone workspace-session %s: %w", condition, err)
		}
		sequence := index + 1
		request := fmt.Sprintf("{\"schemaVersion\":\"fairyc.experimental.workspace-check-session/v0\",\"kind\":\"check\",\"sequence\":%d}\n", sequence)
		started := time.Now()
		if _, err := io.WriteString(stdin, request); err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("write standalone workspace-session request %d: %w", sequence, err)
		}
		line, err := reader.ReadBytes('\n')
		duration := time.Since(started).Nanoseconds()
		if err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("read standalone workspace-session response %d: %w", sequence, err)
		}
		var response standaloneWorkspaceSessionResponse
		if err := json.Unmarshal(line, &response); err != nil {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("decode standalone workspace-session response %d: %w", sequence, err)
		}
		if response.Sequence != sequence || response.Status != "ok" || response.ExitCode != cli.ExitSuccess {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf(
				"standalone workspace-session response %d is not successful: %s; stderr %q",
				sequence,
				line,
				stderr.String(),
			)
		}
		report.Samples = append(report.Samples, buildmeasure.SessionProcessSample{
			Sequence: sequence, Condition: condition, DurationNS: duration,
			Outcome: response.Cache.Outcome, Hits: response.Cache.Hits, Misses: response.Cache.Misses,
			Lowered: response.Cache.Lowered, MIRFunctions: response.MIRFunctions,
			RetainedEntries: response.Cache.RetainedEntries, RetainedBytes: response.Cache.RetainedBytes,
			VerifiedHits: response.Cache.VerifiedHits,
		})
		workingSet, pagefile, err := probe.snapshot()
		if err != nil {
			return buildmeasure.SessionProcessReport{}, err
		}
		if workingSet > report.Memory.PeakWorkingSetBytes {
			report.Memory.PeakWorkingSetBytes = workingSet
		}
		if pagefile > report.Memory.PeakPagefileBytes {
			report.Memory.PeakPagefileBytes = pagefile
		}
	}
	if err := stdin.Close(); err != nil {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("close standalone workspace-session stdin: %w", err)
	}
	if err := command.Wait(); err != nil {
		waited = true
		if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
			return buildmeasure.SessionProcessReport{}, fmt.Errorf("standalone workspace session exceeded timeout")
		}
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("standalone workspace session failed: %w; stderr %q", err, stderr.String())
	}
	waited = true
	if stderr.Len() != 0 {
		return buildmeasure.SessionProcessReport{}, fmt.Errorf("standalone workspace session produced stderr %q", stderr.String())
	}
	return report, nil
}
