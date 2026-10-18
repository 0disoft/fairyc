package main

import (
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestMeasureAtlasCallRetainedHeapProducesClosedPositiveBatches(t *testing.T) {
	report, err := measureAtlasCallRetainedHeap()
	if err != nil {
		t.Fatal(err)
	}
	if err := buildmeasure.ValidateAtlasCallRetained(report); err != nil {
		t.Fatal(err)
	}
	if len(report.Shapes) != 3 {
		t.Fatalf("Atlas call retained-heap shapes = %+v", report.Shapes)
	}
	for _, shape := range report.Shapes {
		if shape.IndexesPerSample != buildmeasure.AtlasCallRetainedIndexesPerSample ||
			len(shape.Samples) != buildmeasure.AtlasCallRetainedIterationsPerShape {
			t.Fatalf("Atlas call retained-heap shape = %+v", shape)
		}
	}
}
