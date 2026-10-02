package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func taskManagerWorkspace(t *testing.T) string {
	t.Helper()
	workspace := t.TempDir()
	example := filepath.Join("..", "..", "examples", "task_manager_cli")
	entries, err := os.ReadDir(filepath.Join(example, "src"))
	if err != nil {
		t.Fatal(err)
	}
	copySource := func(from, to string) {
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, workspace, to, string(data))
	}
	copySource(filepath.Join(example, "fairy.toml"), "fairy.toml")
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".fy") {
			copySource(filepath.Join(example, "src", entry.Name()), filepath.Join("src", entry.Name()))
		}
	}
	return workspace
}

func taskManagerUnit(t *testing.T, targetID target.ID) (cbackend.Unit, string) {
	t.Helper()
	workspace := taskManagerWorkspace(t)
	var diagnostics strings.Builder
	lowered, code := lowerWorkspaceExecutable(workspace, false, workspaceCompilePolicy{stableDiagnostics: true}, "portable task manager", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: %d %s", code, diagnostics.String())
	}
	profile := cbackend.Stage0Profile()
	if targetID == target.X8664WindowsMSVC {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, profile)
	if err != nil {
		t.Fatal(err)
	}
	var support string
	if targetID == target.X8664WindowsMSVC {
		support, err = windowsVectorSupportSource(unit.ForeignSymbols, false)
	} else {
		support, err = linuxVectorSupportSource(unit.ForeignSymbols)
	}
	if err != nil {
		t.Fatal(err)
	}
	return unit, support
}

func TestTaskManagerPortableEmission(t *testing.T) {
	for _, id := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		t.Run(string(id), func(t *testing.T) {
			unit, support := taskManagerUnit(t, id)
			if id == target.X8664LinuxGNU {
				if strings.Contains(support, "#include <windows.h>") || !strings.Contains(unit.Source, "int main(int argc, char **argv)") || !strings.Contains(support, "fy_std_file_replace_v0") {
					t.Fatal("Linux source/linkage boundary drift")
				}
			} else if !strings.Contains(support, "fy_std_windows_file_replace_v0") {
				t.Fatal("Windows adapter missing")
			}
		})
	}
}

func TestTaskManagerPortableNativeSmoke(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("native lane requires Windows MSVC or Linux GNU amd64")
	}
	id := target.X8664LinuxGNU
	if runtime.GOOS == "windows" {
		id = target.X8664WindowsMSVC
	}
	unit, support := taskManagerUnit(t, id)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			var image []byte
			if runtime.GOOS == "windows" {
				toolchain, err := cdriver.ResolveWindowsHostMSVC()
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				image = artifact.Bytes
			} else {
				toolchain, err := cdriver.Resolve(id, "gcc")
				if err != nil {
					t.Fatal(err)
				}
				artifact, err := cdriver.Build(toolchain, profile, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				image = artifact.Bytes
			}
			root := t.TempDir()
			exe := filepath.Join(root, "task-manager")
			if runtime.GOOS == "windows" {
				exe += ".exe"
			}
			if err := os.WriteFile(exe, image, 0700); err != nil {
				t.Fatal(err)
			}
			taskManagerCases(t, exe, root)
		})
	}
}
