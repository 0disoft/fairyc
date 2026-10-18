package main

import (
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestMeasureAtlasCallScaleProducesClosedGrowingShapes(t *testing.T) {
	clockResolution, err := observeMonotonicClockResolution()
	if err != nil {
		t.Fatal(err)
	}
	report, err := measureAtlasCallScale(clockResolution)
	if err != nil {
		t.Fatal(err)
	}
	if err := buildmeasure.ValidateAtlasCallScale(report); err != nil {
		t.Fatal(err)
	}
	if len(report.Shapes) != 3 || report.Shapes[0].Packages != 2 || report.Shapes[0].Calls != 8 ||
		report.Shapes[1].Packages != 8 || report.Shapes[1].Calls != 64 ||
		report.Shapes[2].Packages != 32 || report.Shapes[2].Calls != 512 {
		t.Fatalf("Atlas call scale shapes = %+v", report.Shapes)
	}
}
