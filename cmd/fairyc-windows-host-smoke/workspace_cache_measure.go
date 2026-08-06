package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func measureWorkspaceProcessCache(root string, clockResolution int64) (buildmeasure.CacheReport, error) {
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		return buildmeasure.CacheReport{}, err
	}
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes},
		MaxArtifactBytes: processCacheMaxArtifactBytes,
	})
	if err != nil {
		return buildmeasure.CacheReport{}, err
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return buildmeasure.CacheReport{}, err
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	report := buildmeasure.CacheReport{
		Schema: buildmeasure.CacheSchema, Workload: "workspace_quickstart",
		Scope: "semantic-through-verified-mir", Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "observed-only-no-speedup-claim",
		Environment: buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: string(target.X8664WindowsMSVC)},
		Policy: buildmeasure.CachePolicy{
			MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes, MaxArtifactBytes: processCacheMaxArtifactBytes,
		},
	}
	for index, condition := range []string{buildmeasure.CacheConditionCold, buildmeasure.CacheConditionWarm} {
		started := time.Now()
		checked, err := session.Check(workspace)
		duration := time.Since(started)
		if err != nil {
			return buildmeasure.CacheReport{}, err
		}
		if !checked.Compiler.Verified {
			return buildmeasure.CacheReport{}, fmt.Errorf("%s process-cache sample did not produce verified MIR", condition)
		}
		outcome, err := interpreter.Execute(checked.Compiler.MIR, entry)
		if err != nil {
			return buildmeasure.CacheReport{}, err
		}
		stats := session.Stats()
		durationNS := duration.Nanoseconds()
		report.Samples = append(report.Samples, buildmeasure.CacheSample{
			Sequence: index + 1, Condition: condition, DurationNS: duration.Nanoseconds(),
			BelowClockResolution: durationNS < clockResolution,
			Outcome:              checked.Cache.Outcome, Packages: checked.Cache.Packages,
			Hits: checked.Cache.Hits, Misses: checked.Cache.Misses, Rejected: checked.Cache.Rejected,
			Stored: checked.Cache.Stored, StoreSkipped: checked.Cache.StoreSkipped,
			Lowered: checked.Cache.Lowered, ReassemblyFallback: checked.Cache.ReassemblyFallback,
			MIRFunctions: len(checked.Compiler.MIR.Functions), InterpreterResult: int(outcome.Value),
			RetainedEntries: stats.Store.Entries, RetainedBytes: stats.Store.Bytes, VerifiedHits: stats.VerifiedHits,
		})
	}
	return report, nil
}

func measureWorkspaceProcessCacheEdits(sourceRoot string, clockResolution int64) (buildmeasure.CacheEditReport, error) {
	report := buildmeasure.CacheEditReport{
		Schema: buildmeasure.CacheEditSchema, Workload: "workspace_quickstart",
		Scope: "semantic-through-verified-mir", Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "raw-distributions-only-no-speedup-claim",
		Setup: "fresh-session-per-sample;edit-and-load-outside-timer", WarmupRequests: 1,
		IterationsPerCondition: buildmeasure.CacheEditIterationsPerCondition,
		Environment:            buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: string(target.X8664WindowsMSVC)},
		Policy: buildmeasure.CachePolicy{
			MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes, MaxArtifactBytes: processCacheMaxArtifactBytes,
		},
	}
	conditions := []string{
		buildmeasure.CacheEditNoOpWarm,
		buildmeasure.CacheEditPrivateRoot,
		buildmeasure.CacheEditPrivateDependency,
		buildmeasure.CacheEditPublicDependency,
	}
	for _, condition := range conditions {
		for iteration := 1; iteration <= buildmeasure.CacheEditIterationsPerCondition; iteration++ {
			sample, err := measureWorkspaceProcessCacheEditSample(sourceRoot, condition, iteration, clockResolution)
			if err != nil {
				return buildmeasure.CacheEditReport{}, fmt.Errorf("%s iteration %d: %w", condition, iteration, err)
			}
			sample.Sequence = len(report.Samples) + 1
			report.Samples = append(report.Samples, sample)
		}
	}
	return report, nil
}

func measureWorkspaceProcessCacheEditSample(sourceRoot, condition string, iteration int, clockResolution int64) (buildmeasure.CacheEditSample, error) {
	measurementRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	defer os.RemoveAll(measurementRoot)

	workspace, err := workspaceloader.Load(measurementRoot)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	session, err := workspacecompiler.NewSession(workspacecompiler.Configuration{
		Target:           target.X8664WindowsMSVC,
		Cache:            packagecache.Limits{MaxEntries: processCacheMaxEntries, MaxBytes: processCacheMaxBytes},
		MaxArtifactBytes: processCacheMaxArtifactBytes,
	})
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	warmup, err := session.Check(workspace)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	if !warmup.Compiler.Verified || warmup.Cache.Outcome != workspacecompiler.OutcomeFallback || !warmup.Cache.Lowered {
		return buildmeasure.CacheEditSample{}, fmt.Errorf("untimed warmup did not populate a verified fallback")
	}
	if condition != buildmeasure.CacheEditNoOpWarm {
		if err := mutateWorkspaceMeasurementFixture(measurementRoot, condition); err != nil {
			return buildmeasure.CacheEditSample{}, err
		}
		workspace, err = workspaceloader.Load(measurementRoot)
		if err != nil {
			return buildmeasure.CacheEditSample{}, err
		}
	}
	started := time.Now()
	checked, err := session.Check(workspace)
	duration := time.Since(started)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	if !checked.Compiler.Verified {
		return buildmeasure.CacheEditSample{}, fmt.Errorf("measured request did not produce verified MIR")
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	outcome, err := interpreter.Execute(checked.Compiler.MIR, entry)
	if err != nil {
		return buildmeasure.CacheEditSample{}, err
	}
	stats := session.Stats()
	durationNS := duration.Nanoseconds()
	return buildmeasure.CacheEditSample{
		Condition: condition, Iteration: iteration, DurationNS: durationNS, BelowClockResolution: durationNS < clockResolution,
		Outcome: checked.Cache.Outcome, Packages: checked.Cache.Packages,
		Hits: checked.Cache.Hits, Misses: checked.Cache.Misses, Rejected: checked.Cache.Rejected,
		Stored: checked.Cache.Stored, StoreSkipped: checked.Cache.StoreSkipped,
		Lowered: checked.Cache.Lowered, ReassemblyFallback: checked.Cache.ReassemblyFallback,
		MIRFunctions: len(checked.Compiler.MIR.Functions), InterpreterResult: int(outcome.Value),
		RetainedEntries: stats.Store.Entries, RetainedBytes: stats.Store.Bytes, VerifiedHits: stats.VerifiedHits,
	}, nil
}

func measureStandaloneWorkspaceCache(executablePath, sourceRoot string, clockResolution int64) (buildmeasure.PersistentCacheProcessReport, error) {
	workspaceRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("copy standalone persistent-cache fixture: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	prepareCode, prepareStdout, prepareStderr, err := runStandaloneCommand(workspaceRoot, executablePath, "experimental", "check-workspace")
	if err != nil {
		return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("prepare standalone persistent-cache lockfile: %w", err)
	}
	if prepareCode != cli.ExitSuccess || prepareStderr != "" {
		return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("prepare standalone persistent-cache lockfile returned code %d, stdout %q, stderr %q", prepareCode, prepareStdout, prepareStderr)
	}

	measurement := buildmeasure.PersistentCacheProcessReport{
		Schema: buildmeasure.PersistentCacheProcessSchema, Workload: "workspace_quickstart",
		Scope: "caller-process-run-through-exit;workspace-and-cache-io-included", Clock: "caller-monotonic-wall-nanoseconds",
		ClockResolutionNS: clockResolution, CacheImplementation: "bounded-immutable-disk-package-mir-v2",
		Decision: "raw-paired-distributions-only-no-speedup-claim",
		Setup:    "fresh-cache-per-pair;separate-fairyc-process-per-sample", RetryPolicy: "none;preserve-first-observation",
		IterationsPerCondition: buildmeasure.PersistentCacheProcessIterations,
		Environment:            buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc"},
		Policy: buildmeasure.PersistentCachePolicy{
			MaxEntries: 8, MaxBytes: 67108864, MaxRecordBytes: 8388608,
			MaxArtifactBytes: 4194304, TempMaxAgeSecs: 60,
		},
	}
	for pair := 1; pair <= buildmeasure.PersistentCacheProcessIterations; pair++ {
		cacheDirectory := fmt.Sprintf("build/persistent-cache-pair-%d", pair)
		arguments := []string{
			"experimental", "check-workspace-cache", "--locked",
			"--target", "x86_64-windows-msvc",
			"--cache-dir", cacheDirectory,
			"--cache-max-entries", "8",
			"--cache-max-bytes", "67108864",
			"--cache-max-record-bytes", "8388608",
			"--cache-max-artifact-bytes", "4194304",
			"--cache-temp-max-age-seconds", "60",
		}
		for _, condition := range []string{buildmeasure.PersistentCacheProcessCold, buildmeasure.PersistentCacheProcessWarm} {
			started := time.Now()
			code, stdout, stderr, runErr := runStandaloneCommand(workspaceRoot, executablePath, arguments...)
			duration := time.Since(started).Nanoseconds()
			if runErr != nil {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("execute standalone persistent-cache pair %d %s: %w", pair, condition, runErr)
			}
			if code != cli.ExitSuccess || stderr != "" {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("standalone persistent-cache pair %d %s returned code %d, stdout %q, stderr %q", pair, condition, code, stdout, stderr)
			}
			var report standaloneWorkspaceCacheReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("decode standalone persistent-cache pair %d %s report: %w", pair, condition, err)
			}
			if report.SchemaVersion != "fairyc.experimental.workspace-cache-check/v0" || report.Cache.Storage != workspacecompiler.StorageDisk {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("standalone persistent-cache pair %d %s report is incompatible: %+v", pair, condition, report)
			}
			if measurement.CompilerBuild == "" {
				measurement.CompilerBuild = report.CompilerBuild
			} else if measurement.CompilerBuild != report.CompilerBuild {
				return buildmeasure.PersistentCacheProcessReport{}, fmt.Errorf("standalone persistent-cache compiler identity changed within measurement")
			}
			measurement.Samples = append(measurement.Samples, buildmeasure.PersistentCacheProcessSample{
				Sequence: len(measurement.Samples) + 1, Pair: pair, Condition: condition,
				DurationNS: duration, BelowClockResolution: duration < clockResolution,
				Outcome: report.Cache.Outcome, Packages: report.Cache.Packages,
				Hits: report.Cache.Hits, Misses: report.Cache.Misses, Rejected: report.Cache.Rejected,
				Stored: report.Cache.Stored, StoreSkipped: report.Cache.StoreSkipped, StoreFailed: report.Cache.StoreFailed,
				Lowered: report.Cache.Lowered, ReassemblyFallback: report.Cache.ReassemblyFallback,
				MIRFunctions: report.MIRFunctions, RetainedEntries: report.Cache.RetainedEntries,
				RetainedBytes: report.Cache.RetainedBytes, RecordHits: report.Cache.RecordHits,
				RecordMisses: report.Cache.RecordMisses, RecordRejected: report.Cache.RecordRejected,
				Evictions: report.Cache.Evictions, VerifiedHits: report.Cache.VerifiedHits,
			})
		}
	}
	if err := buildmeasure.ValidatePersistentCacheProcess(measurement); err != nil {
		return buildmeasure.PersistentCacheProcessReport{}, err
	}
	return measurement, nil
}

func measureStandaloneWorkspaceCacheGrowth(executablePath, sourceRoot string) (buildmeasure.PersistentCacheGrowthReport, error) {
	workspaceRoot, err := copyWorkspaceMeasurementFixture(sourceRoot)
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("copy standalone persistent-cache growth fixture: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)
	prepareCode, prepareStdout, prepareStderr, err := runStandaloneCommand(workspaceRoot, executablePath, "experimental", "check-workspace")
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("prepare standalone persistent-cache growth lockfile: %w", err)
	}
	if prepareCode != cli.ExitSuccess || prepareStderr != "" {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("prepare standalone persistent-cache growth lockfile returned code %d, stdout %q, stderr %q", prepareCode, prepareStdout, prepareStderr)
	}
	baseRoot, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(workspaceMainPath)))
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("read persistent-cache growth root source: %w", err)
	}
	baseDependency, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(workspaceMemberSourcePath)))
	if err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("read persistent-cache growth dependency source: %w", err)
	}
	report := buildmeasure.PersistentCacheGrowthReport{
		Schema: buildmeasure.PersistentCacheGrowthSchema, Workload: "workspace_quickstart-retention-sequence",
		Scope:               "retained-disk-state-after-completed-process;source-mutation-excluded",
		CacheImplementation: "bounded-immutable-disk-package-mir-v2", Decision: "bounded-growth-control-only-no-default-policy",
		Setup: "one-cache;separate-fairyc-process-per-sample;mutation-before-process", RetryPolicy: "none;preserve-first-observation",
		Environment: buildmeasure.CacheEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc"},
		Policy: buildmeasure.PersistentCachePolicy{
			MaxEntries: 6, MaxBytes: 67108864, MaxRecordBytes: 8388608,
			MaxArtifactBytes: 4194304, TempMaxAgeSecs: 60,
		},
	}
	conditions := []string{
		buildmeasure.PersistentGrowthBaselineCold,
		buildmeasure.PersistentGrowthPrivateRootOne,
		buildmeasure.PersistentGrowthPrivateRootTwo,
		buildmeasure.PersistentGrowthPrivateDependency,
		buildmeasure.PersistentGrowthPublicDependency,
		buildmeasure.PersistentGrowthPublicNoOp,
		buildmeasure.PersistentGrowthPrivateRootPublic,
		buildmeasure.PersistentGrowthPrivateRootNoOp,
		buildmeasure.PersistentGrowthRepeatedNoOp,
	}
	arguments := []string{
		"experimental", "check-workspace-cache",
		"--target", "x86_64-windows-msvc",
		"--cache-dir", "build/persistent-cache-growth",
		"--cache-max-entries", "6",
		"--cache-max-bytes", "67108864",
		"--cache-max-record-bytes", "8388608",
		"--cache-max-artifact-bytes", "4194304",
		"--cache-temp-max-age-seconds", "60",
	}
	previousEntries := 0
	previousBytes := uint64(0)
	for index, condition := range conditions {
		if err := writePersistentCacheGrowthSources(workspaceRoot, condition, baseRoot, baseDependency); err != nil {
			return buildmeasure.PersistentCacheGrowthReport{}, err
		}
		code, stdout, stderr, runErr := runStandaloneCommand(workspaceRoot, executablePath, arguments...)
		if runErr != nil {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("execute standalone persistent-cache growth sample %d %s: %w", index+1, condition, runErr)
		}
		if code != cli.ExitSuccess || stderr != "" {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("standalone persistent-cache growth sample %d %s returned code %d, stdout %q, stderr %q", index+1, condition, code, stdout, stderr)
		}
		var observation standaloneWorkspaceCacheReport
		if err := json.Unmarshal([]byte(stdout), &observation); err != nil {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("decode standalone persistent-cache growth sample %d %s: %w", index+1, condition, err)
		}
		if observation.SchemaVersion != "fairyc.experimental.workspace-cache-check/v0" || observation.Cache.Storage != workspacecompiler.StorageDisk {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("standalone persistent-cache growth sample %d %s is incompatible: %+v", index+1, condition, observation)
		}
		if report.CompilerBuild == "" {
			report.CompilerBuild = observation.CompilerBuild
		} else if report.CompilerBuild != observation.CompilerBuild {
			return buildmeasure.PersistentCacheGrowthReport{}, fmt.Errorf("standalone persistent-cache compiler identity changed within growth measurement")
		}
		byteDelta := int64(observation.Cache.RetainedBytes) - int64(previousBytes)
		report.Samples = append(report.Samples, buildmeasure.PersistentCacheGrowthSample{
			Sequence: index + 1, Condition: condition,
			Outcome: observation.Cache.Outcome, Packages: observation.Cache.Packages,
			Hits: observation.Cache.Hits, Misses: observation.Cache.Misses, Rejected: observation.Cache.Rejected,
			Stored: observation.Cache.Stored, StoreSkipped: observation.Cache.StoreSkipped, StoreFailed: observation.Cache.StoreFailed,
			Lowered: observation.Cache.Lowered, ReassemblyFallback: observation.Cache.ReassemblyFallback,
			MIRFunctions: observation.MIRFunctions, RetainedEntries: observation.Cache.RetainedEntries,
			RetainedBytes:      observation.Cache.RetainedBytes,
			RetainedEntryDelta: observation.Cache.RetainedEntries - previousEntries, RetainedByteDelta: byteDelta,
			RecordHits: observation.Cache.RecordHits, RecordMisses: observation.Cache.RecordMisses,
			RecordRejected: observation.Cache.RecordRejected, Evictions: observation.Cache.Evictions,
			VerifiedHits: observation.Cache.VerifiedHits,
		})
		previousEntries = observation.Cache.RetainedEntries
		previousBytes = observation.Cache.RetainedBytes
	}
	if err := buildmeasure.ValidatePersistentCacheGrowth(report); err != nil {
		return buildmeasure.PersistentCacheGrowthReport{}, err
	}
	return report, nil
}

func writePersistentCacheGrowthSources(root, condition string, baseRoot, baseDependency []byte) error {
	rootSource := string(baseRoot)
	dependencySource := string(baseDependency)
	switch condition {
	case buildmeasure.PersistentGrowthBaselineCold:
	case buildmeasure.PersistentGrowthPrivateRootOne:
		rootSource = strings.Replace(rootSource, "\nfn i32 main()", "\nstruct retention_root_v1 { i32 value; }\n\nfn i32 main()", 1)
	case buildmeasure.PersistentGrowthPrivateRootTwo:
		rootSource = strings.Replace(rootSource, "\nfn i32 main()", "\nstruct retention_root_v2 { i32 first; i32 second; }\n\nfn i32 main()", 1)
	case buildmeasure.PersistentGrowthPrivateDependency:
		dependencySource = strings.Replace(dependencySource, "\npub fn i32 value()", "\nstruct retention_dependency_v1 { i32 value; }\n\npub fn i32 value()", 1)
	case buildmeasure.PersistentGrowthPublicDependency, buildmeasure.PersistentGrowthPublicNoOp:
		dependencySource += "\npub fn i32 retention_public_v1() {\n    return 7;\n}\n"
	case buildmeasure.PersistentGrowthPrivateRootPublic, buildmeasure.PersistentGrowthPrivateRootNoOp, buildmeasure.PersistentGrowthRepeatedNoOp:
		rootSource = strings.Replace(rootSource, "\nfn i32 main()", "\nstruct retention_root_public_v1 { i32 value; }\n\nfn i32 main()", 1)
		dependencySource += "\npub fn i32 retention_public_v1() {\n    return 7;\n}\n"
	default:
		return fmt.Errorf("unknown persistent-cache growth condition %q", condition)
	}
	if !strings.Contains(rootSource, "fn i32 main()") || !strings.Contains(dependencySource, "pub fn i32 value()") {
		return fmt.Errorf("persistent-cache growth condition %q did not preserve required entry functions", condition)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(workspaceMainPath)), []byte(rootSource), 0o600); err != nil {
		return fmt.Errorf("write persistent-cache growth root source: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(workspaceMemberSourcePath)), []byte(dependencySource), 0o600); err != nil {
		return fmt.Errorf("write persistent-cache growth dependency source: %w", err)
	}
	return nil
}
