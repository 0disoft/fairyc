package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestNestedEnumSourceRunsWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("nested enum native proof requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("nested enum native proof requires local x64 MSVC: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "nested_enum.fy"), []byte(nestedEnumSource), 0o600); err != nil {
		t.Fatalf("write nested enum source: %v", err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(
			root,
			"nested inert enum payload "+profile,
			"nested_enum.fy",
			"",
			profile,
			cli.ExitSuccess,
			"native result: 42 (windows/amd64 host-smoke, msvc ",
			"",
		); err != nil {
			t.Fatal(err)
		}
	}
}
