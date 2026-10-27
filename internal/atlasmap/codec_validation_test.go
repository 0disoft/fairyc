package atlasmap

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPartitionValidationPreservesOwnedRecords(t *testing.T) {
	for _, schema := range []string{LogicalSchema, LogicalSchemaV2, LogicalSchemaV3} {
		t.Run(schema, func(t *testing.T) {
			_, plans := planInputsFixture(t, 4, schema)
			if schema == LogicalSchemaV3 {
				workspace, result := loadAggregateMapWorkspace(t, true)
				var err error
				_, plans, err = PrepareWorkspaceV3(workspace, result, testCompilerBuild())
				if err != nil {
					t.Fatal(err)
				}
			}
			partition := plans[0]
			for _, invalid := range []bool{false, true} {
				t.Run(fmt.Sprintf("invalid=%t", invalid), func(t *testing.T) {
					if invalid {
						partition.data.functions[0].Provenance = "invalid"
					}
					// Keep a detached copy of every owned record, including nested
					// generic/aggregate facts and nil versus empty slices.
					before := partitionData{
						header: partition.Header(), files: partition.Files(),
						modules: partition.Modules(), symbols: partition.Symbols(),
						declarations: partition.Declarations(), functions: partition.Functions(),
						specializations: partition.Specializations(), completeness: partition.Completeness(),
					}
					if !reflect.DeepEqual(partition.data, before) {
						t.Fatal("detached test copy differs from the original partition")
					}
					err := validatePartitionVersion(partition, schema)
					if (err != nil) != invalid {
						t.Fatalf("validation error = %v, invalid = %t", err, invalid)
					}
					if !reflect.DeepEqual(partition.data, before) {
						t.Fatal("validation changed the partition's owned records")
					}
				})
			}
		})
	}
}

// Preparation is outside the timer; measure the shared read-only validator.
func BenchmarkValidatePartition(b *testing.B) {
	for _, count := range []int{64, 512} {
		for _, schema := range []string{LogicalSchema, LogicalSchemaV2, LogicalSchemaV3} {
			b.Run(fmt.Sprintf("declarations=%d/%s", count, schema), func(b *testing.B) {
				_, plans := planInputsFixture(b, count, schema)
				partition := plans[0]
				b.ReportAllocs()
				for b.Loop() {
					if err := validatePartitionVersion(partition, schema); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
