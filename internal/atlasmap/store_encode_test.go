package atlasmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMapPublicationUsesEncodedChecksumAcrossVersions(t *testing.T) {
	partition := decodeBenchmarkPartition(t, 4)
	for _, version := range []struct {
		name, logical, schema, encoding string
		encode                          func(Partition, uint64) ([]byte, error)
		newStore                        func(string, MapStoreLimits) (*MapStore, error)
	}{
		{"v1", LogicalSchema, TransportSchema, TransportEncoding, Encode, NewMapStore},
		{"v2", LogicalSchemaV2, TransportSchemaV2, TransportEncodingV2, EncodeV2, NewMapStoreV2},
		{"v3", LogicalSchemaV3, TransportSchemaV3, TransportEncodingV3, EncodeV3, NewMapStoreV3},
	} {
		t.Run(version.name, func(t *testing.T) {
			partition.data.header.LogicalSchema = version.logical
			encoded, err := version.encode(partition, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			var envelope transportEnvelope
			if err := json.Unmarshal(encoded, &envelope); err != nil {
				t.Fatal(err)
			}
			want, err := ParseChecksum(envelope.Checksum)
			if err != nil {
				t.Fatal(err)
			}
			for _, limit := range []uint64{uint64(len(encoded) - 1), uint64(len(encoded)), uint64(len(encoded) + 1)} {
				limits := testMapStoreLimits()
				limits.MaxMapBytes = limit
				store, err := version.newStore(t.TempDir(), limits)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { store.Close() })
				checksum, err := store.Publish(partition)
				if limit < uint64(len(encoded)) {
					if !errors.Is(err, ErrMapTooLarge) || checksum != (Checksum{}) {
						t.Fatalf("rejected publish: checksum=%s error=%v", checksum, err)
					}
					if _, err := store.root.Lstat(partition.data.header.PackageInstance); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("size rejection created a package directory: %v", err)
					}
					continue
				}
				if err != nil || checksum != want {
					t.Fatalf("published checksum=%s want=%s error=%v", checksum, want, err)
				}
				path := filepath.Join(partition.data.header.PackageInstance, mapGenerationName(want))
				stored, err := store.root.ReadFile(path)
				if err != nil || !bytes.Equal(stored, encoded) {
					t.Fatalf("generation name and canonical bytes disagree: %v", err)
				}
				again, err := store.Publish(partition)
				if err != nil || again != checksum {
					t.Fatalf("repeated publication changed identity: %v", err)
				}
			}
			// Invalid input must not return a usable checksum or encoded bytes.
			for _, limit := range []uint64{0, 1} {
				encoded, checksum, err := encodeVersionWithChecksum(Partition{}, limit, version.schema, version.encoding, version.logical)
				if err == nil || encoded != nil || checksum != (Checksum{}) {
					t.Fatalf("invalid partition exposed encoded bytes/checksum: %v", err)
				}
			}
		})
	}
}

// Isolate CPU/allocations from filesystem flush timing while retaining the
// former publisher preparation as a comparison for the same canonical input.
func BenchmarkMapPublishPreparation(b *testing.B) {
	partition := decodeBenchmarkPartition(b, 512)
	partition.data.header.LogicalSchema = LogicalSchemaV3
	for _, repeat := range []bool{true, false} {
		name := "encode-once"
		if repeat {
			name = "encode-and-rehash"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				encoded, checksum, err := encodeVersionWithChecksum(partition, testMapLimit, TransportSchemaV3, TransportEncodingV3, LogicalSchemaV3)
				if err == nil && repeat {
					checksum, err = computeVersionChecksum(documentFromPartition(partition), TransportSchemaV3, TransportEncodingV3)
				}
				if err != nil || len(encoded) == 0 || checksum == (Checksum{}) {
					b.Fatalf("prepare publication: %v", err)
				}
			}
		})
	}
}

func BenchmarkMapStorePublishEncoding(b *testing.B) {
	partition := decodeBenchmarkPartition(b, 512)
	partition.data.header.LogicalSchema = LogicalSchemaV3
	store, err := NewMapStoreV3(b.TempDir(), testMapStoreLimits())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { store.Close() })
	b.ReportAllocs()
	for b.Loop() {
		if _, err := store.Publish(partition); err != nil {
			b.Fatal(err)
		}
	}
}
