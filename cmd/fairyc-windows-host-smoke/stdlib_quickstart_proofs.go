package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/pecoff"
)

var processArgsProofArguments = []string{"", "Fairy", "--locked", "a b\"c\\d\\", "한글λ"}

func runStdlibQuickstartProof(root string, directory string, artifactBase string, module string, expectedResult int32, freshFiles ...string) (int, error) {
	workspaceRoot := filepath.Join(root, "examples", directory)
	artifactPath := filepath.Join(workspaceRoot, "build", artifactBase+".exe")
	profiles := []struct {
		name       string
		arguments  []string
		lockStatus string
	}{
		{name: "development", arguments: []string{"build"}, lockStatus: "lock updated"},
		{name: "release", arguments: []string{"build", "--release", "--locked"}, lockStatus: "lock verified"},
	}
	for _, profile := range profiles {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(profile.arguments, workspaceRoot, &stdout, &stderr)
		want := fmt.Sprintf("built: build/%s.exe (x86_64-windows-msvc, %s, %s)\n", artifactBase, profile.name, profile.lockStatus)
		if code != cli.ExitSuccess || stderr.Len() != 0 || stdout.String() != want {
			return 0, fmt.Errorf("%s %s build returned code %d, stdout %q, stderr %q", module, profile.name, code, stdout.String(), stderr.String())
		}
		artifact, err := os.ReadFile(artifactPath)
		if err != nil {
			return 0, fmt.Errorf("read %s %s artifact: %w", module, profile.name, err)
		}
		if _, err := pecoff.InspectExecutable(artifact); err != nil {
			return 0, fmt.Errorf("inspect %s %s PE image: %w", module, profile.name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact); err != nil {
			return 0, fmt.Errorf("inspect %s %s unwind data: %w", module, profile.name, err)
		}
		for _, name := range freshFiles {
			path := filepath.Join(filepath.Dir(artifactPath), name)
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return 0, fmt.Errorf("remove %s %s pre-run file %q: %w", module, profile.name, name, err)
			}
		}
		if err := executeExpectResult(artifactPath, expectedResult); err != nil {
			return 0, fmt.Errorf("execute %s %s artifact: %w", module, profile.name, err)
		}
	}
	return len(profiles), nil
}

func runStdIoScalarQuickstartProof(root string) (int, error) {
	workspaceRoot := filepath.Join(root, "examples", "std_io_scalar_quickstart")
	artifactPath := filepath.Join(workspaceRoot, "build", "std-io-scalar-quickstart.exe")
	expectedStdout := "0\n1\n4294967295\n0\n1\n18446744073709551615\ntrue\nfalse\n"
	expectedStderr := "0\n-42\n-2147483648\n2147483647\n0\n4294967295\n0\n18446744073709551615\ntrue\nfalse\n"
	profiles := []struct {
		name       string
		arguments  []string
		lockStatus string
	}{
		{name: "development", arguments: []string{"build"}, lockStatus: "lock updated"},
		{name: "release", arguments: []string{"build", "--release", "--locked"}, lockStatus: "lock verified"},
	}
	for _, profile := range profiles {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(profile.arguments, workspaceRoot, &stdout, &stderr)
		want := fmt.Sprintf("built: build/std-io-scalar-quickstart.exe (x86_64-windows-msvc, %s, %s)\n", profile.name, profile.lockStatus)
		if code != cli.ExitSuccess || stderr.Len() != 0 || stdout.String() != want {
			return 0, fmt.Errorf("std::io scalar %s build returned code %d, stdout %q, stderr %q", profile.name, code, stdout.String(), stderr.String())
		}
		artifact, err := os.ReadFile(artifactPath)
		if err != nil {
			return 0, fmt.Errorf("read std::io scalar %s artifact: %w", profile.name, err)
		}
		if _, err := pecoff.InspectExecutable(artifact); err != nil {
			return 0, fmt.Errorf("inspect std::io scalar %s PE image: %w", profile.name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact); err != nil {
			return 0, fmt.Errorf("inspect std::io scalar %s unwind data: %w", profile.name, err)
		}
		if err := executeExpectOutput(artifactPath, 0, expectedStdout, expectedStderr); err != nil {
			return 0, fmt.Errorf("execute std::io scalar %s artifact: %w", profile.name, err)
		}
	}
	return len(profiles), nil
}

func runProcessArgsQuickstartProof(root string) (int, int, error) {
	workspaceRoot := filepath.Join(root, "examples", "process_args_quickstart")
	artifactPath := filepath.Join(workspaceRoot, "build", "process-args-quickstart.exe")
	profiles := []struct {
		name       string
		arguments  []string
		lockStatus string
	}{
		{name: "development", arguments: []string{"build"}, lockStatus: "lock updated"},
		{name: "release", arguments: []string{"build", "--release", "--locked"}, lockStatus: "lock verified"},
	}
	executions := 0
	for _, profile := range profiles {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(profile.arguments, workspaceRoot, &stdout, &stderr)
		want := fmt.Sprintf("built: build/process-args-quickstart.exe (x86_64-windows-msvc, %s, %s)\n", profile.name, profile.lockStatus)
		if code != cli.ExitSuccess || stderr.Len() != 0 || stdout.String() != want {
			return 0, 0, fmt.Errorf("%s process-args build returned code %d, stdout %q, stderr %q", profile.name, code, stdout.String(), stderr.String())
		}
		artifact, err := os.ReadFile(artifactPath)
		if err != nil {
			return 0, 0, fmt.Errorf("read %s process-args artifact: %w", profile.name, err)
		}
		if _, err := pecoff.InspectExecutable(artifact); err != nil {
			return 0, 0, fmt.Errorf("inspect %s process-args PE image: %w", profile.name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact); err != nil {
			return 0, 0, fmt.Errorf("inspect %s process-args unwind data: %w", profile.name, err)
		}
		if err := executeExpectResultWithArgs(artifactPath, 42, processArgsProofArguments...); err != nil {
			return 0, 0, fmt.Errorf("execute %s process-args artifact: %w", profile.name, err)
		}
		executions++
	}
	runArguments := append([]string{"run", "--"}, processArgsProofArguments...)
	var developmentStdout bytes.Buffer
	var developmentStderr bytes.Buffer
	developmentCode := cli.RunWithInput(runArguments, workspaceRoot, strings.NewReader(""), &developmentStdout, &developmentStderr)
	if developmentCode != 42 || developmentStdout.Len() != 0 || developmentStderr.Len() != 0 {
		return 0, 0, fmt.Errorf("stable development run with program arguments returned code %d, stdout %q, stderr %q", developmentCode, developmentStdout.String(), developmentStderr.String())
	}
	executions++
	releaseRunArguments := append([]string{"run", "--release", "--locked", "--"}, processArgsProofArguments...)
	var releaseStdout bytes.Buffer
	var releaseStderr bytes.Buffer
	releaseCode := cli.RunWithInput(releaseRunArguments, workspaceRoot, strings.NewReader(""), &releaseStdout, &releaseStderr)
	if releaseCode != 42 || releaseStdout.Len() != 0 || releaseStderr.Len() != 0 {
		return 0, 0, fmt.Errorf("stable release run with program arguments returned code %d, stdout %q, stderr %q", releaseCode, releaseStdout.String(), releaseStderr.String())
	}
	executions++
	for _, rejection := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "embedded NUL", arguments: []string{"run", "--", "a\x00b"}, contains: "NUL byte"},
		{name: "unknown flag before delimiter", arguments: []string{"run", "--bogus", "--", "x"}, contains: "accepts only optional --release and --locked"},
		{name: "build delimiter", arguments: []string{"build", "--", "x"}, contains: "accepts only optional --release and --locked"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := cli.Run(rejection.arguments, workspaceRoot, &stdout, &stderr)
		if code != cli.ExitInvalidInvocation || stdout.Len() != 0 || !strings.Contains(stderr.String(), rejection.contains) {
			return 0, 0, fmt.Errorf("%s rejection returned code %d, stdout %q, stderr %q", rejection.name, code, stdout.String(), stderr.String())
		}
	}
	return len(profiles), executions, nil
}
