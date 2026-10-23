package main

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestFileBufferSDKAllocationFailuresNative(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Skipf("MSVC unavailable: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for name, probe := range map[string]func(string, string, cdriver.WindowsHostToolchain) (int, int, error){
		"read_append": runGeneratedFairyReadAppendFailureProbe,
		"read_to_end": runGeneratedFairyReadToEndFailureProbe,
	} {
		t.Run(name, func(t *testing.T) {
			images, cases, err := probe(root, root, tc)
			if err != nil || images != 2 || cases != 2 {
				t.Fatalf("allocation failure: %d images, %d cases: %v", images, cases, err)
			}
			t.Log("low/optimized: allocation failure, preserved bytes/cursor and zero retained allocations passed")
		})
	}
}
