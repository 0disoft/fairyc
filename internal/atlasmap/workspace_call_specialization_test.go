package atlasmap

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"testing"
)

func TestWorkspaceCallSpecializationOwnershipAcrossVersions(t *testing.T) {
	for _, version := range []struct {
		name, logical string
		join          func([]Partition, map[string]LoadExpectation, WorkspaceCallBudget) (*WorkspaceCallIndex, error)
	}{
		{"v2", LogicalSchemaV2, NewWorkspaceCallIndexV2},
		{"v3", LogicalSchemaV3, NewWorkspaceCallIndexV3},
	} {
		t.Run(version.name, func(t *testing.T) {
			partitions, expectations := genericCallPartitions(t)
			for i := range partitions {
				partitions[i].data.header.LogicalSchema = version.logical
			}
			budget := WorkspaceCallBudget{MaxPackages: 2, MaxCalls: 4}
			index, err := version.join(partitions, expectations, budget)
			if err != nil {
				t.Fatal(err)
			}
			// Each node must retain the sorted subsequence admitted by the codec.
			for _, partition := range partitions {
				query, err := newQueryIndex(partition, expectations[partition.data.header.PackageInstance], version.logical)
				if err != nil {
					t.Fatal(err)
				}
				for _, node := range query.nodes {
					if !slices.IsSortedFunc(node.specializations, func(a, b QuerySpecializationFacts) int {
						return cmp.Compare(a.ID, b.ID)
					}) {
						t.Fatal("node lost canonical specialization ordering")
					}
					for _, implementation := range node.specializations {
						if !node.hasSpecialization(implementation.ID) {
							t.Fatal("owned implementation was not found")
						}
					}
				}
			}
			slices.Reverse(partitions)
			again, err := version.join(partitions, expectations, budget)
			if err != nil || !slices.Equal(index.Calls(), again.Calls()) {
				t.Fatalf("partition order changed the join: %v", err)
			}
			// The substituted ID exists in the caller package, but does not belong
			// to the dependency declaration named by the call target.
			changed := false
			for i := range partitions {
				for j := range partitions[i].data.specializations {
					implementation := &partitions[i].data.specializations[j]
					for k := range implementation.Function.Calls {
						call := &implementation.Function.Calls[k]
						if call.Target.PackageInstance != partitions[i].data.header.PackageInstance {
							call.Specialization = implementation.ID
							changed = true
						}
					}
				}
			}
			if !changed {
				t.Fatal("fixture has no cross-package specialization call")
			}
			if _, err := version.join(partitions, expectations, budget); !errors.Is(err, ErrWorkspaceJoinIncomplete) {
				t.Fatalf("another declaration's implementation was admitted: %v", err)
			}
		})
	}
}

// Measure only the declaration-local membership check used by workspace joins.
// Preparation and semantic admission are outside this lookup benchmark.
func BenchmarkWorkspaceCallSpecializationLookup(b *testing.B) {
	for _, count := range []int{32, 512} {
		b.Run(fmt.Sprintf("implementations=%d", count), func(b *testing.B) {
			node := indexedQueryNode{specializations: make([]QuerySpecializationFacts, count)}
			for i := range node.specializations {
				node.specializations[i].ID = fmt.Sprintf("g0_%064x", 2*i+2)
			}
			for _, hit := range []bool{true, false} {
				b.Run(fmt.Sprintf("hit=%t", hit), func(b *testing.B) {
					targets := make([]string, count)
					for i := range targets {
						if hit {
							targets[i] = node.specializations[i].ID
						} else {
							targets[i] = fmt.Sprintf("g0_%064x", 2*i+1)
						}
					}
					b.ReportAllocs()
					i := 0
					for b.Loop() {
						if node.hasSpecialization(targets[i]) != hit {
							b.Fatal("lookup returned an incorrect result")
						}
						i = (i + 1) % count
					}
				})
			}
		})
	}
}
