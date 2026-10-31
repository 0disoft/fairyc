//go:build windows && amd64

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const retainedDebugFairy = `module main;
@link_name("fairyc_debug_boundary")
unsafe extern C fn i32 boundary(i32 value);
fn i32 main() {
    let i32 value = unsafe { boundary(40) };
    assert(value == 42);
    return value;
}
`

func buildRetainedDebugFixture(t *testing.T, tc cdriver.WindowsHostToolchain, profile cdriver.Profile) (cdriver.WindowsHostArtifact, windowsDebugManifest, map[string][]byte) {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(retainedDebugFairy)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("debug fixture parse: %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		t.Fatalf("debug fixture check: %v", err)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	runtime += "\n__declspec(noinline) int32_t fairyc_debug_boundary(int32_t value) {\n    volatile int32_t result = value + 2;\n    return result;\n}\n"
	artifact, err := cdriver.BuildWindowsHostWithDebugMap(tc, profile, unit.Source, runtime)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "unit.obj" || artifact.Objects[1].Name != "runtime.obj" {
		t.Fatal("missing debug object receipts")
	}
	revision, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	manifest := windowsDebugManifest{SourceRevision: strings.TrimSpace(string(revision)), Profile: profile.Name, Toolchain: tc.Version}
	files := map[string][]byte{
		"program.exe": artifact.Bytes, "program.pdb": artifact.ProgramDatabase,
		"main.fy": []byte(retainedDebugFairy), "unit.c": []byte(unit.Source), "runtime.c": []byte(runtime),
		"unit.obj": artifact.Objects[0].Bytes, "runtime.obj": artifact.Objects[1].Bytes, "program.map": artifact.LinkMap,
	}
	return artifact, manifest, files
}

func TestWindowsDebugStore(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, manifest, files := buildRetainedDebugFixture(t, tc, profile)
			outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("debug image: %+v %v", outcome, err)
			}
			store := t.TempDir()
			key, err := retainWindowsDebugBundle(store, manifest, files)
			if err != nil {
				t.Fatal(err)
			}
			if again, err := retainWindowsDebugBundle(store, manifest, files); err != nil || again != key {
				t.Fatalf("idempotent retain: %s %v", again, err)
			}
			loaded, restored, err := loadWindowsDebugBundle(store, key)
			if err != nil || len(restored) != 8 {
				t.Fatalf("reload: %v", err)
			}
			if loaded.SourceRevision != manifest.SourceRevision || loaded.Age == 0 {
				t.Fatal("lost debug identity")
			}
			manifest.Toolchain += "-changed"
			if _, err := retainWindowsDebugBundle(store, manifest, files); err == nil {
				t.Fatal("metadata replacement accepted")
			}
			for _, name := range []string{"program.exe", "program.pdb", "unit.c", "runtime.c", "unit.obj", "runtime.obj"} {
				t.Run("corrupt-"+name, func(t *testing.T) {
					path := filepath.Join(store, key, name)
					bad := append([]byte(nil), files[name]...)
					bad[0] ^= 1
					if err := os.WriteFile(path, bad, 0600); err != nil {
						t.Fatal(err)
					}
					if _, _, err := loadWindowsDebugBundle(store, key); err == nil {
						t.Fatal("corrupt debug file accepted")
					}
					if err := os.WriteFile(path, files[name], 0600); err != nil {
						t.Fatal(err)
					}
				})
			}
			path := filepath.Join(store, key, "manifest.json")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"guidRawBytes", "age", "schema", "sourceRevision", "files"} {
				t.Run("manifest-"+field, func(t *testing.T) {
					var value map[string]any
					if err := json.Unmarshal(original, &value); err != nil {
						t.Fatal(err)
					}
					delete(value, field)
					bad, _ := json.Marshal(value)
					if err := os.WriteFile(path, bad, 0600); err != nil {
						t.Fatal(err)
					}
					if _, _, err := loadWindowsDebugBundle(store, key); err == nil {
						t.Fatal("incomplete manifest accepted")
					}
					if err := os.WriteFile(path, original, 0600); err != nil {
						t.Fatal(err)
					}
				})
			}
			if err := os.Remove(filepath.Join(store, key, "program.pdb")); err != nil {
				t.Fatal(err)
			}
			if _, _, err := loadWindowsDebugBundle(store, key); err == nil {
				t.Fatal("missing PDB accepted")
			}
			for _, invalid := range []string{"../escape", key[:12], "", "C:\\program.pdb"} {
				if _, _, err := loadWindowsDebugBundle(store, invalid); err == nil {
					t.Fatal("invalid lookup key accepted")
				}
			}
			t.Logf("%s native exit=42, retained key=%s, PDB=%d bytes; idempotence and replacement/corruption/missing-file/key rejections passed", profile.Name, key, len(files["program.pdb"]))
		})
	}
}
