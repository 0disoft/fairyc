package cbackend

import (
	"context"
	"testing"
)

var benchmarkUnit Unit

func benchmarkEmissionPlan(b *testing.B) *emissionAnalysis {
	b.Helper()
	program := fanoutDropProgram(b, 14)
	plan, err := analyzeEmissionContext(context.Background(), program, "main::main", Stage0Profile(), false)
	if err != nil {
		b.Fatal(err)
	}
	return plan
}

func BenchmarkSmallBuildBodyEmission(b *testing.B) {
	plan := benchmarkEmissionPlan(b)
	b.Run("legacy_duplicate", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			measured := &emitter{analysis: plan, partitioned: true}
			if _, err := measured.emitUnitBodies(); err != nil {
				b.Fatal(err)
			}
			unit, err := (&emitter{analysis: plan}).emitUnit()
			if err != nil {
				b.Fatal(err)
			}
			benchmarkUnit = unit
		}
	})
	b.Run("reuse_measured", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			e := &emitter{analysis: plan, partitioned: true}
			bodies, err := e.emitUnitBodies()
			if err != nil {
				b.Fatal(err)
			}
			e.partitioned = false
			unit, err := e.emitUnitFromBodies(bodies)
			if err != nil {
				b.Fatal(err)
			}
			benchmarkUnit = unit
		}
	})
}

func BenchmarkRegularEmitStreaming(b *testing.B) {
	plan := benchmarkEmissionPlan(b)
	b.ReportAllocs()
	for range b.N {
		unit, err := (&emitter{analysis: plan}).emitUnit()
		if err != nil {
			b.Fatal(err)
		}
		benchmarkUnit = unit
	}
}
