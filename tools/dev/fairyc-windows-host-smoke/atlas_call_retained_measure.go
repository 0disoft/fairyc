package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func measureAtlasCallRetainedHeap() (buildmeasure.AtlasCallRetainedReport, error) {
	temporaryRoot, err := os.MkdirTemp("", "fairyc-atlas-call-retained-")
	if err != nil {
		return buildmeasure.AtlasCallRetainedReport{}, fmt.Errorf("create Atlas call retained-heap root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return buildmeasure.AtlasCallRetainedReport{}, err
	}
	report := buildmeasure.AtlasCallRetainedReport{
		Schema: buildmeasure.AtlasCallRetainedSchema, Workload: "generated-star-workspace-call-retained-index-batches-v0",
		Scope:              "post-forced-gc-process-live-heap-delta-for-retained-workspace-call-index-batches;prepared-inputs-kept-live;workspace-generation-loading-compilation-and-map-planning-excluded",
		MemoryMetric:       "runtime-memstats-heapalloc-and-heapobjects-post-forced-gc-delta;process-wide;not-atlas-only",
		Setup:              "deterministic-generated-star-workspaces;prepare-and-warm-before-sampling;forced-gc-before-baseline-and-after-batch;first-observations-only",
		Decision:           "retained-heap-control-only-no-per-index-budget-default-traversal-impact-or-regression-claim",
		IterationsPerShape: buildmeasure.AtlasCallRetainedIterationsPerShape,
		CompilerBuild:      compilerBuild.String(), LogicalSchema: atlasmap.LogicalSchema,
		Environment: buildmeasure.AtlasCallScaleEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH},
	}
	for _, definition := range buildmeasure.AtlasCallScaleDefinitions() {
		shapeRoot := filepath.Join(temporaryRoot, definition.Name)
		if err := writeAtlasCallScaleWorkspace(shapeRoot, definition); err != nil {
			return buildmeasure.AtlasCallRetainedReport{}, err
		}
		prepared, err := prepareAtlasCallWorkspace(shapeRoot, atlasCallWorkspaceDefinition{
			Name: definition.Name, Packages: definition.Packages, Calls: definition.Calls,
		}, compilerBuild)
		if err != nil {
			return buildmeasure.AtlasCallRetainedReport{}, err
		}
		warmup, err := buildAtlasCallIndex(prepared)
		if err != nil {
			return buildmeasure.AtlasCallRetainedReport{}, fmt.Errorf("warm Atlas call retained-heap shape %q: %w", definition.Name, err)
		}
		runtime.KeepAlive(warmup)
		warmup = nil
		runtime.GC()
		shape := buildmeasure.AtlasCallRetainedShape{
			Name: definition.Name, Packages: definition.Packages, Calls: definition.Calls,
			MapRecords: prepared.shape.MapRecords, IndexesPerSample: buildmeasure.AtlasCallRetainedIndexesPerSample,
		}
		for sequence := 1; sequence <= buildmeasure.AtlasCallRetainedIterationsPerShape; sequence++ {
			sample, err := measureAtlasCallRetainedSample(prepared, sequence)
			if err != nil {
				return buildmeasure.AtlasCallRetainedReport{}, fmt.Errorf("measure Atlas call retained-heap shape %q sample %d: %w", definition.Name, sequence, err)
			}
			shape.Samples = append(shape.Samples, sample)
		}
		report.Shapes = append(report.Shapes, shape)
	}
	if err := buildmeasure.ValidateAtlasCallRetained(report); err != nil {
		return buildmeasure.AtlasCallRetainedReport{}, err
	}
	return report, nil
}

func measureAtlasCallRetainedSample(
	prepared preparedAtlasCallWorkspace,
	sequence int,
) (buildmeasure.AtlasCallRetainedSample, error) {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	indexes := make([]*atlasmap.WorkspaceCallIndex, 0, buildmeasure.AtlasCallRetainedIndexesPerSample)
	for count := 0; count < buildmeasure.AtlasCallRetainedIndexesPerSample; count++ {
		index, err := buildAtlasCallIndex(prepared)
		if err != nil {
			return buildmeasure.AtlasCallRetainedSample{}, err
		}
		indexes = append(indexes, index)
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(indexes)
	runtime.KeepAlive(prepared)
	if after.HeapAlloc <= before.HeapAlloc || after.HeapObjects <= before.HeapObjects {
		return buildmeasure.AtlasCallRetainedSample{}, fmt.Errorf(
			"post-GC process live heap did not increase: alloc %d -> %d, objects %d -> %d",
			before.HeapAlloc, after.HeapAlloc, before.HeapObjects, after.HeapObjects,
		)
	}
	return buildmeasure.AtlasCallRetainedSample{
		Sequence: sequence, RetainedHeapAllocBytes: after.HeapAlloc - before.HeapAlloc,
		RetainedHeapObjects: after.HeapObjects - before.HeapObjects,
	}, nil
}
