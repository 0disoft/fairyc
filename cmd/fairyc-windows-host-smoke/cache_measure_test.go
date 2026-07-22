package main

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestMeasureWorkspaceProcessCacheEditsProducesValidRawDistributions(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "examples", "workspace_quickstart")
	report, err := measureWorkspaceProcessCacheEdits(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := buildmeasure.ValidateCacheEdits(report); err != nil {
		t.Fatal(err)
	}
}
