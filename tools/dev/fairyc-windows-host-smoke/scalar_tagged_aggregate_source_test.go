package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestScalarTaggedAggregateSourceRunsWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("scalar tagged aggregate native proof requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("scalar tagged aggregate native proof requires local x64 MSVC: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "scalar_tagged_aggregate.fy"), []byte(scalarTaggedAggregateSource), 0o600); err != nil {
		t.Fatalf("write scalar tagged aggregate source: %v", err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(
			root,
			"scalar tagged aggregate fields "+profile,
			"scalar_tagged_aggregate.fy",
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
