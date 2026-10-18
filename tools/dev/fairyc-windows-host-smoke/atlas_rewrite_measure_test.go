package main

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestMeasureAtlasRewriteProducesDeterministicWholePackageDistribution(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "examples", "workspace_quickstart")
	clockResolution, err := observeMonotonicClockResolution()
	if err != nil {
		t.Fatal(err)
	}
	report, err := measureAtlasRewrite(root, clockResolution)
	if err != nil {
		t.Fatal(err)
	}
	if err := buildmeasure.ValidateAtlasRewrite(report); err != nil {
		t.Fatal(err)
	}
	if report.Shape.Packages != 2 || report.Shape.SourceFiles != 2 || report.Shape.SourceBytes == 0 || report.Shape.MapRecords == 0 {
		t.Fatalf("Atlas rewrite shape = %+v", report.Shape)
	}
}
