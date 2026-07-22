//go:build windows

package packagecache

import (
	"strings"
	"testing"
)

func TestWindowsExtendedPathUsesVerbatimAbsoluteForm(t *testing.T) {
	path, err := windowsExtendedPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, `\\?\`) {
		t.Fatalf("windows extended path = %q", path)
	}
}
