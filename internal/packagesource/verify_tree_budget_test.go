package packagesource

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestVerifyTreeSharedEntryBudget(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, dir := range []string{"package", "package/a", "package/b"} {
		if err := root.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	b := bundle{files: map[string][]byte{"a/one": []byte("1"), "b/two": []byte("2")}}
	for name, data := range b.files {
		if err := root.WriteFile("package/"+name, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Root + two directories + two files: one budget shared by siblings.
	for _, limit := range []int{4, 5, 6} {
		err := verifyTreeWithin(context.Background(), root, "package", b, limit)
		if limit < 5 {
			if err == nil || err.Error() != "existing package entry limit exceeded" {
				t.Fatalf("limit=%d: %v", limit, err)
			}
		} else if err != nil {
			t.Fatalf("limit=%d: %v", limit, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := verifyTreeWithin(ctx, root, "package", b, 5); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	// No recursive directory handles may survive any success/failure return.
	if err := root.RemoveAll("package"); err != nil {
		t.Fatalf("directory handle leaked: %v", err)
	}
}

func TestVerifyTreeWideDirectoryFailsBeforeFileValidation(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir("package", 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one", "two", "three"} {
		if err := root.WriteFile("package/"+name, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Extra files must not turn a partial directory scan into a reuse verdict.
	err = verifyTreeWithin(context.Background(), root, "package", bundle{}, 3)
	if err == nil || err.Error() != "existing package entry limit exceeded" {
		t.Fatalf("oversized directory accepted or validated before admission: %v", err)
	}
}
