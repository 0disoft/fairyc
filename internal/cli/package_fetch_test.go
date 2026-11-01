package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
)

func remoteCLIWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	member := `schema="fairyc.manifest/v1"
[package]
uid="example.math"
name="math"
version="1.0.0"
language="fairyc-language/20"
source="src"
`
	manifest := strings.Replace(member, `uid="example.math"`, `uid="example.app"`, 1) + "[workspace]\nmembers=[\"vendor/math\"]\n[dependencies.math]\npath=\"vendor/math\"\nversion=\"1.0.0\"\nfeatures=[]\n"
	writeWorkspaceFile(t, root, "fairy.toml", manifest)
	writeWorkspaceFile(t, root, "src/main.fy", "module main;import math as m;fn i32 main(){return m::value();}\n")
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for name, body := range map[string]string{"fairy.toml": member, "src/math.fy": "module math;pub fn i32 value(){return 42;}\n"} {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(out.Bytes())
	digest := hex.EncodeToString(sum[:])
	plan := packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.example/math.zip", SHA256: digest}}}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	writeWorkspaceFile(t, root, packagesource.FileName, string(data))
	writeWorkspaceFile(t, root, packagesource.CachePath+"/"+digest+".zip", out.String())
	return root
}

func TestRemotePackageCLIOfflineLockAndFailures(t *testing.T) {
	root := remoteCLIWorkspace(t)
	for _, install := range []int{1, 0} {
		code, out, err := runForTest([]string{"experimental", "fetch-packages", "--offline"}, root)
		if code != 0 || err != "" || !strings.Contains(out, "1 cache hit(s)") {
			t.Fatalf("fetch: %d %q %q", code, out, err)
		}
		if install == 1 && !strings.Contains(out, "1 installed") {
			t.Fatal(out)
		}
		if install == 0 && !strings.Contains(out, "1 reused") {
			t.Fatal(out)
		}
	}
	for _, args := range [][]string{{"check"}, {"check", "--locked"}} {
		code, out, err := runForTest(args, root)
		if code != 0 || err != "" {
			t.Fatalf("check: %d %q %q", code, out, err)
		}
	}
	lockBefore, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, stderr := runForTest([]string{"experimental", "fetch-packages", "--offline"}, root)
	if code != 0 || stderr != "" {
		t.Fatalf("fetch locked tree: %d %q %q", code, out, stderr)
	}
	lockAfter, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatal("fetch changed graph lock")
	}
	writeWorkspaceFile(t, root, "vendor/math/src/math.fy", "module math;pub fn i32 value(){return 41;}\n")
	code, out, stderr = runForTest([]string{"experimental", "fetch-packages", "--offline"}, root)
	if code != ExitPackageFailure || out != "" || !strings.Contains(stderr, "differs from pinned archive") {
		t.Fatalf("conflict: %d %q %q", code, out, stderr)
	}
	code, _, _ = runForTest([]string{"check", "--locked"}, root)
	if code != ExitPackageFailure {
		t.Fatal("old graph lock accepted changed dependency")
	}
	for _, args := range [][]string{{"--locked"}, {"--offline", "--offline"}, {"--force"}} {
		code, out, _ := runForTest(append([]string{"experimental", "fetch-packages"}, args...), t.TempDir())
		if code != ExitInvalidInvocation || out != "" {
			t.Fatal("invalid fetch invocation accepted")
		}
	}
	code, out, _ = runForTest([]string{"experimental", "fetch-packages", "--offline"}, t.TempDir())
	if code != ExitPackageFailure || out != "" {
		t.Fatal("missing source pins accepted")
	}
}

func TestRemotePackageCLINative(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("existing Windows/Linux amd64 native hosts only")
	}
	if runtime.GOOS == "windows" {
		if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
			t.Skipf("MSVC unavailable: %v", err)
		}
	}
	root := remoteCLIWorkspace(t)
	for _, args := range [][]string{{"experimental", "fetch-packages", "--offline"}, {"check"}} {
		code, out, err := runForTest(args, root)
		if code != 0 || err != "" {
			t.Fatalf("prepare: %d %q %q", code, out, err)
		}
	}
	commands := [][]string{{"run", "--locked"}, {"run", "--release", "--locked"}}
	if runtime.GOOS == "linux" {
		commands = nil
		for _, cc := range []string{"gcc", "clang"} {
			for _, p := range []string{"low", "optimized"} {
				commands = append(commands, []string{"experimental", "run-workspace", "--target", "x86_64-linux-gnu", "--cc", cc, "--profile", p, "--locked"})
			}
		}
	}
	for _, args := range commands {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			code, out, err := runForTest(args, root)
			if code != 42 || out != "" || err != "" {
				t.Fatalf("native dependency: %d %q %q", code, out, err)
			}
		})
	}
	var stderr bytes.Buffer
	if code := Run([]string{"experimental", "fetch-packages", "--offline"}, root, failingWriter{}, &stderr); code != ExitInternalFailure {
		t.Fatal("fetch output failure hidden")
	}
}
