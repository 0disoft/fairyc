package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime"

	"github.com/0disoft/fairyc/internal/desktopapp"
)

type desktopPackageReport = desktopapp.Report
type desktopPython = desktopapp.Python

func desktopDeclared(root string) (bool, error)         { return desktopapp.Declared(root) }
func desktopHelper(root, script string) (string, error) { return desktopapp.Helper(root, script) }
func findDesktopPython() (desktopPython, error)         { return desktopapp.FindPython() }
func runDesktopHelper(python desktopPython, root, script string, args ...string) ([]byte, error) {
	return desktopapp.RunHelper(python, root, script, args...)
}
func prepareDesktopBuild(root string) error { return desktopapp.PrepareBuild(root) }
func parseDesktopPackageReport(data []byte, root string) (desktopPackageReport, error) {
	return desktopapp.ParseReport(data, root)
}

func runStablePackage(arguments []string, root string, stdout, stderr io.Writer) int {
	seen, err := parseCommandFlags(arguments, flagSchema{Command: "package", Flags: []flagDefinition{{Name: "--locked", Kind: flagSwitch}, {Name: "--json", Kind: flagSwitch}}}, func(string, string) error { return nil })
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return invocationError(stderr, "package currently supports Windows amd64 desktop projects only")
	}
	declared, err := desktopDeclared(root)
	if err != nil {
		return desktopPackageFailure(stderr, "invalid app.toml/fairy.windows.json", ExitPackageFailure)
	}
	if !declared {
		return invocationError(stderr, "package requires a Win32 desktop project with app.toml and fairy.windows.json")
	}
	if _, err = desktopHelper(root, "package_release.py"); err != nil {
		return desktopPackageFailure(stderr, "package_release.py is missing or outside the project", ExitPackageFailure)
	}
	python, err := findDesktopPython()
	if err != nil {
		return desktopPackageFailure(stderr, err.Error(), ExitBuildFailure)
	}
	buildArgs := []string{"--release"}
	if seen["--locked"] {
		buildArgs = append(buildArgs, "--locked")
	}
	executable, code := runStableBuild(buildArgs, root, io.Discard, stderr)
	if code != ExitSuccess {
		return code
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return internalError(stderr)
	}
	data, err := runDesktopHelper(python, root, "package_release.py", "--exe", filepath.Join(absolute, filepath.FromSlash(executable)), "--output", filepath.Join(absolute, "dist"))
	if err != nil {
		return desktopPackageFailure(stderr, err.Error(), ExitBuildFailure)
	}
	report, err := parseDesktopPackageReport(data, root)
	if err != nil {
		return desktopPackageFailure(stderr, err.Error(), ExitBuildFailure)
	}
	if seen["--json"] {
		if err = json.NewEncoder(stdout).Encode(report); err != nil {
			return outputError(stderr)
		}
	} else {
		if _, err = fmt.Fprintf(stdout, "packaged: %s (unsigned, release, local only)\n", report.Archive); err != nil {
			return outputError(stderr)
		}
	}
	return ExitSuccess
}
func desktopPackageFailure(stderr io.Writer, message string, code int) int {
	if _, err := fmt.Fprintf(stderr, "fairyc: %s\n", message); err != nil {
		return ExitInternalFailure
	}
	return code
}
