package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
)

func TestNestedCompactAggregateSourceRunsWithMSVC(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("nested compact aggregate native proof requires Windows amd64")
	}
	if _, err := cdriver.ResolveWindowsHostMSVC(); err != nil {
		t.Skipf("nested compact aggregate native proof requires local x64 MSVC: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "nested_compact_aggregate.fy"), []byte(nestedCompactAggregateSource), 0o600); err != nil {
		t.Fatalf("write nested compact aggregate source: %v", err)
	}
	for _, profile := range []string{"low", "optimized"} {
		if err := runHostCase(root, "nested compact aggregate "+profile, "nested_compact_aggregate.fy", "", profile, cli.ExitSuccess, "native result: 42 (windows/amd64 host-smoke, msvc ", ""); err != nil {
			t.Fatal(err)
		}
	}
}
