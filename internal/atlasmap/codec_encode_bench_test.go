package atlasmap

import "testing"

// Preparation is outside the timer; include both conversion and the public encoder.
func BenchmarkPartitionTransport(b *testing.B) {
	for _, version := range []struct {
		name   string
		schema string
		encode func(Partition, uint64) ([]byte, error)
	}{
		{"v1", LogicalSchema, Encode},
		{"v2", LogicalSchemaV2, EncodeV2},
		{"v3", LogicalSchemaV3, EncodeV3},
	} {
		b.Run(version.name, func(b *testing.B) {
			_, plans := planInputsFixture(b, 512, version.schema)
			partition := plans[0]
			b.Run("document", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					document := documentFromPartition(partition)
					if len(document.Declarations) != len(partition.data.declarations) {
						b.Fatal("conversion lost declarations")
					}
				}
			})
			b.Run("encode", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := version.encode(partition, testMapLimit); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
