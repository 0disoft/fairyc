package cbackendnative

import (
	"bytes"
	"fmt"

	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"os"
	"path/filepath"
	"strings"
)

func runBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "product-build-smoke", buildSmokeSource)
}

func runStandardLibraryBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "stdlib-build-smoke", textValidationSourceCallSmokeSource)
}

func runPublicFileBuildSmoke(temporaryRoot string) (int, error) {
	definition, err := sdkstdlib.Load()
	if err != nil {
		return 0, fmt.Errorf("load compiler-bundled standard library: %w", err)
	}
	return runBuildSourceSmokeWithAdditionalSources(
		temporaryRoot,
		"public-file-build-smoke",
		publicFileBuildSmokeSource,
		definition.Sources,
	)
}

func runByteStringLiteralBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "byte-string-literal-build-smoke", byteStringLiteralBuildSmokeSource)
}

func runUTF8StringLiteralBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "utf8-string-literal-build-smoke", utf8StringLiteralBuildSmokeSource)
}

func runCStringLiteralBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "c-string-literal-build-smoke", cStringLiteralBuildSmokeSource)
}

func runBuildSourceSmoke(temporaryRoot, smokeID, source string) (int, error) {
	return runBuildSourceSmokeWithAdditionalSources(temporaryRoot, smokeID, source, nil)
}

func runBuildSourceSmokeWithAdditionalSources(
	temporaryRoot,
	smokeID,
	source string,
	additionalSources []sdkstdlib.Source,
) (int, error) {
	if smokeID == "" || source == "" {
		return 0, fmt.Errorf("product build smoke identity and source must be non-empty")
	}
	sourceRoot := filepath.Join(temporaryRoot, smokeID)
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		return 0, fmt.Errorf("create %s root: %w", smokeID, err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.fy"), []byte(source), 0o600); err != nil {
		return 0, fmt.Errorf("write %s source: %w", smokeID, err)
	}
	for _, additional := range additionalSources {
		path := filepath.Join(sourceRoot, filepath.FromSlash(additional.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return 0, fmt.Errorf("create %s additional-source directory: %w", smokeID, err)
		}
		if err := os.WriteFile(path, additional.Bytes, 0o600); err != nil {
			return 0, fmt.Errorf("write %s additional source %s: %w", smokeID, additional.Path, err)
		}
	}
	executions := 0
	for _, compiler := range []string{"clang", "gcc"} {
		for _, profile := range nativeProfiles {
			output := fmt.Sprintf("build/smoke-%s-%s", compiler, profile.Name)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			arguments := []string{"experimental", "build", "main.fy"}
			for _, additional := range additionalSources {
				arguments = append(arguments, "--source", additional.Path)
			}
			arguments = append(arguments,
				"--cc", compiler,
				"--profile", profile.Name,
				"--output", output,
			)
			code := cli.Run(arguments, sourceRoot, &stdout, &stderr)
			if code != cli.ExitSuccess {
				return 0, fmt.Errorf("%s with %s/%s failed with code %d: %s", smokeID, compiler, profile.Name, code, strings.TrimSpace(stderr.String()))
			}
			expectedReceipt := fmt.Sprintf("built: %s (%s, %s, %s)\n", output, target.X8664LinuxGNU, compiler, profile.Name)
			if stdout.String() != expectedReceipt || stderr.Len() != 0 {
				return 0, fmt.Errorf("%s with %s/%s produced stdout %q and stderr %q", smokeID, compiler, profile.Name, stdout.String(), stderr.String())
			}
			runRoot := filepath.Join(sourceRoot, fmt.Sprintf("run-%s-%s", compiler, profile.Name))
			if err := os.Mkdir(runRoot, 0o700); err != nil {
				return 0, fmt.Errorf("create %s execution root for %s/%s: %w", smokeID, compiler, profile.Name, err)
			}
			if err := executeInDirectory(smokeID, filepath.Join(sourceRoot, filepath.FromSlash(output)), runRoot, 42, nil); err != nil {
				return 0, fmt.Errorf("%s with %s/%s: %w", smokeID, compiler, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}
