package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
)

type targetBuildOptions struct {
	source            string
	additionalSources []string
	targetID          target.ID
	profile           cdriver.Profile
	output            string
}

func runTargetBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseTargetBuildOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsTarget(programSourceSet(options.source, options.additionalSources), options.targetID, options.profile, sourceRoot, stderr)
	if code != ExitSuccess {
		return code
	}
	if err := writeBuildArtifact(sourceRoot, options.output, artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(
		stdout,
		"built: %s (%s, msvc %s, %s)\n",
		options.output,
		options.targetID,
		artifact.Toolchain.Version,
		artifact.Profile.Name,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func compileWindowsTarget(sources []string, targetID target.ID, profile cdriver.Profile, sourceRoot string, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
	if targetID != target.X8664WindowsMSVC {
		return cdriver.WindowsHostArtifact{}, invocationError(stderr, fmt.Sprintf("Windows target compilation requires %s", target.X8664WindowsMSVC))
	}
	lowered, code := lowerProgramSources(sources, sourceRoot, stderr, "Windows target C-emission")
	if code != ExitSuccess {
		return cdriver.WindowsHostArtifact{}, code
	}
	return compileWindowsProgram(lowered.program, lowered.entryName, targetID, profile, stderr)
}

func compileWindowsProgram(program mir.Program, entryName string, targetID target.ID, profile cdriver.Profile, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
	return compileWindowsProgramWithRunReceipt(program, entryName, targetID, profile, false, stderr)
}

func compileWindowsProgramWithRunReceipt(program mir.Program, entryName string, targetID target.ID, profile cdriver.Profile, runReceipt bool, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
	return compileWindowsProgramPolicy(program, entryName, targetID, profile, runReceipt, false, stderr)
}

func compileWindowsProgramPolicy(program mir.Program, entryName string, targetID target.ID, profile cdriver.Profile, runReceipt, sdkStorage bool, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
	if targetID != target.X8664WindowsMSVC {
		return cdriver.WindowsHostArtifact{}, invocationError(stderr, fmt.Sprintf("Windows target compilation requires %s", target.X8664WindowsMSVC))
	}
	emit := cbackend.Emit
	if sdkStorage {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(program, entryName, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return cdriver.WindowsHostArtifact{}, subsetCapabilityError(stderr, "Windows target C-emission", capability)
		}
		return cdriver.WindowsHostArtifact{}, internalError(stderr)
	}
	if err := validateWindowsTargetUnit(unit); err != nil {
		return cdriver.WindowsHostArtifact{}, subsetCapabilityError(stderr, "Windows target build", err)
	}
	var supportSource string
	if sdkStorage {
		supportSource, err = windowsVectorSupportSource(unit.ForeignSymbols, runReceipt)
	} else if runReceipt {
		supportSource, err = windowsTargetRunSupportSource(unit.ForeignSymbols)
	} else {
		supportSource, err = windowsTargetSupportSource(unit.ForeignSymbols)
	}
	if err != nil {
		return cdriver.WindowsHostArtifact{}, subsetCapabilityError(stderr, "Windows target support linkage", err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return cdriver.WindowsHostArtifact{}, buildFailure(stderr, err)
	}
	artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, supportSource)
	if err != nil {
		return cdriver.WindowsHostArtifact{}, buildFailure(stderr, err)
	}
	artifact.TrapControl = runReceipt
	if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
		return cdriver.WindowsHostArtifact{}, buildFailure(stderr, fmt.Errorf("inspect Windows target PE: %w", err))
	}
	if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
		return cdriver.WindowsHostArtifact{}, buildFailure(stderr, fmt.Errorf("inspect Windows target unwind data: %w", err))
	}
	return artifact, ExitSuccess
}

func parseTargetBuildOptions(arguments []string) (targetBuildOptions, error) {
	if len(arguments) == 0 {
		return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" expects one entry source file and explicit --target, --profile, and --output values")
	}
	options := targetBuildOptions{source: arguments[0]}
	if options.source == "" || strings.HasPrefix(options.source, "-") {
		return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" expects the source file before its flags")
	}
	seen := make(map[string]bool, 3)
	for index := 1; index < len(arguments); {
		flag := arguments[index]
		if flag != "--source" && flag != "--target" && flag != "--profile" && flag != "--output" {
			return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" does not accept flag %q", flag)
		}
		if flag != "--source" && seen[flag] {
			return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" received %s more than once", flag)
		}
		if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
			return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" requires a value for %s", flag)
		}
		if flag != "--source" {
			seen[flag] = true
		}
		value := arguments[index+1]
		switch flag {
		case "--source":
			if err := appendProgramSource("build-target", &options.additionalSources, value); err != nil {
				return targetBuildOptions{}, err
			}
		case "--target":
			if target.ID(value) != target.X8664WindowsMSVC {
				return targetBuildOptions{}, fmt.Errorf("experimental build-target currently requires --target %s", target.X8664WindowsMSVC)
			}
			options.targetID = target.X8664WindowsMSVC
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" requires --profile low or --profile optimized")
			}
		case "--output":
			normalized, err := validateBuildOutput(value)
			if err != nil {
				return targetBuildOptions{}, err
			}
			if !strings.HasSuffix(strings.ToLower(normalized), ".exe") {
				return targetBuildOptions{}, fmt.Errorf("experimental build-target Windows output must end with .exe")
			}
			options.output = normalized
		}
		index += 2
	}
	for _, flag := range []string{"--target", "--profile", "--output"} {
		if !seen[flag] {
			return targetBuildOptions{}, fmt.Errorf("experimental command \"build-target\" requires %s", flag)
		}
	}
	return options, nil
}

func validateWindowsTargetUnit(unit cbackend.Unit) error {
	if unit.Target != string(target.X8664WindowsMSVC) || unit.Dialect != cbackend.DialectC11 || unit.Instrumentation != "" {
		return fmt.Errorf("emitted unit does not use the closed Windows MSVC C11 profile")
	}
	return nil
}

func windowsTargetSupportSource(foreignSymbols []string) (string, error) {
	standardLibrarySource, err := cstdlib.WindowsMSVCSourceForSymbols(foreignSymbols)
	if err != nil {
		return "", err
	}
	return combineProductSupportSource(standardLibrarySource)
}

func windowsTargetRunSupportSource(foreignSymbols []string) (string, error) {
	standardLibrarySource, err := cstdlib.WindowsMSVCSourceForSymbols(foreignSymbols)
	if err != nil {
		return "", err
	}
	return combineWindowsRunSupportSource(standardLibrarySource)
}
