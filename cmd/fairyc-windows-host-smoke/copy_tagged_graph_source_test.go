package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestCopyTaggedGraphSourceRunsWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("copy tagged graph native proof requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("copy tagged graph native proof requires local x64 MSVC: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "copy_tagged_graph.fy"), []byte(copyTaggedGraphSource), 0o600); err != nil {
		t.Fatalf("write copy tagged graph source: %v", err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(root, "copy tagged graph "+profile, "copy_tagged_graph.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			t.Fatal(err)
		}
	}
}
