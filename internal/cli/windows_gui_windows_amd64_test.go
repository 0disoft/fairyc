package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestWindowsGUIWorkspaceBuild(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, "module main;unsafe extern C fn i32 ui_probe();fn i32 main(){return unsafe{ui_probe()};}")
	writeWorkspaceFile(t, root, "fairy.windows.json", `{"schema":"fairyc.windows-build/v1","subsystem":"windows","libraries":["user32.lib"],"sources":["ui.c"]}`)
	writeWorkspaceFile(t, root, "ui.c", "#include <windows.h>\n#include <stdint.h>\nint32_t ui_probe(void){return GetDesktopWindow()!=NULL ? 42 : 1;}\n")
	for _, release := range []bool{false, true} {
		args := []string{"build"}
		if release {
			args = append(args, "--release")
		}
		code, out, diag := runForTest(args, root)
		if code != 0 || diag != "" {
			t.Fatalf("%v: %d %s %s", args, code, out, diag)
		}
		image, err := os.ReadFile(filepath.Join(root, "build/app.exe"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pecoff.InspectExecutable(image); err == nil {
			t.Fatal("GUI accepted as console")
		}
		if _, err := pecoff.InspectExecutableSubsystem(image, 2); err != nil {
			t.Fatal(err)
		}
		args[0] = "run"
		if code, out, diag = runForTest(args, root); code != 42 || out != "" || diag != "" {
			t.Fatalf("%v: %d %s %s", args, code, out, diag)
		}
	}
	before, err := os.ReadFile(filepath.Join(root, "build/app.exe"))
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, "ui.c", "invalid C code\n")
	if code, _, _ := runForTest([]string{"build"}, root); code != ExitBuildFailure {
		t.Fatalf("bad C exit=%d", code)
	}
	after, err := os.ReadFile(filepath.Join(root, "build/app.exe"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed native build replaced executable", err)
	}
}

func TestWindowsGUIResources(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, "module main;unsafe extern C fn i32 resource_probe();fn i32 main(){return unsafe{resource_probe()};}")
	writeWorkspaceFile(t, root, "fairy.windows.json", `{"schema":"fairyc.windows-build/v1","subsystem":"windows","libraries":["user32.lib"],"sources":["ui.c"],"resources":["app.rc"]}`)
	writeWorkspaceFile(t, root, "app.rc", "#include <windows.h>\nSTRINGTABLE\nBEGIN\n 101 L\"Fairy GUI\"\nEND\n")
	writeWorkspaceFile(t, root, "ui.c", "#include <windows.h>\n#include <stdint.h>\nint32_t resource_probe(void){WCHAR text[32];return GetConsoleWindow()==NULL && LoadStringW(GetModuleHandleW(NULL),101,text,32)==9 ? 42:1;}\n")
	for _, release := range []bool{false, true} {
		args := []string{"run"}
		if release {
			args = append(args, "--release")
		}
		if code, out, diag := runForTest(args, root); code != 42 || out != "" || diag != "" {
			t.Fatalf("resource %v: %d %s %s", args, code, out, diag)
		}
	}
}
