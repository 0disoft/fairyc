package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestCopyScalarTaggedAggregateSourceRunsWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("copy-scalar tagged aggregate native proof requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("copy-scalar tagged aggregate native proof requires local x64 MSVC: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "copy_scalar_tagged_aggregate.fy"), []byte(copyScalarTaggedAggregateSource), 0o600); err != nil {
		t.Fatalf("write copy-scalar tagged aggregate source: %v", err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(
			root,
			"copy-scalar tagged aggregate fields "+profile,
			"copy_scalar_tagged_aggregate.fy",
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
