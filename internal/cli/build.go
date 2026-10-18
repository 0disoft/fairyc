package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	productTrapExitBase   = cruntime.NativeTrapExitBase
	maxBuildPathBytes     = 4096
	maxBuildComponentSize = 255
)

type buildOptions struct {
	source            string
	additionalSources []string
	compilerName      string
	profile           cdriver.Profile
	output            string
}

func runBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseBuildOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	lowered, code := lowerProgramSources(programSourceSet(options.source, options.additionalSources), sourceRoot, stderr, "C-emission")
	if code != ExitSuccess {
		return code
	}
	unit, err := cbackend.Emit(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return subsetCapabilityError(stderr, "C-emission", capability)
		}
		return internalError(stderr)
	}
	supportSource, err := productSupportSource(unit.ForeignSymbols)
	if err != nil {
		return internalError(stderr)
	}
	toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, options.compilerName)
	if err != nil {
		return buildFailure(stderr, err)
	}
	artifact, err := cdriver.Build(toolchain, options.profile, unit.Source, supportSource)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if err := writeBuildArtifact(sourceRoot, options.output, artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(
		stdout,
		"built: %s (%s, %s, %s)\n",
		options.output,
		target.X8664LinuxGNU,
		artifact.Toolchain.Name,
		artifact.Profile.Name,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func productSupportSource(foreignSymbols []string) (string, error) {
	standardLibrarySource, err := cstdlib.SourceForSymbols(foreignSymbols)
	if err != nil {
		return "", fmt.Errorf("resolve Stage 0 standard-library source: %w", err)
	}
	return combineProductSupportSource(standardLibrarySource)
}

func combineProductSupportSource(standardLibrarySource string) (string, error) {
	runtimeSource, err := cruntime.ScalarSource(productTrapExitBase)
	if err != nil {
		return "", err
	}
	return combineStandardLibraryAndRuntime(standardLibrarySource, runtimeSource), nil
}

func combineWindowsRunSupportSource(standardLibrarySource string) (string, error) {
	runtimeSource, err := cruntime.WindowsRunSource(productTrapExitBase)
	if err != nil {
		return "", err
	}
	return combineStandardLibraryAndRuntime(standardLibrarySource, runtimeSource), nil
}

func combineStandardLibraryAndRuntime(standardLibrarySource, runtimeSource string) string {
	if standardLibrarySource == "" {
		return runtimeSource
	}
	return standardLibrarySource + "\n" + runtimeSource
}

func parseBuildOptions(arguments []string) (buildOptions, error) {
	if len(arguments) == 0 {
		return buildOptions{}, fmt.Errorf("experimental command \"build\" expects one entry source file and explicit --cc, --profile, and --output values")
	}
	options := buildOptions{source: arguments[0]}
	if options.source == "" || strings.HasPrefix(options.source, "-") {
		return buildOptions{}, fmt.Errorf("experimental command \"build\" expects the source file before its flags")
	}
	_, err := parseCommandFlags(arguments[1:], flagSchema{Command: "build", Experimental: true,
		Flags: []flagDefinition{
			{Name: "--source", Kind: flagSlice},
			{Name: "--cc", Kind: flagScalar},
			{Name: "--profile", Kind: flagScalar},
			{Name: "--output", Kind: flagScalar},
		}, Required: []string{"--cc", "--profile", "--output"}}, func(flag, value string) error {
		switch flag {
		case "--source":
			if err := appendProgramSource("build", &options.additionalSources, value); err != nil {
				return err
			}
		case "--cc":
			if value != "clang" && value != "gcc" {
				return fmt.Errorf("experimental command \"build\" requires --cc clang or --cc gcc")
			}
			options.compilerName = value
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return fmt.Errorf("experimental command \"build\" requires --profile low or --profile optimized")
			}
		case "--output":
			normalized, err := validateBuildOutput(value)
			if err != nil {
				return err
			}
			options.output = normalized
		}
		return nil
	})
	if err != nil {
		return buildOptions{}, err
	}

	return options, nil
}

func validateBuildOutput(value string) (string, error) {
	if value == "" || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("experimental build output must be valid non-empty UTF-8")
	}
	if len([]byte(value)) > maxBuildPathBytes {
		return "", fmt.Errorf("experimental build output exceeds %d UTF-8 bytes", maxBuildPathBytes)
	}
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf("experimental build output must use forward slashes")
	}
	if path.IsAbs(value) || strings.HasPrefix(value, "//") || isWindowsVolumePath(value) {
		return "", fmt.Errorf("experimental build output must be relative to the source root")
	}
	if path.Clean(value) != value {
		return "", fmt.Errorf("experimental build output must be a canonical relative path")
	}
	components := strings.Split(value, "/")
	if len(components) < 2 || components[0] != "build" {
		return "", fmt.Errorf("experimental build output must be inside the ignored build/ directory")
	}
	for _, component := range components {
		if err := validateBuildComponent(component); err != nil {
			return "", err
		}
	}
	return value, nil
}

func isWindowsVolumePath(value string) bool {
	if len(value) >= 2 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' {
		return true
	}
	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, `//?/`) || strings.HasPrefix(value, `//./`)
}

func validateBuildComponent(component string) error {
	return validatePortablePathComponent("experimental build output", component)
}

func validatePortablePathComponent(label, component string) error {
	if component == "" || component == "." || component == ".." {
		return fmt.Errorf("%s contains an invalid path component", label)
	}
	if len([]byte(component)) > maxBuildComponentSize {
		return fmt.Errorf("%s component exceeds %d UTF-8 bytes", label, maxBuildComponentSize)
	}
	if strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
		return fmt.Errorf("%s components cannot end with a dot or space", label)
	}
	for _, character := range component {
		if character < 0x20 || strings.ContainsRune(`<>:"|?*`, character) {
			return fmt.Errorf("%s contains a platform-reserved character", label)
		}
	}
	base, _, _ := strings.Cut(component, ".")
	upper := strings.ToUpper(base)
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" ||
		(len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9') {
		return fmt.Errorf("%s contains the reserved name %q", label, component)
	}
	return nil
}

func writeBuildArtifact(sourceRoot, relativePath string, contents []byte) error {
	return publishBuildArtifact(sourceRoot, relativePath, contents, false)
}

func replaceBuildArtifact(sourceRoot, relativePath string, contents []byte) error {
	return publishBuildArtifact(sourceRoot, relativePath, contents, true)
}

func publishBuildArtifact(sourceRoot, relativePath string, contents []byte, replace bool) error {
	if _, err := validateBuildOutput(relativePath); err != nil {
		return err
	}
	if len(contents) == 0 {
		return fmt.Errorf("refusing to write an empty executable")
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return fmt.Errorf("open source root for build output: %w", err)
	}
	defer root.Close()
	localPath := filepath.FromSlash(relativePath)
	parent := filepath.Dir(localPath)
	if err := root.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create build output directory %q: %w", path.Dir(relativePath), err)
	}
	temporaryPath, file, err := createBuildStagingFile(root, parent)
	if err != nil {
		return err
	}
	stagingExists := true
	defer func() {
		if stagingExists {
			_ = root.Remove(temporaryPath)
		}
	}()
	if _, err := bytes.NewReader(contents).WriteTo(file); err != nil {
		_ = file.Close()
		return fmt.Errorf("write staged build output: %w", err)
	}
	if err := file.Chmod(0o700); err != nil {
		_ = file.Close()
		return fmt.Errorf("set staged build output executable mode: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync staged build output: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close staged build output: %w", err)
	}
	if replace {
		if err := root.Rename(temporaryPath, localPath); err != nil {
			return fmt.Errorf("replace build output %q: %w", relativePath, err)
		}
		stagingExists = false
		return nil
	}
	if err := root.Link(temporaryPath, localPath); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("build output %q already exists; refusing to overwrite it", relativePath)
		}
		return fmt.Errorf("promote build output %q: %w", relativePath, err)
	}
	if err := root.Remove(temporaryPath); err != nil {
		removeErr := root.Remove(localPath)
		if removeErr != nil {
			return fmt.Errorf("remove staged build output after promotion: %v; rollback also failed: %v", err, removeErr)
		}
		return fmt.Errorf("remove staged build output after promotion: %w", err)
	}
	stagingExists = false
	return nil
}

func createBuildStagingFile(root *os.Root, parent string) (string, *os.File, error) {
	for attempt := 0; attempt < 8; attempt++ {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", nil, fmt.Errorf("generate build staging name: %w", err)
		}
		name := ".fairyc-" + hex.EncodeToString(random[:]) + ".tmp"
		candidate := filepath.Join(parent, name)
		file, err := root.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
		if err == nil {
			return candidate, file, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, fmt.Errorf("create staged build output: %w", err)
		}
	}
	return "", nil, fmt.Errorf("cannot allocate a unique staged build output name")
}

func buildFailure(stderr io.Writer, err error) int {
	message := "build failed"
	if err != nil {
		message += ": " + strings.TrimSpace(err.Error())
	}
	if _, writeErr := fmt.Fprintf(stderr, "fairyc: %s\n", message); writeErr != nil {
		return ExitInternalFailure
	}
	return ExitBuildFailure
}
