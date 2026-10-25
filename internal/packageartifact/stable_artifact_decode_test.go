package packageartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestStableArtifactDecodeKeepsValidationAndCanonicalBytes(t *testing.T) {
	processArtifact := typedProjectionArtifact(t, 4)
	for _, transfers := range []bool{false, true} {
		t.Run(fmt.Sprintf("transfers=%v", transfers), func(t *testing.T) {
			artifact, err := describeStableArtifact(processArtifact, stablePackageArtifactLimit, transfers)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := encodeStableArtifact(artifact, stablePackageArtifactLimit, transfers)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decodeStableArtifact(encoded, stablePackageArtifactLimit, transfers)
			if err != nil || !reflect.DeepEqual(decoded, artifact) {
				t.Fatalf("stable artifact changed after decode: %v", err)
			}
			reencoded, err := encodeStableArtifact(decoded, stablePackageArtifactLimit, transfers)
			if err != nil || !bytes.Equal(reencoded, encoded) {
				t.Fatalf("canonical bytes changed after decode: %v", err)
			}
			artifact.Functions[0].Function.Symbol = 7
			if _, err := encodeStableArtifact(artifact, stablePackageArtifactLimit, transfers); err == nil {
				t.Fatal("encoder accepted a build-local function symbol")
			}
			schema, version := stableArtifactContract(transfers)
			corrupt, err := json.Marshal(stableArtifactEnvelope{Schema: schema, Version: version, Artifact: artifact})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeStableArtifact(corrupt, stablePackageArtifactLimit, transfers); err == nil || !strings.Contains(err.Error(), "verify decoded") {
				t.Fatalf("decoder must reject invalid MIR before canonical encoding: %v", err)
			}
		})
	}
}

func BenchmarkDecodeStableArtifact(b *testing.B) {
	for _, count := range []int{1, 64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			processArtifact := typedProjectionArtifact(b, count)
			artifact, err := DescribeStableArtifact(processArtifact, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			encoded, err := EncodeStableArtifact(artifact, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(encoded)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				decoded, err := DecodeStableArtifact(encoded, stablePackageArtifactLimit)
				if err != nil || len(decoded.Functions) != count {
					b.Fatalf("decode %d functions: %v", count, err)
				}
			}
		})
	}
}
