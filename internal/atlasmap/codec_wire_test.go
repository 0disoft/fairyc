package atlasmap

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestPartitionTransportCanonicalAndDetached(t *testing.T) {
	// Canonical digests were captured before removing defensive accessor copies.
	for _, version := range []struct {
		name   string
		schema string
		encode func(Partition, uint64) ([]byte, error)
		digest string
	}{
		{"v1", LogicalSchema, Encode, "083589035dc50b1770747690a2f5710905ac92f4c25af0d9b59c033037814eae"},
		{"v2", LogicalSchemaV2, EncodeV2, "2cca7eb5795435053571b1c909f86cb362dd06fdf301b5d878ab0c75f3eaad25"},
		{"v3", LogicalSchemaV3, EncodeV3, "db4ebae525a105dec3dad9e3467ae9c79d1137e0898e159ae280c0f2f54b4fc0"},
	} {
		t.Run(version.name, func(t *testing.T) {
			_, plans := planInputsFixture(t, 4, version.schema)
			if version.schema == LogicalSchemaV3 {
				workspace, result := loadAggregateMapWorkspace(t, true)
				var err error
				_, plans, err = PrepareWorkspaceV3(workspace, result, testCompilerBuild())
				if err != nil {
					t.Fatal(err)
				}
			}
			partition := plans[0]
			encoded, err := version.encode(partition, testMapLimit)
			if err != nil {
				t.Fatal(err)
			}
			digest := fmt.Sprintf("%x", sha256.Sum256(encoded))
			if digest != version.digest {
				t.Fatalf("canonical bytes changed: digest=%s", digest)
			}
			document := documentFromPartition(partition)
			mutateTransportStrings(reflect.ValueOf(&document))
			after, err := version.encode(partition, testMapLimit)
			if err != nil || !bytes.Equal(encoded, after) {
				t.Fatalf("mutating transport changed the partition: %v", err)
			}
			// Also check the opposite direction without sharing the expected value.
			document = documentFromPartition(partition)
			before, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			partition.data.header.PackageReferences[0].UID = "changed"
			if len(partition.data.header.Features) != 0 {
				partition.data.header.Features[0] = "changed"
			}
			for i := range partition.data.declarations {
				declaration := &partition.data.declarations[i]
				if len(declaration.TypeParameters) != 0 {
					declaration.TypeParameters[0] = "changed"
				}
			}
			after, err = json.Marshal(document)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("mutating the partition changed transport: %v", err)
			}
		})
	}
}

// Exercise every nested string slice, including generic and cleanup metadata.
func mutateTransportStrings(value reflect.Value) {
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			mutateTransportStrings(value.Elem())
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			mutateTransportStrings(value.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			mutateTransportStrings(value.Index(i))
		}
	case reflect.String:
		value.SetString(value.String() + "-changed")
	}
}
