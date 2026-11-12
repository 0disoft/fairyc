package cli

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Hooks exist only in the owned integration workspace, never the shipped EXE.
func prepareWindowsHashSlowIO(t *testing.T, root string) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(root, "native/reader.c"))
	if err != nil {
		t.Fatal(err)
	}
	hook, err := os.ReadFile("../../examples/win32_hash/testing/delayed_reader.c")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, pair := range [][2]string{
		{"value->file = CreateFileW(", "value->file = delayed_open("},
		{"ReadFile(value->file,", "delayed_read(value->file,"},
		{"GetFileInformationByHandleEx(file, FileIdInfo,", "delayed_metadata(file, FileIdInfo,"},
	} {
		if strings.Count(text, pair[0]) != 1 {
			t.Fatalf("slow-I/O hook no longer matches transport: %s", pair[0])
		}
		text = strings.Replace(text, pair[0], pair[1], 1)
	}
	writeWorkspaceFile(t, root, "native/reader.c", string(hook)+"\n"+text)
	t.Setenv("FAIRY_HASH_TEST_PIPE", `\\.\pipe\fairy-hash-`+rand.Text())
}
