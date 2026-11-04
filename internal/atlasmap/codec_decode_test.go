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

func TestDecodeVerifiedChecksumPreservesCanonicalGate(t *testing.T) {
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
			before := bytes.Clone(encoded)
			for _, limit := range []uint64{uint64(len(encoded) - 1), uint64(len(encoded)), uint64(len(encoded) + 1)} {
				decoded, err := version.decode(encoded, limit)
				if limit < uint64(len(encoded)) {
					if !errors.Is(err, ErrMapTooLarge) {
						t.Fatalf("limit %d: %v", limit, err)
					}
					continue
				}
				again, encodeErr := version.encode(decoded, testMapLimit)
				if err != nil || encodeErr != nil || !bytes.Equal(encoded, again) {
					t.Fatalf("exact/above limit round trip: decode=%v encode=%v", err, encodeErr)
				}
			}
			for _, field := range []string{"features", "dependencies", "arguments", "borrows", "unsafe_sites"} {
				t.Run(field+" null", func(t *testing.T) {
					mutated := bytes.Replace(encoded, []byte(`"`+field+`":[]`), []byte(`"`+field+`":null`), 1)
					if bytes.Equal(mutated, encoded) {
						t.Fatal("fixture has no empty collection to mutate")
					}
					var envelope transportEnvelope
					if err := json.Unmarshal(mutated, &envelope); err != nil {
						t.Fatal(err)
					}
					checksum, err := computeVersionChecksum(envelope.Map, version.schema, version.encoding)
					if err != nil {
						t.Fatal(err)
					}
					envelope.Checksum = checksum.String()
					mutated, err = json.Marshal(envelope)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := version.decode(mutated, uint64(len(mutated))); !errors.Is(err, ErrNonCanonical) {
						t.Fatalf("verified checksum admitted normalized %s: %v", field, err)
					}
					if field == "features" {
						// Omitted fields decode as nil too. Rebuilding the canonical
						// envelope can exceed the original, shorter input's limit.
						omitted := bytes.Replace(mutated, []byte(`"features":null,`), nil, 1)
						if _, err := version.decode(omitted, uint64(len(omitted))); !errors.Is(err, ErrMapTooLarge) {
							t.Fatalf("canonical size must still precede byte mismatch: %v", err)
						}
						if _, err := version.decode(omitted, testMapLimit); !errors.Is(err, ErrNonCanonical) {
							t.Fatalf("omitted collection with sufficient limit: %v", err)
						}
					}
				})
			}
			duplicate := bytes.Replace(encoded, []byte(`"schema":`), []byte(`"schema":"`+version.schema+`","schema":`), 1)
			escaped := bytes.Replace(encoded, []byte(`"package_uid":"test.`), []byte(`"package_uid":"\u0074est.`), 1)
			for _, input := range [][]byte{duplicate, escaped, append(bytes.Clone(encoded), '\n')} {
				if _, err := version.decode(input, testMapLimit); !errors.Is(err, ErrNonCanonical) {
					t.Fatalf("semantically equivalent noncanonical JSON: %v", err)
				}
			}
			if !bytes.Equal(encoded, before) {
				t.Fatal("decode modified caller-owned bytes")
			}
		})
	}
}
