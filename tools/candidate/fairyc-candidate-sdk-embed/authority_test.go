package main

import (
	"path/filepath"
	"testing"
)

func TestCheckedInAuthorityMatchesBundledSDK(t *testing.T) {
	if err := checkAuthority(filepath.Join("..", "..", "..", "examples", "lexer_candidate", "src")); err != nil {
		t.Fatal(err)
	}
}
