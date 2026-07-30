package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/workspacecallcatalog"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	atlasCallProcessPackageUID     = "example.workspace-quickstart.app"
	atlasCallProcessMaxPackages    = 8
	atlasCallProcessMaxCalls       = 8
	atlasCallProcessMaxOutputBytes = uint64(2 << 20)
)

type standaloneAtlasSymbolCatalog struct {
	Catalog struct {
		Symbols []struct {
			Symbol struct {
				Visible string `json:"visible"`
			} `json:"symbol"`
		} `json:"symbols"`
	} `json:"catalog"`
}

type standaloneAtlasCallCatalog struct {
	CatalogVersion string `json:"catalog_version"`
	Encoding       string `json:"encoding"`
	Catalog        struct {
		Context struct {
			CompilerBuild string `json:"compiler_build"`
			PackageUID    string `json:"package_uid"`
		} `json:"context"`
		Direction string `json:"direction"`
		Usage     struct {
			Packages          int `json:"packages"`
			Calls             int `json:"calls"`
			CrossPackageCalls int `json:"cross_package_calls"`
			SelectedCalls     int `json:"selected_calls"`
		} `json:"usage"`
		Calls []struct {
			Source struct {
				PackageInstance string `json:"package_instance"`
			} `json:"source"`
			Target struct {
				PackageInstance string `json:"package_instance"`
			} `json:"target"`
		} `json:"calls"`
		Complete bool `json:"complete"`
	} `json:"catalog"`
}

func measureStandaloneAtlasCallProcesses(
	sourceRoot, executablePath string,
	clockResolution int64,
) (buildmeasure.AtlasCallProcessReport, error) {
	workspace, err := workspaceloader.Load(sourceRoot)
	if err != nil {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("load Atlas call process workload: %w", err)
	}
	shape := buildmeasure.AtlasCallProcessShape{Packages: len(workspace.Packages)}
	for _, loaded := range workspace.Packages {
		for _, file := range loaded.Sources {
			shape.SourceFiles++
			if uint64(len(file.Bytes)) > ^uint64(0)-shape.SourceBytes {
				return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("Atlas call process source-byte count overflow")
			}
			shape.SourceBytes += uint64(len(file.Bytes))
		}
	}

	symbolCode, symbolStdout, symbolStderr, err := runStandaloneCommand(
		sourceRoot, executablePath,
		"experimental", "atlas-symbols", "list",
		"--package-uid", atlasCallProcessPackageUID,
		"--kind", "function", "--name", "main",
		"--max-symbols", "4", "--max-selection-bytes", "1048576", "--max-output-bytes", "2097152", "--locked",
	)
	if err != nil {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("discover Atlas call process subject: %w", err)
	}
	if symbolCode != cli.ExitSuccess || symbolStderr != "" {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf(
			"Atlas call process symbol discovery returned code %d, stdout %q, stderr %q",
			symbolCode, symbolStdout, symbolStderr,
		)
	}
	var symbols standaloneAtlasSymbolCatalog
	if err := json.Unmarshal([]byte(symbolStdout), &symbols); err != nil {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("decode Atlas call process symbol catalog: %w", err)
	}
	if len(symbols.Catalog.Symbols) != 1 || symbols.Catalog.Symbols[0].Symbol.Visible == "" {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("Atlas call process symbol discovery was not exact")
	}

	arguments := atlasCallProcessArguments(symbols.Catalog.Symbols[0].Symbol.Visible)
	warmupCode, warmupStdout, warmupStderr, err := runStandaloneCommand(sourceRoot, executablePath, arguments...)
	if err != nil {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("warm Atlas call process workload: %w", err)
	}
	warmup, err := decodeStandaloneAtlasCallCatalog(warmupCode, warmupStdout, warmupStderr)
	if err != nil {
		return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("validate Atlas call process warmup: %w", err)
	}
	shape.Calls = warmup.Catalog.Usage.Calls
	shape.CrossPackageCalls = warmup.Catalog.Usage.CrossPackageCalls
	shape.SelectedCalls = warmup.Catalog.Usage.SelectedCalls

	report := buildmeasure.AtlasCallProcessReport{
		Schema: buildmeasure.AtlasCallProcessSchema, Workload: "workspace_quickstart-root-main-outgoing",
		Scope: "caller-command-start-through-exit;compiler-build-symbol-discovery-and-warmup-excluded",
		Clock: "caller-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		Setup:          "one-untimed-warmup;fresh-standalone-fairyc-process-per-sample;locked-workspace;os-caches-uncontrolled",
		RetryPolicy:    "none;preserve-first-observation",
		MemoryMetric:   "windows-process-memory-counters-peak-working-set-and-pagefile-bytes",
		Decision:       "local-control-only-no-reference-budget-speedup-traversal-or-impact-claim",
		Iterations:     buildmeasure.AtlasCallProcessIterations,
		CompilerBuild:  warmup.Catalog.Context.CompilerBuild,
		CatalogVersion: workspacecallcatalog.CatalogVersion, CatalogEncoding: workspacecallcatalog.CatalogEncoding,
		Environment: buildmeasure.AtlasCallProcessEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH},
		Policy: buildmeasure.AtlasCallProcessPolicy{
			MaxPackages: atlasCallProcessMaxPackages, MaxCalls: atlasCallProcessMaxCalls,
			MaxOutputBytes: atlasCallProcessMaxOutputBytes,
		},
		Shape: shape,
	}
	for index := 0; index < buildmeasure.AtlasCallProcessIterations; index++ {
		code, stdout, stderr, duration, workingSet, pagefile, err := runMeasuredStandaloneCommand(
			sourceRoot, executablePath, arguments...,
		)
		if err != nil {
			return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("measure Atlas call process sample %d: %w", index+1, err)
		}
		if _, err := decodeStandaloneAtlasCallCatalog(code, stdout, stderr); err != nil {
			return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("validate Atlas call process sample %d: %w", index+1, err)
		}
		if stdout != warmupStdout {
			return buildmeasure.AtlasCallProcessReport{}, fmt.Errorf("Atlas call process sample %d changed canonical catalog facts", index+1)
		}
		report.Samples = append(report.Samples, buildmeasure.AtlasCallProcessSample{
			Sequence: index + 1, DurationNS: duration, BelowClockResolution: duration < clockResolution,
			PeakWorkingSetBytes: workingSet, PeakPagefileBytes: pagefile, OutputBytes: uint64(len(stdout)),
		})
	}
	if err := buildmeasure.ValidateAtlasCallProcess(report); err != nil {
		return buildmeasure.AtlasCallProcessReport{}, err
	}
	return report, nil
}

func atlasCallProcessArguments(symbol string) []string {
	return []string{
		"experimental", "atlas-calls", "list",
		"--package-uid", atlasCallProcessPackageUID,
		"--symbol", symbol, "--direction", "outgoing",
		"--max-packages", strconv.Itoa(atlasCallProcessMaxPackages),
		"--max-calls", strconv.Itoa(atlasCallProcessMaxCalls),
		"--max-output-bytes", strconv.FormatUint(atlasCallProcessMaxOutputBytes, 10),
		"--locked",
	}
}

func decodeStandaloneAtlasCallCatalog(code int, stdout, stderr string) (standaloneAtlasCallCatalog, error) {
	if code != cli.ExitSuccess || stderr != "" {
		return standaloneAtlasCallCatalog{}, fmt.Errorf(
			"Atlas call catalog returned code %d, stdout %q, stderr %q", code, stdout, stderr,
		)
	}
	var result standaloneAtlasCallCatalog
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		return standaloneAtlasCallCatalog{}, fmt.Errorf("decode Atlas call catalog: %w", err)
	}
	if result.CatalogVersion != workspacecallcatalog.CatalogVersion ||
		result.Encoding != workspacecallcatalog.CatalogEncoding ||
		result.Catalog.Context.PackageUID != atlasCallProcessPackageUID ||
		result.Catalog.Direction != "outgoing" || !result.Catalog.Complete ||
		result.Catalog.Usage.Packages != 2 || result.Catalog.Usage.Calls != 1 ||
		result.Catalog.Usage.CrossPackageCalls != 1 || result.Catalog.Usage.SelectedCalls != 1 ||
		len(result.Catalog.Calls) != 1 ||
		result.Catalog.Calls[0].Source.PackageInstance == result.Catalog.Calls[0].Target.PackageInstance {
		return standaloneAtlasCallCatalog{}, fmt.Errorf("Atlas call catalog did not preserve the fixed cross-package workload")
	}
	return result, nil
}

func runMeasuredStandaloneCommand(
	directory, executablePath string,
	arguments ...string,
) (int, string, string, int64, uint64, uint64, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath, arguments...)
	command.Dir = directory
	command.Stdin = nil
	var stdout boundedBuffer
	var stderr boundedBuffer
	stdout.limit = subprocessOutputLimit
	stderr.limit = subprocessOutputLimit
	command.Stdout = &stdout
	command.Stderr = &stderr
	started := time.Now()
	if err := command.Start(); err != nil {
		return -1, stdout.String(), stderr.String(), 0, 0, 0, err
	}
	probe, err := openProcessMemoryProbe(command.Process.Pid)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return -1, stdout.String(), stderr.String(), 0, 0, 0, err
	}
	waitErr := command.Wait()
	duration := time.Since(started).Nanoseconds()
	workingSet, pagefile, snapshotErr := probe.snapshot()
	closeErr := probe.close()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return -1, stdout.String(), stderr.String(), duration, workingSet, pagefile, fmt.Errorf("command exceeded timeout")
	}
	if snapshotErr != nil {
		return -1, stdout.String(), stderr.String(), duration, 0, 0, snapshotErr
	}
	if closeErr != nil {
		return -1, stdout.String(), stderr.String(), duration, workingSet, pagefile, closeErr
	}
	if waitErr == nil {
		return cli.ExitSuccess, stdout.String(), stderr.String(), duration, workingSet, pagefile, nil
	}
	var exitError *exec.ExitError
	if !errors.As(waitErr, &exitError) {
		return -1, stdout.String(), stderr.String(), duration, workingSet, pagefile, waitErr
	}
	return exitError.ExitCode(), stdout.String(), stderr.String(), duration, workingSet, pagefile, nil
}
