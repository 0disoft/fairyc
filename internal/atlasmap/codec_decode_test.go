package atlasmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodePreservesValidationAndCanonicalizationAcrossVersions(t *testing.T) {
	t.Parallel()
	partition := decodeBenchmarkPartition(t, 4)
	for _, version := range []struct {
		name, logical, schema, encoding string
		encode                          func(Partition, uint64) ([]byte, error)
		decode                          func([]byte, uint64) (Partition, error)
	}{
		{"v1", LogicalSchema, TransportSchema, TransportEncoding, Encode, Decode},
		{"v2", LogicalSchemaV2, TransportSchemaV2, TransportEncodingV2, EncodeV2, DecodeV2},
		{"v3", LogicalSchemaV3, TransportSchemaV3, TransportEncodingV3, EncodeV3, DecodeV3},
	} {
		t.Run(version.name, func(t *testing.T) {
			partition.data.header.LogicalSchema = version.logical
			encoded, err := version.encode(partition, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := version.decode(encoded, uint64(len(encoded)))
			if err != nil {
				t.Fatal(err)
			}
			again, err := version.encode(decoded, uint64(len(encoded)))
			if err != nil || !bytes.Equal(encoded, again) {
				t.Fatalf("canonical bytes changed: %v", err)
			}
			for _, mutation := range []struct {
				name   string
				change func(*transportDocument)
				match  string
			}{
				{"counts", func(d *transportDocument) { d.Header.Counts.Files++ }, "counts"},
				{"symbol identity", func(d *transportDocument) { d.Declarations[0].Unsafe = !d.Declarations[0].Unsafe }, "does not match its declaration"},
				// Semantic records accept empty collections; canonical wire bytes require [].
				{"nested null", func(d *transportDocument) { d.Functions[1].Calls = nil }, ""},
			} {
				t.Run(mutation.name, func(t *testing.T) {
					var envelope transportEnvelope
					if err := json.Unmarshal(encoded, &envelope); err != nil {
						t.Fatal(err)
					}
					mutation.change(&envelope.Map)
					checksum, err := computeVersionChecksum(envelope.Map, version.schema, version.encoding)
					if err != nil {
						t.Fatal(err)
					}
					envelope.Checksum = checksum.String()
					mutated, err := json.Marshal(envelope)
					if err != nil {
						t.Fatal(err)
					}
					got, err := version.decode(mutated, testMapLimit)
					if err == nil || got.data.declarations != nil {
						t.Fatalf("invalid input returned a partition: %v", err)
					}
					if mutation.match == "" {
						if !errors.Is(err, ErrNonCanonical) {
							t.Fatalf("noncanonical wire error = %v", err)
						}
					} else if !strings.Contains(err.Error(), mutation.match) {
						t.Fatalf("semantic validation error = %v", err)
					}
				})
			}
		})
	}
}
