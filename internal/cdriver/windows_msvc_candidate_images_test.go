package cdriver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestCandidateMSVCBuiltImages(t *testing.T) {
	root := os.Getenv("FAIRYC_MSVC_BUILT_IMAGES")
	if root == "" {
		t.Skip("explicit authored MSVC build proof only")
	}
	paths, err := filepath.Glob(filepath.Join(root, "*.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Fatalf("want four authored/profile image combinations, got %d", len(paths))
	}
	for _, path := range paths {
		image, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pecoff.InspectExecutable(image); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
}
