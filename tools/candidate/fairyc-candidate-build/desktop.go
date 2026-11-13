package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/0disoft/fairyc/internal/desktopapp"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/projectscaffold"
)

// This sidecar only extracts assets and runs project-owned resource/archive
// tools. Fairy source analysis and release C generation stay in the candidate.
func desktopError(stderr io.Writer, message string, code int) int {
	if _, err := fmt.Fprintf(stderr, "fairyc: %s\n", message); err != nil {
		return 4
	}
	return code
}

func desktopReady(root string, stderr io.Writer) (desktopapp.Python, int) {
	declared, err := desktopapp.Declared(root)
	if err != nil {
		return desktopapp.Python{}, desktopError(stderr, "invalid app.toml/fairy.windows.json", 7)
	}
	if !declared {
		return desktopapp.Python{}, desktopError(stderr, "package requires a Win32 desktop project with app.toml and fairy.windows.json", 2)
	}
	if _, err := desktopapp.Helper(root, "package_release.py"); err != nil {
		return desktopapp.Python{}, desktopError(stderr, "package_release.py is missing or outside the project", 7)
	}
	python, err := desktopapp.FindPython()
	if err != nil {
		return python, desktopError(stderr, err.Error(), 6)
	}
	return python, 0
}

func runDesktop(args []string, root string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return desktopError(stderr, "desktop operation is required", 2)
	}
	switch args[0] {
	case "new":
		if len(args) != 3 {
			return desktopError(stderr, "desktop new expects name and UID", 2)
		}
		spec := projectscaffold.Spec{Name: args[1], UID: args[2], Template: "win32"}
		if err := projectscaffold.Validate(spec); err != nil {
			return desktopError(stderr, "command \"new\": "+err.Error(), 2)
		}
		if err := projectscaffold.Create(root, spec); err != nil {
			return desktopError(stderr, "project creation failed: "+err.Error(), 7)
		}
		if _, err := fmt.Fprintf(stdout, "created Fairy C package %q at %s\n", spec.Name, spec.Name); err != nil {
			return 4
		}
		return 0
	case "prepare":
		if len(args) != 1 {
			return desktopError(stderr, "desktop prepare does not accept arguments", 2)
		}
		if err := desktopapp.PrepareBuild(root); err != nil {
			return desktopError(stderr, err.Error(), 6)
		}
		return 0
	case "validate", "package":
		asJSON := len(args) == 2 && args[1] == "--json"
		if len(args) != 1 && !(args[0] == "package" && asJSON) {
			return desktopError(stderr, "invalid desktop packaging arguments", 2)
		}
		if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
			return desktopError(stderr, "package currently supports Windows amd64 desktop projects only", 2)
		}
		python, code := desktopReady(root, stderr)
		if code != 0 || args[0] == "validate" {
			return code
		}
		return desktopPackage(root, python, asJSON, stdout, stderr)
	default:
		return desktopError(stderr, "unknown desktop operation", 2)
	}
}

func desktopPackage(root string, python desktopapp.Python, asJSON bool, stdout, stderr io.Writer) int {
	root, err := filepath.Abs(root)
	if err != nil {
		return 4
	}
	data, err := os.ReadFile(filepath.Join(root, "fairy.toml"))
	if err != nil {
		return desktopError(stderr, "cannot read package manifest", 7)
	}
	manifest, err := projectmanifest.Parse(data)
	if err != nil {
		return desktopError(stderr, "invalid package manifest", 7)
	}
	data, err = desktopapp.RunHelper(python, root, "package_release.py", "--exe", filepath.Join(root, "build", manifest.Package.Name+".exe"), "--output", filepath.Join(root, "dist"))
	if err != nil {
		return desktopError(stderr, err.Error(), 6)
	}
	report, err := desktopapp.ParseReport(data, root)
	if err != nil {
		return desktopError(stderr, err.Error(), 6)
	}
	if asJSON {
		err = json.NewEncoder(stdout).Encode(report)
	} else {
		_, err = fmt.Fprintf(stdout, "packaged: %s (unsigned, release, local only)\n", report.Archive)
	}
	if err != nil {
		return desktopError(stderr, "output write failed", 4)
	}
	return 0
}
