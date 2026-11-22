package languageserver

import (
	"path/filepath"
	"testing"
)

func TestWindowsEditorPathsRequireDriveVolumes(t *testing.T) {
	local := filepath.Join(t.TempDir(), "source.fy")
	if got, err := filePath(fileURI(local)); err != nil || got != local {
		t.Fatalf("local document path: %q, %v", got, err)
	}
	if roots, err := (initializeParams{RootPath: filepath.Dir(local)}).roots(); err != nil ||
		len(roots) != 1 || roots[0] != filepath.Dir(local) {
		t.Fatalf("local workspace root: %v, %v", roots, err)
	}

	for _, path := range []string{
		`\\host.invalid\share\source.fy`,
		`\\.\namespace\source.fy`,
	} {
		t.Run(filepath.VolumeName(path), func(t *testing.T) {
			// Check admission without opening or otherwise accessing the path.
			if _, err := filePath(fileURI(path)); err == nil {
				t.Fatal("non-drive document path admitted")
			}
			root := filepath.Dir(path)
			if _, err := (initializeParams{RootPath: root}).roots(); err == nil {
				t.Fatal("non-drive legacy workspace root admitted")
			}
			if _, err := (initializeParams{RootURI: fileURI(root)}).roots(); err == nil {
				t.Fatal("non-drive workspace URI admitted")
			}
		})
	}
}
