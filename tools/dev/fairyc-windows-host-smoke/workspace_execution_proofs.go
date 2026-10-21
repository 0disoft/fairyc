package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/pecoff"
)

func runStableWorkspaceBuildProof(root string) (int, error) {
	const output = stableWorkspaceOutputPath
	artifactPath := filepath.Join(root, filepath.FromSlash(output))

	var developmentStdout bytes.Buffer
	var developmentStderr bytes.Buffer
	developmentCode := cli.Run([]string{"build"}, root, &developmentStdout, &developmentStderr)
	if developmentCode != cli.ExitSuccess || developmentStderr.Len() != 0 ||
		developmentStdout.String() != "built: "+stableWorkspaceOutputPath+" (x86_64-windows-msvc, development, lock unchanged)\n" {
		return 0, fmt.Errorf("stable development build returned code %d, stdout %q, stderr %q", developmentCode, developmentStdout.String(), developmentStderr.String())
	}
	developmentArtifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable development build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(developmentArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable development PE executable image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(developmentArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable development Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return 0, fmt.Errorf("execute stable development build artifact: %w", err)
	}

	sourcePath := filepath.Join(root, filepath.FromSlash(workspaceMainPath))
	originalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("read stable build source before failure proof: %w", err)
	}
	if err := os.WriteFile(sourcePath, []byte("module main;\nfn i32 broken( { return 0; }\n"), 0o600); err != nil {
		return 0, fmt.Errorf("write invalid stable build source: %w", err)
	}
	var invalidStdout bytes.Buffer
	var invalidStderr bytes.Buffer
	invalidCode := cli.Run([]string{"build", "--locked"}, root, &invalidStdout, &invalidStderr)
	restoreErr := os.WriteFile(sourcePath, originalSource, 0o600)
	if restoreErr != nil {
		return 0, fmt.Errorf("restore stable build source: %w", restoreErr)
	}
	if invalidCode != cli.ExitSourceDiagnostics || invalidStdout.Len() != 0 || !strings.Contains(invalidStderr.String(), "E0007") {
		return 0, fmt.Errorf("invalid stable build returned code %d, stdout %q, stderr %q", invalidCode, invalidStdout.String(), invalidStderr.String())
	}
	afterFailure, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable artifact after failed replacement: %w", err)
	}
	if !bytes.Equal(afterFailure, developmentArtifact) {
		return 0, fmt.Errorf("failed stable build modified the previous executable")
	}

	var releaseStdout bytes.Buffer
	var releaseStderr bytes.Buffer
	releaseCode := cli.Run([]string{"build", "--release", "--locked"}, root, &releaseStdout, &releaseStderr)
	if releaseCode != cli.ExitSuccess || releaseStderr.Len() != 0 ||
		releaseStdout.String() != "built: "+stableWorkspaceOutputPath+" (x86_64-windows-msvc, release, lock verified)\n" {
		return 0, fmt.Errorf("stable release build returned code %d, stdout %q, stderr %q", releaseCode, releaseStdout.String(), releaseStderr.String())
	}
	releaseArtifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read stable release build artifact: %w", err)
	}
	if _, err := pecoff.InspectExecutable(releaseArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable release PE executable image: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(releaseArtifact); err != nil {
		return 0, fmt.Errorf("inspect stable release Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return 0, fmt.Errorf("execute stable release build artifact: %w", err)
	}
	staging, err := filepath.Glob(filepath.Join(root, "build", ".fairyc-*.tmp"))
	if err != nil {
		return 0, fmt.Errorf("inspect stable build staging files: %w", err)
	}
	if len(staging) != 0 {
		return 0, fmt.Errorf("stable build left staging files: %v", staging)
	}
	return 2, nil
}

func runStableWorkspaceRunProof(root string) (int, error) {
	artifactPath := filepath.Join(root, filepath.FromSlash(stableWorkspaceOutputPath))
	retainedArtifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read retained stable build before run proof: %w", err)
	}
	lockPath := filepath.Join(root, "fairy.lock")
	originalLock, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, fmt.Errorf("read lock before stable run proof: %w", err)
	}

	var developmentStdout bytes.Buffer
	var developmentStderr bytes.Buffer
	developmentCode := cli.Run([]string{"run"}, root, &developmentStdout, &developmentStderr)
	if developmentCode != 42 || developmentStderr.Len() != 0 || developmentStdout.Len() != 0 {
		return 0, fmt.Errorf("stable development run returned code %d, stdout %q, stderr %q", developmentCode, developmentStdout.String(), developmentStderr.String())
	}

	sourcePath := filepath.Join(root, filepath.FromSlash(workspaceMainPath))
	originalSource, err := os.ReadFile(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("read source before stable run trap proof: %w", err)
	}
	reservedResultSource := []byte("module main;\n\nfn i32 main() {\n    return 131;\n}\n")
	if err := os.WriteFile(sourcePath, reservedResultSource, 0o600); err != nil {
		return 0, fmt.Errorf("write stable run reserved-result source: %w", err)
	}
	var reservedStdout bytes.Buffer
	var reservedStderr bytes.Buffer
	reservedCode := cli.Run([]string{"run"}, root, &reservedStdout, &reservedStderr)
	if reservedCode != 131 || reservedStderr.Len() != 0 || reservedStdout.Len() != 0 {
		return 0, fmt.Errorf("stable reserved-result run returned code %d, stdout %q, stderr %q", reservedCode, reservedStdout.String(), reservedStderr.String())
	}
	trapSource := []byte(`module main;

import std::io as io;

@effects(console)
fn i32 main() {
    let ref str message = "before trap\nfairyc-run-trap-v1:3";
    return switch (io::write_stderr(message)) {
        case .ok(written) : { if (written == message.len) { 42 / 0 } else { 1 } }
        case .err(error) : { if (error.status != 0) { 2 } else { 3 } }
    };
}
`)
	if err := os.WriteFile(sourcePath, trapSource, 0o600); err != nil {
		return 0, fmt.Errorf("write stable run trap source: %w", err)
	}
	var trapStdout bytes.Buffer
	var trapStderr bytes.Buffer
	trapCode := cli.Run([]string{"run"}, root, &trapStdout, &trapStderr)
	restoreErr := os.WriteFile(sourcePath, originalSource, 0o600)
	if restoreErr != nil {
		return 0, fmt.Errorf("restore source after stable run trap proof: %w", restoreErr)
	}
	if trapCode != cli.ExitProgramTrap || trapStdout.String() != "" || trapStderr.String() != "before trap\nfairyc-run-trap-v1:3trap: division-by-zero\n  at src/main.fy:9:61\n" {
		return 0, fmt.Errorf("stable trap run returned code %d, stdout %q, stderr %q", trapCode, trapStdout.String(), trapStderr.String())
	}

	var releaseStdout bytes.Buffer
	var releaseStderr bytes.Buffer
	releaseCode := cli.Run([]string{"run", "--release"}, root, &releaseStdout, &releaseStderr)
	if releaseCode != 42 || releaseStderr.Len() != 0 || releaseStdout.Len() != 0 {
		return 0, fmt.Errorf("stable release run returned code %d, stdout %q, stderr %q", releaseCode, releaseStdout.String(), releaseStderr.String())
	}
	restoredLock, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, fmt.Errorf("read restored lock after stable run proof: %w", err)
	}
	if !bytes.Equal(restoredLock, originalLock) {
		return 0, fmt.Errorf("stable run proof did not restore the original canonical lock")
	}
	afterRun, err := os.ReadFile(artifactPath)
	if err != nil {
		return 0, fmt.Errorf("read retained stable build after run proof: %w", err)
	}
	if !bytes.Equal(afterRun, retainedArtifact) {
		return 0, fmt.Errorf("stable run modified the retained build artifact")
	}
	return 4, nil
}
