package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestNestedCompactTagSourceRunsWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("nested compact-tag native proof requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("nested compact-tag native proof requires local x64 MSVC: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "nested_compact_tag.fy"), []byte(nestedCompactTagSource), 0o600); err != nil {
		t.Fatalf("write nested compact-tag source: %v", err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(root, "nested compact tag "+profile, "nested_compact_tag.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			t.Fatal(err)
		}
	}
}
