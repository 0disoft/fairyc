package main

import (
	"bytes"
	"fmt"
	"runtime"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/buildmeasure"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func measureAtlasRewrite(root string, clockResolution int64) (buildmeasure.AtlasRewriteReport, error) {
	const maxMapBytesPerPackage = uint64(64 << 20)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	if len(checked.Semantic.Diagnostics) != 0 || len(checked.Semantic.Unsupported) != 0 || !checked.Verified {
		return buildmeasure.AtlasRewriteReport{}, fmt.Errorf("Atlas rewrite workload did not produce clean verified MIR")
	}
	compilerBuild, err := buildidentity.Current()
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, checked, compilerBuild)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	shape, err := atlasRewriteShape(workspace, partitions)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	warmup, _, err := encodeAtlasPartitions(partitions, maxMapBytesPerPackage)
	if err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	report := buildmeasure.AtlasRewriteReport{
		Schema: buildmeasure.AtlasRewriteSchema, Workload: "workspace_quickstart",
		Scope: "canonical-fymap-encode-all-packages;workspace-analysis-map-planning-and-publication-excluded",
		Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: clockResolution,
		Decision: "raw-distribution-only-no-format-or-incremental-policy",
		Setup:    "one-untimed-warmup;fixed-preplanned-partitions;gc-uncontrolled", Iterations: buildmeasure.AtlasRewriteIterations,
		CompilerBuild: compilerBuild.String(), LogicalSchema: atlasmap.LogicalSchema,
		TransportSchema: atlasmap.TransportSchema, TransportEncoding: atlasmap.TransportEncoding,
		MaxMapBytesPerPackage: maxMapBytesPerPackage,
		Environment:           buildmeasure.AtlasRewriteEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH},
		Shape:                 shape,
	}
	for index := 0; index < buildmeasure.AtlasRewriteIterations; index++ {
		started := time.Now()
		encoded, encodedBytes, encodeErr := encodeAtlasPartitions(partitions, maxMapBytesPerPackage)
		duration := time.Since(started).Nanoseconds()
		if encodeErr != nil {
			return buildmeasure.AtlasRewriteReport{}, encodeErr
		}
		if !equalAtlasEncodings(warmup, encoded) {
			return buildmeasure.AtlasRewriteReport{}, fmt.Errorf("Atlas rewrite sample %d changed canonical map bytes", index+1)
		}
		report.Samples = append(report.Samples, buildmeasure.AtlasRewriteSample{
			Sequence: index + 1, DurationNS: duration, BelowClockResolution: duration < clockResolution,
			Packages: shape.Packages, MapRecords: shape.MapRecords, EncodedBytes: encodedBytes,
		})
	}
	if err := buildmeasure.ValidateAtlasRewrite(report); err != nil {
		return buildmeasure.AtlasRewriteReport{}, err
	}
	return report, nil
}

func atlasRewriteShape(workspace workspaceloader.Workspace, partitions []atlasmap.Partition) (buildmeasure.AtlasRewriteShape, error) {
	shape := buildmeasure.AtlasRewriteShape{Packages: len(partitions)}
	for _, loaded := range workspace.Packages {
		for _, file := range loaded.Sources {
			shape.SourceFiles++
			if uint64(len(file.Bytes)) > ^uint64(0)-shape.SourceBytes {
				return buildmeasure.AtlasRewriteShape{}, fmt.Errorf("Atlas rewrite source-byte count overflow")
			}
			shape.SourceBytes += uint64(len(file.Bytes))
		}
	}
	for _, partition := range partitions {
		counts := partition.Header().Counts
		values := []int{counts.Files, counts.Modules, counts.Symbols, counts.Functions, counts.Relations, counts.Completeness}
		for _, value := range values {
			if value < 0 || uint64(value) > ^uint64(0)-shape.MapRecords {
				return buildmeasure.AtlasRewriteShape{}, fmt.Errorf("Atlas rewrite record count overflow")
			}
			shape.MapRecords += uint64(value)
		}
	}
	return shape, nil
}

func encodeAtlasPartitions(partitions []atlasmap.Partition, maxBytes uint64) ([][]byte, uint64, error) {
	result := make([][]byte, 0, len(partitions))
	var total uint64
	for _, partition := range partitions {
		encoded, err := atlasmap.Encode(partition, maxBytes)
		if err != nil {
			return nil, 0, err
		}
		if uint64(len(encoded)) > ^uint64(0)-total {
			return nil, 0, fmt.Errorf("Atlas rewrite encoded-byte count overflow")
		}
		total += uint64(len(encoded))
		result = append(result, encoded)
	}
	return result, total, nil
}

func equalAtlasEncodings(left, right [][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !bytes.Equal(left[index], right[index]) {
			return false
		}
	}
	return true
}
