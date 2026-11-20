package atlasmap

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"
)

type WorkspaceDiffResult struct {
	Schema   string                 `json:"schema"`
	State    string                 `json:"state"`
	Packages []WorkspacePackageDiff `json:"packages"`
}

type WorkspacePackageDiff struct {
	Coordinate         string        `json:"coordinate"`
	Action             string        `json:"action"`
	State              string        `json:"state"`
	Before             *QueryContext `json:"before"`
	After              *QueryContext `json:"after"`
	BeforeDependencies []string      `json:"beforeDependencies"`
	AfterDependencies  []string      `json:"afterDependencies"`
	Changes            []DiffChange  `json:"changes"`
	Omissions          []string      `json:"omissions"`
}

// CompareWorkspaceV3 compares the root-reachable partitions supplied by the
// locked workspace compiler. Budgets cover both snapshots before any body
// fingerprints are requested.
func CompareWorkspaceV3(before, after []Partition, beforeExpected, afterExpected map[string]LoadExpectation,
	maxPackages, maxDeclarations int,
	bodyFingerprints func(Partition, Partition) (map[string][sha256.Size]byte, map[string][sha256.Size]byte),
) (WorkspaceDiffResult, error) {
	if len(before) == 0 || len(after) == 0 || maxPackages < 1 ||
		len(before) > maxPackages || len(after) > maxPackages-len(before) {
		return WorkspaceDiffResult{}, fmt.Errorf("Atlas workspace diff packages exceed limit or snapshot is empty")
	}
	left, err := indexWorkspaceDiff(before, beforeExpected, &maxDeclarations)
	if err != nil {
		return WorkspaceDiffResult{}, fmt.Errorf("Atlas workspace diff baseline: %w", err)
	}
	right, err := indexWorkspaceDiff(after, afterExpected, &maxDeclarations)
	if err != nil {
		return WorkspaceDiffResult{}, fmt.Errorf("Atlas workspace diff current: %w", err)
	}
	coordinates := make(map[string]struct{}, len(left)+len(right))
	for _, side := range []map[string]Partition{left, right} {
		for coordinate := range side {
			coordinates[coordinate] = struct{}{}
		}
	}
	result := WorkspaceDiffResult{
		Schema: "fairyc.atlas-workspace-diff/v1", State: "bounded",
		Packages: make([]WorkspacePackageDiff, 0, len(coordinates)),
	}
	for _, coordinate := range slices.Sorted(maps.Keys(coordinates)) {
		old, oldOK := left[coordinate]
		current, currentOK := right[coordinate]
		entry := WorkspacePackageDiff{Coordinate: coordinate}
		switch {
		case !oldOK:
			entry = oneSidedWorkspaceDiff(entry, current, "added")
		case !currentOK:
			entry = oneSidedWorkspaceDiff(entry, old, "removed")
		default:
			if !matchingDiffContext(old, current) {
				return WorkspaceDiffResult{}, fmt.Errorf("Atlas workspace diff requires matching compiler and target identities for %s", coordinate)
			}
			report := compareDiffFacts(old, current,
				func() (map[string][sha256.Size]byte, map[string][sha256.Size]byte) {
					return bodyFingerprints(old, current)
				})
			entry.Before, entry.After = &report.Before, &report.After
			entry.BeforeDependencies, entry.AfterDependencies = diffDependencies(old), diffDependencies(current)
			entry.Changes, entry.Omissions, entry.State = report.Changes, report.Omissions, report.State
			entry.Action = "unchanged"
			if len(entry.Changes) != 0 || !slices.Equal(entry.BeforeDependencies, entry.AfterDependencies) {
				entry.Action = "modified"
			}
		}
		if entry.State == "partial" {
			result.State = "partial"
		}
		result.Packages = append(result.Packages, entry)
	}
	return result, nil
}

func indexWorkspaceDiff(partitions []Partition, expectations map[string]LoadExpectation, remaining *int) (map[string]Partition, error) {
	if len(partitions) != len(expectations) {
		return nil, fmt.Errorf("package set does not match current expectations")
	}
	index := make(map[string]Partition, len(partitions))
	instances := make(map[string]struct{}, len(partitions))
	for _, partition := range partitions {
		header := partition.data.header
		if len(partition.data.declarations) > *remaining {
			return nil, fmt.Errorf("declarations exceed limit")
		}
		*remaining -= len(partition.data.declarations)
		expectation, ok := expectations[header.PackageInstance]
		if header.LogicalSchema != LogicalSchemaV3 || !ok {
			return nil, fmt.Errorf("missing current v3 package expectation")
		}
		if err := compatibilityError(partition, expectation); err != nil {
			return nil, err
		}
		if _, duplicate := index[header.PackageCoordinate]; duplicate {
			return nil, fmt.Errorf("duplicate package coordinate")
		}
		index[header.PackageCoordinate] = partition
		instances[header.PackageInstance] = struct{}{}
	}
	for _, partition := range partitions {
		for _, dependency := range partition.data.header.Dependencies {
			if _, found := instances[dependency]; !found {
				return nil, fmt.Errorf("package dependency is absent from the snapshot")
			}
		}
	}
	return index, nil
}

func oneSidedWorkspaceDiff(entry WorkspacePackageDiff, partition Partition, action string) WorkspacePackageDiff {
	context := diffContext(partition)
	entry.State, entry.Omissions = diffCoverage(partition)
	entry.Action = action
	entry.Changes = make([]DiffChange, 0, len(partition.data.declarations))
	entry.BeforeDependencies, entry.AfterDependencies = []string{}, []string{}
	if action == "added" {
		entry.After, entry.AfterDependencies = &context, diffDependencies(partition)
	} else {
		entry.Before, entry.BeforeDependencies = &context, diffDependencies(partition)
	}
	for _, declaration := range partition.data.declarations {
		key := declaration.Key
		entry.Changes = append(entry.Changes, DiffChange{Module: key.Module, Kind: key.Kind, Name: key.Name, Action: action})
	}
	return entry
}

func diffDependencies(partition Partition) []string {
	header := partition.data.header
	dependencies := make([]string, 0, len(header.Dependencies))
	for _, reference := range header.PackageReferences {
		if _, direct := slices.BinarySearch(header.Dependencies, reference.Instance); direct {
			dependencies = append(dependencies, reference.Coordinate)
		}
	}
	slices.Sort(dependencies)
	return dependencies
}
