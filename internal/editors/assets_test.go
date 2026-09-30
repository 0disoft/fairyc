package editors

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetsMatchCompilerVocabulary(t *testing.T) {
	root := filepath.Join("..", "..")
	generated, err := Assets(root)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range generated {
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, want) {
			t.Fatalf("%s is stale: regenerate editor assets", name)
		}
	}
}
