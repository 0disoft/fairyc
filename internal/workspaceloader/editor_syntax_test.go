package workspaceloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEditorSyntaxReloadReparsesOnlyChangedLocalFiles(t *testing.T) {
	root := t.TempDir()
	writePackage(t, root, ".", rootManifest(""), map[string]string{
		"src/main.fy":   "module main; import answer as answer; fn i32 main() { return answer::value(); }\n",
		"src/answer.fy": "module answer; pub fn i32 value() { return 1; }\n",
		"src/other.fy":  "module other; fn i32 unused() { return 3; }\n",
	})
	ctx := context.Background()
	before, previous, err := LoadEditorSyntaxContext(ctx, root, nil, nil)
	if err != nil || len(previous.trees) != 3 || previous.Weight() <= 0 {
		t.Fatalf("initial syntax snapshot: %v", err)
	}
	answer := filepath.Join(root, "src", "answer.fy")
	overlay := map[string][]byte{answer: []byte("module answer; pub fn i32 value() { return 2; }\n")}
	reloaded, next, err := LoadEditorSyntaxContext(ctx, root, overlay, previous)
	if err != nil {
		t.Fatal(err)
	}
	hits, misses := 0, 0
	for key, tree := range next.trees {
		if tree == previous.trees[key] {
			hits++
		} else {
			misses++
			if key.absolute != answer {
				t.Fatalf("unchanged file reparsed: %s", key.relative)
			}
		}
	}
	if hits != 2 || misses != 1 || before.Packages[0].Digest == reloaded.Packages[0].Digest {
		t.Fatalf("reload: hits=%d misses=%d changed digest=%v", hits, misses, before.Packages[0].Digest != reloaded.Packages[0].Digest)
	}
	fresh, err := LoadEditorOverlaysContext(ctx, root, overlay)
	if err != nil || !reflect.DeepEqual(reloaded, fresh) {
		t.Fatalf("reused syntax did not lower to the same workspace as a fresh parse: %v", err)
	}
	for i, left := range before.Packages[0].Sources {
		if left.Module == reloaded.Packages[0].Sources[i].Module {
			t.Fatalf("semantic input retained mutable AST for %s", left.Path)
		}
	}
	// A same-length disk replacement must miss even without an editor overlay.
	if err := os.WriteFile(answer, []byte("module answer; pub fn i32 value() { return 9; }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, disk, err := LoadEditorSyntaxContext(ctx, root, nil, next)
	key := editorSyntaxKey{answer, "answer.fy"}
	if err != nil || disk.trees[key].tree == nil || disk.trees[key].tree == next.trees[key].tree {
		t.Fatalf("same-length on-disk change reused syntax: %v", err)
	}
	overlay[answer] = []byte("module answer; pub fn i32 value() { return 2 }")
	_, bad, err := LoadEditorSyntaxContext(ctx, root, overlay, next)
	var load *LoadError
	if !errors.As(err, &load) || load.Reason != ReasonSourceSyntax || bad != nil {
		t.Fatalf("malformed overlay retained syntax or bypassed checks: %v", err)
	}
}
