package languageserver

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsDocumentCaseAliasUnderShortWorkspacePath(t *testing.T) {
	root, filename := editorFixture(t)
	name, err := windows.UTF16PtrFromString(root)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, len(root)+1)
	n, err := windows.GetShortPathName(name, &buffer[0], uint32(len(buffer)))
	if err != nil || n >= uint32(len(buffer)) {
		t.Fatalf("get short workspace path: length %d, error %v", n, err)
	}
	shortRoot := windows.UTF16ToString(buffer[:n])
	if strings.EqualFold(root, shortRoot) {
		t.Skip("temporary volume did not generate an 8.3 workspace ancestor")
	}
	relative, err := filepath.Rel(root, filename)
	if err != nil {
		t.Fatal(err)
	}
	checkDocumentCaseAliasOverlay(t, shortRoot, filepath.Join(shortRoot, relative))
}
