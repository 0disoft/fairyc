package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStableNativeBuildFailurePreservesImage(t *testing.T) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("requires Windows MSVC or Linux GNU amd64")
	}
	root := t.TempDir()
	writeNativeWorkspace(t, root, true)
	if code, _, diag := runForTest([]string{"build"}, root); code != ExitSuccess {
		t.Fatalf("prepare valid image: %d %s", code, diag)
	}
	name := "app"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	assertStableInvalidBuildPreservesImage(t, root, filepath.Join(root, "build", name))
}

// Source digests are lock inputs: stale --locked fails before semantic analysis.
// Unlocked mode refreshes the lock and reports the actual source diagnostic.
// Neither failure may replace an already valid executable.
func assertStableInvalidBuildPreservesImage(t *testing.T, root, image string) {
	t.Helper()
	before, err := os.ReadFile(image)
	if err != nil || len(before) == 0 {
		t.Fatalf("read valid image: %v", err)
	}
	lockPath := filepath.Join(root, "fairy.lock")
	lockBefore, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/main.fy", "module main;\nfn i32 main() { return missing; }\n")
	code, out, diag := runForTest([]string{"build", "--locked"}, root)
	if code != ExitPackageFailure || out != "" || !strings.Contains(diag, "lockfile is stale") {
		t.Fatalf("stale locked build: %d stdout=%q stderr=%q", code, out, diag)
	}
	practicalFileEquals(t, image, string(before))
	practicalFileEquals(t, lockPath, string(lockBefore))
	code, out, diag = runForTest([]string{"build"}, root)
	if code != ExitSourceDiagnostics || out != "" || !strings.Contains(diag, "E0101") {
		t.Fatalf("unlocked invalid source build: %d stdout=%q stderr=%q", code, out, diag)
	}
	practicalFileEquals(t, image, string(before))
}
