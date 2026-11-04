package sourceedit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestSnapshotPreservesFingerprintAndRefreshesInputs(t *testing.T) {
	root := t.TempDir()
	editFile(t, root, "fairy.lock", "lock")
	editFile(t, root, "fairy.toml", "root")
	editFile(t, root, "libs/a/fairy.toml", "member")
	w := workspaceloader.Workspace{Root: root, Packages: []workspaceloader.Package{
		{Path: "libs/a", SourceKind: workspaceloader.SourceKindLocal, SourceRoot: filepath.Join(root, "libs", "a", "src"), Sources: []workspaceloader.SourceFile{{Path: "a.fy", Bytes: []byte("member source")}}},
		{Path: ".", SourceKind: workspaceloader.SourceKindLocal, SourceRoot: filepath.Join(root, "src"), Sources: []workspaceloader.SourceFile{{Path: "main.fy", Bytes: []byte("root source")}}},
		{Path: "sdk", SourceKind: workspaceloader.SourceKindSDK},
	}}
	// The existing sorted path/NUL/raw-byte-digest/NUL fingerprint is unchanged,
	// including loaded source bytes and exclusion of bundled SDK inputs.
	want := Digest([]byte("fairy.lock\x00" + Digest([]byte("lock")) + "\x00" +
		"fairy.toml\x00" + Digest([]byte("root")) + "\x00" +
		"libs/a/fairy.toml\x00" + Digest([]byte("member")) + "\x00" +
		"libs/a/src/a.fy\x00" + Digest([]byte("member source")) + "\x00" +
		"src/main.fy\x00" + Digest([]byte("root source")) + "\x00"))
	got, err := snapshot(w)
	if err != nil || got != want {
		t.Fatalf("fingerprint: %s != %s, %v", got, want, err)
	}
	for _, name := range []string{"libs/a/fairy.toml", "fairy.lock"} {
		editFile(t, root, name, "changed")
		next, err := snapshot(w)
		if err != nil || next == got {
			t.Fatalf("%s stale fingerprint: %s, %v", name, next, err)
		}
		got = next
	}
	// A case-only rename also proves freshness on a case-insensitive host, where
	// Lstat alone still resolves the old spelling to the same directory.
	if err := os.Rename(filepath.Join(root, "libs", "a"), filepath.Join(root, "libs", "A")); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot(w); err == nil {
		t.Fatal("later snapshot reused removed directory spelling")
	}
}
