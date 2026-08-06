package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fairyc Windows host smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	sources, err := loadHostSmokeSources()
	if err != nil {
		return err
	}
	root, err := os.MkdirTemp("", "fairyc-windows-host-smoke-")
	if err != nil {
		return fmt.Errorf("create temporary source root: %w", err)
	}
	defer os.RemoveAll(root)
	for name, source := range sources {
		destination := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return fmt.Errorf("create %s smoke source directory: %w", name, err)
		}
		if err := os.WriteFile(destination, []byte(source), 0o600); err != nil {
			return fmt.Errorf("write %s smoke source: %w", name, err)
		}
	}

	var buildStdout bytes.Buffer
	var buildStderr bytes.Buffer
	buildCode := cli.Run([]string{
		"experimental", "build-host", checkedInModuleMainPath,
		"--source", checkedInModuleSupportPath,
		"--profile", "optimized", "--output", "build/main.exe",
	}, root, &buildStdout, &buildStderr)
	if buildCode != cli.ExitSuccess || buildStderr.Len() != 0 || !strings.HasPrefix(buildStdout.String(), "built: build/main.exe (windows/amd64 host-smoke, msvc ") {
		return fmt.Errorf("build-host returned code %d, stdout %q, stderr %q", buildCode, buildStdout.String(), buildStderr.String())
	}
	artifactPath := filepath.Join(root, "build", "main.exe")
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("read build-host artifact: %w", err)
	}
	peReport, err := pecoff.InspectExecutable(artifact)
	if err != nil {
		return fmt.Errorf("inspect build-host PE executable image: %w", err)
	}
	unwindReport, err := pecoff.InspectAMD64Unwind(artifact)
	if err != nil {
		return fmt.Errorf("inspect build-host Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return err
	}

	targetBuildExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/windows-target-%s.exe", profile)
		var targetBuildStdout bytes.Buffer
		var targetBuildStderr bytes.Buffer
		targetBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInWindowsTargetPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &targetBuildStdout, &targetBuildStderr)
		if targetBuildCode != cli.ExitSuccess || targetBuildStderr.Len() != 0 || !strings.HasPrefix(targetBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("build-target %s returned code %d, stdout %q, stderr %q", profile, targetBuildCode, targetBuildStdout.String(), targetBuildStderr.String())
		}
		targetArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		targetArtifact, err := os.ReadFile(targetArtifactPath)
		if err != nil {
			return fmt.Errorf("read build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(targetArtifact); err != nil {
			return fmt.Errorf("inspect build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(targetArtifact); err != nil {
			return fmt.Errorf("inspect build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(targetArtifactPath, 42); err != nil {
			return fmt.Errorf("execute build-target %s artifact: %w", profile, err)
		}
		targetBuildExecutions++
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return fmt.Errorf("resolve Windows MSVC probe toolchain: %w", err)
	}
	clockResolution, err := observeMonotonicClockResolution()
	if err != nil {
		return fmt.Errorf("observe process-cache measurement clock: %w", err)
	}
	cacheMeasurement, err := measureWorkspaceProcessCache(root, clockResolution)
	if err != nil {
		return fmt.Errorf("measure process-local workspace cache: %w", err)
	}
	cacheMeasurementJSON, err := buildmeasure.MarshalCache(cacheMeasurement)
	if err != nil {
		return fmt.Errorf("encode process-local workspace cache measurement: %w", err)
	}
	fmt.Printf("Fairy C process-cache measurement: %s\n", cacheMeasurementJSON)
	cacheEditMeasurement, err := measureWorkspaceProcessCacheEdits(root, clockResolution)
	if err != nil {
		return fmt.Errorf("measure process-local workspace cache edits: %w", err)
	}
	cacheEditMeasurementJSON, err := buildmeasure.MarshalCacheEdits(cacheEditMeasurement)
	if err != nil {
		return fmt.Errorf("encode process-local workspace cache edit measurement: %w", err)
	}
	fmt.Printf("Fairy C process-cache edit measurement: %s\n", cacheEditMeasurementJSON)
	atlasRewriteMeasurement, err := measureAtlasRewrite(root, clockResolution)
	if err != nil {
		return fmt.Errorf("measure Atlas whole-package rewrite: %w", err)
	}
	atlasRewriteMeasurementJSON, err := buildmeasure.MarshalAtlasRewrite(atlasRewriteMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas whole-package rewrite measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas rewrite measurement: %s\n", atlasRewriteMeasurementJSON)
	atlasCallScaleMeasurement, err := measureAtlasCallScale(clockResolution)
	if err != nil {
		return fmt.Errorf("measure Atlas workspace-call scaling: %w", err)
	}
	atlasCallScaleMeasurementJSON, err := buildmeasure.MarshalAtlasCallScale(atlasCallScaleMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas workspace-call scaling measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas call scale measurement: %s\n", atlasCallScaleMeasurementJSON)
	measurement := buildmeasure.Report{
		Schema:              buildmeasure.Schema,
		Workload:            "workspace_quickstart",
		Clock:               "process-monotonic-wall-nanoseconds",
		CacheImplementation: "none",
		Decision:            "baseline-only",
		Environment: buildmeasure.Environment{
			OS: runtime.GOOS, Arch: runtime.GOARCH, Target: "x86_64-windows-msvc", Toolchain: "msvc " + toolchain.Version,
		},
	}
	workspaceBuildExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		for _, condition := range []string{buildmeasure.ConditionOrdinary, buildmeasure.ConditionLocked} {
			label := "ordinary"
			arguments := []string{
				"experimental", "build-workspace",
				"--target", "x86_64-windows-msvc", "--profile", profile,
			}
			if condition == buildmeasure.ConditionLocked {
				label = "locked"
				arguments = append(arguments, "--locked")
			}
			output := fmt.Sprintf("build/workspace-%s-%s.exe", profile, label)
			arguments = append(arguments, "--output", output)
			var workspaceBuildStdout bytes.Buffer
			var workspaceBuildStderr bytes.Buffer
			started := time.Now()
			workspaceBuildCode := cli.Run(arguments, root, &workspaceBuildStdout, &workspaceBuildStderr)
			duration := time.Since(started)
			lockState := "verified"
			if condition == buildmeasure.ConditionOrdinary {
				lockState = "unchanged"
				if profile == "low" {
					lockState = "updated"
				}
			}
			if workspaceBuildCode != cli.ExitSuccess || workspaceBuildStderr.Len() != 0 ||
				!strings.HasPrefix(workspaceBuildStdout.String(), "built workspace: "+output+" (x86_64-windows-msvc, msvc ") ||
				!strings.Contains(workspaceBuildStdout.String(), ", lock "+lockState+")") {
				return fmt.Errorf("build-workspace %s %s returned code %d, stdout %q, stderr %q", profile, label, workspaceBuildCode, workspaceBuildStdout.String(), workspaceBuildStderr.String())
			}
			workspaceArtifactPath := filepath.Join(root, filepath.FromSlash(output))
			workspaceArtifact, err := os.ReadFile(workspaceArtifactPath)
			if err != nil {
				return fmt.Errorf("read build-workspace %s %s artifact: %w", profile, label, err)
			}
			if _, err := pecoff.InspectExecutable(workspaceArtifact); err != nil {
				return fmt.Errorf("inspect build-workspace %s %s PE executable image: %w", profile, label, err)
			}
			if _, err := pecoff.InspectAMD64Unwind(workspaceArtifact); err != nil {
				return fmt.Errorf("inspect build-workspace %s %s Microsoft x64 unwind data: %w", profile, label, err)
			}
			if err := executeExpectResult(workspaceArtifactPath, 42); err != nil {
				return fmt.Errorf("execute build-workspace %s %s artifact: %w", profile, label, err)
			}
			measurement.Samples = append(measurement.Samples, buildmeasure.Sample{
				Sequence: len(measurement.Samples) + 1, Profile: profile, Condition: condition,
				LockState: lockState, DurationNS: duration.Nanoseconds(), ArtifactBytes: int64(len(workspaceArtifact)), NativeResult: 42,
			})
			workspaceBuildExecutions++
		}
	}
	stableBuildExecutions, err := runStableWorkspaceBuildProof(root)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stableBuildExecutions
	measurementJSON, err := buildmeasure.Marshal(measurement)
	if err != nil {
		return fmt.Errorf("encode Windows workspace build measurement: %w", err)
	}
	fmt.Printf("Fairy C build measurement: %s\n", measurementJSON)
	windowsFileSourceExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/windows-file-%s.exe", profile)
		var fileBuildStdout bytes.Buffer
		var fileBuildStderr bytes.Buffer
		fileBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInWindowsFilePath,
			"--source", checkedInWindowsFileSupportPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &fileBuildStdout, &fileBuildStderr)
		if fileBuildCode != cli.ExitSuccess || fileBuildStderr.Len() != 0 || !strings.HasPrefix(fileBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("Windows File build-target %s returned code %d, stdout %q, stderr %q", profile, fileBuildCode, fileBuildStdout.String(), fileBuildStderr.String())
		}
		fileArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		fileArtifact, err := os.ReadFile(fileArtifactPath)
		if err != nil {
			return fmt.Errorf("read Windows File build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(fileArtifact); err != nil {
			return fmt.Errorf("inspect Windows File build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(fileArtifact); err != nil {
			return fmt.Errorf("inspect Windows File build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(fileArtifactPath, 42); err != nil {
			return fmt.Errorf("execute Windows File build-target %s artifact: %w", profile, err)
		}
		windowsFileSourceExecutions++
	}
	ownedTextSourceExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/owned-text-%s.exe", profile)
		var ownedTextBuildStdout bytes.Buffer
		var ownedTextBuildStderr bytes.Buffer
		ownedTextBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInOwnedTextPath,
			"--source", checkedInOwnedTextSupportPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &ownedTextBuildStdout, &ownedTextBuildStderr)
		if ownedTextBuildCode != cli.ExitSuccess || ownedTextBuildStderr.Len() != 0 || !strings.HasPrefix(ownedTextBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("owned-text build-target %s returned code %d, stdout %q, stderr %q", profile, ownedTextBuildCode, ownedTextBuildStdout.String(), ownedTextBuildStderr.String())
		}
		ownedTextArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		ownedTextArtifact, err := os.ReadFile(ownedTextArtifactPath)
		if err != nil {
			return fmt.Errorf("read owned-text build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(ownedTextArtifact); err != nil {
			return fmt.Errorf("inspect owned-text build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(ownedTextArtifact); err != nil {
			return fmt.Errorf("inspect owned-text build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(ownedTextArtifactPath, 42); err != nil {
			return fmt.Errorf("execute owned-text build-target %s artifact: %w", profile, err)
		}
		ownedTextSourceExecutions++
	}
	targetRunExecutions := 0
	var targetRunStdout bytes.Buffer
	var targetRunStderr bytes.Buffer
	targetRunCode := cli.Run([]string{
		"experimental", "run-target", checkedInWindowsTargetPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &targetRunStdout, &targetRunStderr)
	if targetRunCode != cli.ExitSuccess || targetRunStderr.Len() != 0 || !strings.HasPrefix(targetRunStdout.String(), "native result: 42 (x86_64-windows-msvc, msvc ") {
		return fmt.Errorf("run-target returned code %d, stdout %q, stderr %q", targetRunCode, targetRunStdout.String(), targetRunStderr.String())
	}
	targetRunExecutions++
	workspaceRunExecutions := 0
	var workspaceRunStdout bytes.Buffer
	var workspaceRunStderr bytes.Buffer
	workspaceRunCode := cli.Run([]string{
		"experimental", "run-workspace", "--locked",
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &workspaceRunStdout, &workspaceRunStderr)
	if workspaceRunCode != cli.ExitSuccess || workspaceRunStderr.Len() != 0 || !strings.HasPrefix(workspaceRunStdout.String(), "native workspace result: 42 (x86_64-windows-msvc, msvc ") {
		return fmt.Errorf("run-workspace returned code %d, stdout %q, stderr %q", workspaceRunCode, workspaceRunStdout.String(), workspaceRunStderr.String())
	}
	workspaceRunExecutions++
	stableRunExecutions, err := runStableWorkspaceRunProof(root)
	if err != nil {
		return err
	}
	workspaceRunExecutions += stableRunExecutions
	targetExecExecutions := 0
	var targetExecStdout bytes.Buffer
	var targetExecStderr bytes.Buffer
	targetExecCode := cli.Run([]string{
		"experimental", "exec-target", checkedInWindowsOutputPath,
		"--source", checkedInWindowsOutputSupportPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &targetExecStdout, &targetExecStderr)
	if targetExecCode != 42 || targetExecStdout.String() != "Fairy C says hi\n" || targetExecStderr.Len() != 0 {
		return fmt.Errorf("exec-target returned code %d, stdout %q, stderr %q", targetExecCode, targetExecStdout.String(), targetExecStderr.String())
	}
	targetExecExecutions++
	stdIORoot := filepath.Join(root, "examples", "std_io_quickstart")
	var stdIOBuildStdout bytes.Buffer
	var stdIOBuildStderr bytes.Buffer
	stdIOBuildCode := cli.Run([]string{"build", "--release"}, stdIORoot, &stdIOBuildStdout, &stdIOBuildStderr)
	if stdIOBuildCode != cli.ExitSuccess || stdIOBuildStderr.Len() != 0 ||
		stdIOBuildStdout.String() != "built: build/std-io-quickstart.exe (x86_64-windows-msvc, release, lock updated)\n" {
		return fmt.Errorf("std::io stable build returned code %d, stdout %q, stderr %q", stdIOBuildCode, stdIOBuildStdout.String(), stdIOBuildStderr.String())
	}
	stdIOArtifactPath := filepath.Join(stdIORoot, "build", "std-io-quickstart.exe")
	stdIOArtifact, err := os.ReadFile(stdIOArtifactPath)
	if err != nil {
		return fmt.Errorf("read std::io stable build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(stdIOArtifact); err != nil {
		return fmt.Errorf("inspect std::io stable build PE image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(stdIOArtifact); err != nil {
		return fmt.Errorf("inspect std::io stable build unwind data: %w", err)
	}
	if err := executeExpectOutput(
		stdIOArtifactPath,
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\n",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return fmt.Errorf("execute std::io stable build artifact: %w", err)
	}
	if err := executeExpectOutputWithInput(
		stdIOArtifactPath,
		"stdin!\n",
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return fmt.Errorf("execute std::io stable build artifact with short stdin: %w", err)
	}
	if err := executeExpectOutputWithInput(
		stdIOArtifactPath,
		"123456789",
		0,
		"Fairy C says hi\nbytes on stdout\n-2147483648\n12345678",
		"Fairy C warns too\nbytes on stderr\n",
	); err != nil {
		return fmt.Errorf("execute std::io stable build artifact with bounded stdin: %w", err)
	}
	var stdIORunStdout bytes.Buffer
	var stdIORunStderr bytes.Buffer
	stdIORunCode := cli.RunWithInput(
		[]string{"run", "--release", "--locked"},
		stdIORoot,
		strings.NewReader("stdin!\n"),
		&stdIORunStdout,
		&stdIORunStderr,
	)
	if stdIORunCode != cli.ExitSuccess || stdIORunStdout.String() != "Fairy C says hi\nbytes on stdout\n-2147483648\nstdin!\n" ||
		stdIORunStderr.String() != "Fairy C warns too\nbytes on stderr\n" {
		return fmt.Errorf("std::io stable run returned code %d, stdout %q, stderr %q", stdIORunCode, stdIORunStdout.String(), stdIORunStderr.String())
	}
	workspaceRunExecutions++
	workspaceBuildExecutions++
	stdAllocBuildExecutions, err := runStdlibQuickstartProof(root, "std_alloc_quickstart", "std-alloc-quickstart", "std::alloc", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdAllocBuildExecutions
	stdBytesBuildExecutions, err := runStdlibQuickstartProof(root, "std_bytes_quickstart", "std-bytes-quickstart", "std::bytes", 0)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdBytesBuildExecutions
	stdTextBuildExecutions, err := runStdlibQuickstartProof(root, "std_text_quickstart", "std-text-quickstart", "std::text", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdTextBuildExecutions
	stdTextFromBufferBuildExecutions, err := runStdlibQuickstartProof(root, "std_text_from_buffer_quickstart", "std-text-from-buffer-quickstart", "std::bytes", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdTextFromBufferBuildExecutions
	stdFSBuildExecutions, err := runStdlibQuickstartProofWithFixtures(
		root,
		"std_fs_quickstart",
		"std-fs-quickstart",
		"std::fs",
		42,
		[]string{"fairyc-std-fs-read-only-metadata.tmp"},
		[]hardLinkFixture{{
			path:  "fairyc-std-fs-hard-link-metadata.tmp",
			alias: "fairyc-std-fs-hard-link-metadata-alias.tmp",
		}},
		"fairyc-std-fs-data.tmp",
		"fairyc-std-fs-drop.tmp",
		"fairyc-std-fs-open-or-create.tmp",
		"fairyc-std-fs-create-new.tmp",
		"fairyc-std-fs-append.tmp",
		"fairyc-std-fs-read-append-error.tmp",
		"fairyc-std-fs-copy-source.tmp",
		"fairyc-std-fs-copy-destination.tmp",
		"fairyc-std-fs-copy-missing-destination.tmp",
		"fairyc-std-fs-hard-link-source.tmp",
		"fairyc-std-fs-hard-link-destination.tmp",
		"fairyc-std-fs-hard-link-missing.tmp",
		"fairyc-std-fs-hard-link-missing-destination.tmp",
		"fairyc-std-fs-read-only-mutation.tmp",
		"fairyc-std-fs-remove-all\\nested\\b.tmp",
		"fairyc-std-fs-remove-all\\a.tmp",
		"fairyc-std-fs-remove-all\\nested",
		"fairyc-std-fs-remove-all",
		"fairyc-std-fs-remove-all-blocked\\child.tmp",
		"fairyc-std-fs-remove-all-blocked",
		"fairyc-std-fs-remove-all-file.tmp",
		"fairyc-std-fs-directory-all\\collision.tmp",
		"fairyc-std-fs-directory-all\\nested\\leaf",
		"fairyc-std-fs-directory-all\\nested",
		"fairyc-std-fs-directory-all",
		"fairyc-std-fs-directory-all-partial\\created",
		"fairyc-std-fs-directory-all-partial",
	)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdFSBuildExecutions
	stdIoScalarBuildExecutions, err := runStdIoScalarQuickstartProof(root)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += stdIoScalarBuildExecutions
	regionBuildExecutions, err := runStdlibQuickstartProof(root, "regions", "regions", "Region", 42)
	if err != nil {
		return err
	}
	workspaceBuildExecutions += regionBuildExecutions
	var atlasCallProcessMeasurement buildmeasure.AtlasCallProcessReport
	standaloneCompilerBytes, err := runStandaloneCompiler(repositoryRoot, root, clockResolution, &atlasCallProcessMeasurement)
	if err != nil {
		return err
	}
	atlasCallProcessMeasurementJSON, err := buildmeasure.MarshalAtlasCallProcess(atlasCallProcessMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas call process measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas call process measurement: %s\n", atlasCallProcessMeasurementJSON)
	atlasCallRetainedMeasurement, err := measureAtlasCallRetainedHeap()
	if err != nil {
		return fmt.Errorf("measure Atlas workspace-call retained heap: %w", err)
	}
	atlasCallRetainedMeasurementJSON, err := buildmeasure.MarshalAtlasCallRetained(atlasCallRetainedMeasurement)
	if err != nil {
		return fmt.Errorf("encode Atlas workspace-call retained-heap measurement: %w", err)
	}
	fmt.Printf("Fairy C Atlas call retained-heap measurement: %s\n", atlasCallRetainedMeasurementJSON)
	targetRunExecutions++
	targetExecExecutions++
	workspaceBuildExecutions += 2
	workspaceRunExecutions += 3

	for _, testCase := range []struct {
		name         string
		source       string
		support      string
		profile      string
		code         int
		stdoutPrefix string
		stderr       string
	}{
		{name: "success low", source: checkedInSuccessPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "multi-source success optimized", source: checkedInModuleMainPath, support: checkedInModuleSupportPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "byte-view iteration low", source: checkedInWindowsQuickstartPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "byte-view iteration optimized", source: checkedInWindowsQuickstartPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "negative low", source: "negative.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "negative optimized", source: "negative.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "division by zero low", source: "division_by_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "division by zero optimized", source: "division_by_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "remainder by zero low", source: "remainder_by_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "remainder by zero optimized", source: "remainder_by_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "signed division overflow low", source: "signed_division_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: signed-division-overflow\n"},
		{name: "signed division overflow optimized", source: "signed_division_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: signed-division-overflow\n"},
		{name: "integer overflow low", source: "integer_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "integer overflow optimized", source: "integer_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "usize overflow low", source: "usize_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "usize overflow optimized", source: "usize_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "unsigned division low", source: "unsigned_division.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned division optimized", source: "unsigned_division.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned division by zero low", source: "unsigned_division_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "unsigned division by zero optimized", source: "unsigned_division_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "unsigned bitwise shift low", source: "unsigned_bitwise_shift.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned bitwise shift optimized", source: "unsigned_bitwise_shift.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "unsigned invalid shift low", source: "unsigned_invalid_shift.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "unsigned invalid shift optimized", source: "unsigned_invalid_shift.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "left shift low", source: "left_shift.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "left shift optimized", source: "left_shift.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "left shift overflow low", source: "left_shift_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "left shift overflow optimized", source: "left_shift_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "invalid shift low", source: "invalid_shift.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "invalid shift optimized", source: "invalid_shift.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "byte-view bounds low", source: "bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view bounds optimized", source: "bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array bounds low", source: "array_bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array bounds optimized", source: "array_bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array store bounds low", source: "array_store_bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "fixed-array store bounds optimized", source: "array_store_bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view slice bounds low", source: "slice_bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view slice bounds optimized", source: "slice_bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "assertion failure low", source: "assertion_failure.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: assertion-failed\n"},
		{name: "assertion failure optimized", source: "assertion_failure.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: assertion-failed\n"},
		{name: "lexical defer low", source: "lexical_defer.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "lexical defer optimized", source: "lexical_defer.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
	} {
		if err := runHostCase(root, testCase.name, testCase.source, testCase.support, testCase.profile, testCase.code, testCase.stdoutPrefix, testCase.stderr); err != nil {
			return err
		}
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(root, "user-enum Result "+profile, "enum_result.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
		if err := runHostCase(root, "user-enum Result propagation "+profile, "enum_result_propagation.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
		if err := runHostCase(root, "user-enum Option "+profile, "enum_option.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
		if err := runHostCase(root, "copy fixed-array Option/Result "+profile, "tagged_array.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			return err
		}
	}
	windowsFileExecutions, err := runWindowsFileProbe(toolchain)
	if err != nil {
		return err
	}
	bufferProbeExecutions, err := runBufferFailureProbe(toolchain)
	if err != nil {
		return err
	}
	ownedTextProbeExecutions, err := runOwnedTextStorageProbe(toolchain)
	if err != nil {
		return err
	}
	arenaProbeExecutions, err := runArenaFailureProbe(toolchain)
	if err != nil {
		return err
	}
	regionFailureProbeImages, regionFailureCases, err := runGeneratedFairyRegionFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	bufferFailureProbeImages, bufferFailureCases, err := runGeneratedFairyBufferFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	readAppendFailureProbeImages, readAppendFailureCases, err := runGeneratedFairyReadAppendFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	readToEndFailureProbeImages, readToEndFailureCases, err := runGeneratedFairyReadToEndFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	stringAppendFailureProbeImages, stringAppendFailureCases, err := runGeneratedFairyStringAppendFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	cstringFailureProbeImages, cstringFailureCases, err := runGeneratedFairyCStringFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	stringCopyFailureProbeImages, stringCopyFailureCases, err := runGeneratedFairyStringCopyFailureProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	stringFromBufferProbeImages, stringFromBufferProbeCases, err := runGeneratedFairyStringFromBufferProbe(repositoryRoot, root, toolchain)
	if err != nil {
		return err
	}
	processArgsBuildImages, processArgsExecutions, err := runProcessArgsQuickstartProof(root)
	if err != nil {
		return err
	}
	processArgsDecodeImages, processArgsDecodeExecutions, err := runWindowsProcessArgsDecodeProbe(toolchain)
	if err != nil {
		return err
	}
	nativeTestArgsExecutions, err := runStableNativeTestArgsProof(root)
	if err != nil {
		return err
	}
	nativeTestStdinExecutions, err := runStableNativeTestStdinProof(root)
	if err != nil {
		return err
	}
	nativeTestOutputExecutions, err := runStableNativeTestOutputProof(root)
	if err != nil {
		return err
	}
	layoutExecutions, layoutCases, err := runWindowsLayoutProbe(toolchain)
	if err != nil {
		return err
	}
	abiExecutions, abiCases, err := runWindowsABIProbe(toolchain)
	if err != nil {
		return err
	}
	relocationExecutions, err := runWindowsRelocationProbe(toolchain)
	if err != nil {
		return err
	}
	weakSearchExecutions, err := runWindowsWeakSearchProbe(toolchain)
	if err != nil {
		return err
	}
	archiveExecutions, err := runWindowsArchiveProbe(toolchain)
	if err != nil {
		return err
	}
	structResultExecutions, err := runFairyStructResultProbe(toolchain)
	if err != nil {
		return err
	}
	cIntCastExecutions, err := runCIntCastProbe(toolchain)
	if err != nil {
		return err
	}
	unwindExecutions, unwindCases, err := runWindowsUnwindProbe(toolchain)
	if err != nil {
		return err
	}
	reportHostSmokeSuccess(hostSmokeReceipt{
		artifactBytes:                  len(artifact),
		validatedSections:              len(peReport.Sections),
		runtimeFunctions:               unwindReport.RuntimeFunctions,
		version2Infos:                  unwindReport.Version2Infos,
		targetBuildExecutions:          targetBuildExecutions,
		workspaceBuildExecutions:       workspaceBuildExecutions,
		workspaceRunExecutions:         workspaceRunExecutions,
		windowsFileSourceExecutions:    windowsFileSourceExecutions,
		ownedTextSourceExecutions:      ownedTextSourceExecutions,
		targetRunExecutions:            targetRunExecutions,
		targetExecExecutions:           targetExecExecutions,
		windowsFileExecutions:          windowsFileExecutions,
		bufferProbeExecutions:          bufferProbeExecutions,
		ownedTextProbeExecutions:       ownedTextProbeExecutions,
		arenaProbeExecutions:           arenaProbeExecutions,
		regionFailureProbeImages:       regionFailureProbeImages,
		regionFailureCases:             regionFailureCases,
		bufferFailureProbeImages:       bufferFailureProbeImages,
		bufferFailureCases:             bufferFailureCases,
		readAppendFailureProbeImages:   readAppendFailureProbeImages,
		readAppendFailureCases:         readAppendFailureCases,
		readToEndFailureProbeImages:    readToEndFailureProbeImages,
		readToEndFailureCases:          readToEndFailureCases,
		stringAppendFailureProbeImages: stringAppendFailureProbeImages,
		stringAppendFailureCases:       stringAppendFailureCases,
		cstringFailureProbeImages:      cstringFailureProbeImages,
		cstringFailureCases:            cstringFailureCases,
		stringCopyFailureProbeImages:   stringCopyFailureProbeImages,
		stringCopyFailureCases:         stringCopyFailureCases,
		stringFromBufferProbeImages:    stringFromBufferProbeImages,
		stringFromBufferProbeCases:     stringFromBufferProbeCases,
		stdFSBuildExecutions:           stdFSBuildExecutions,
		processArgsExecutions:          processArgsExecutions,
		processArgsDecodeExecutions:    processArgsDecodeExecutions,
		processArgsBuildImages:         processArgsBuildImages,
		processArgsDecodeImages:        processArgsDecodeImages,
		nativeTestArgsExecutions:       nativeTestArgsExecutions,
		nativeTestStdinExecutions:      nativeTestStdinExecutions,
		nativeTestOutputExecutions:     nativeTestOutputExecutions,
		stdIoScalarBuildExecutions:     stdIoScalarBuildExecutions,
		cIntCastExecutions:             cIntCastExecutions,
		atlasCallProcessIterations:     buildmeasure.AtlasCallProcessIterations,
		atlasCallScaleShapes:           len(atlasCallScaleMeasurement.Shapes),
		atlasCallRetainedBatches:       len(atlasCallRetainedMeasurement.Shapes) * buildmeasure.AtlasCallRetainedIterationsPerShape,
		layoutCases:                    layoutCases,
		layoutExecutions:               layoutExecutions,
		abiCases:                       abiCases,
		abiExecutions:                  abiExecutions,
		relocationExecutions:           relocationExecutions,
		weakSearchExecutions:           weakSearchExecutions,
		archiveExecutions:              archiveExecutions,
		structResultExecutions:         structResultExecutions,
		unwindCases:                    unwindCases,
		unwindExecutions:               unwindExecutions,
		standaloneCompilerBytes:        standaloneCompilerBytes,
		toolchainVersion:               toolchain.Version,
	})
	return nil
}
