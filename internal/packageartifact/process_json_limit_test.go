package packageartifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestProcessJSONLimitMatchesCanonicalEncoding(t *testing.T) {
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifacts[0]
	artifact.Functions[0].Body.Span.Source = "<>&\"\\\n\x00\u2028\u2029\xff"
	for _, storage := range []bool{false, true} {
		codec := fixedProcessCodec(t, 1)
		codec.storageTransfers = storage
		schema, version, _ := codec.contract()
		want, err := json.Marshal(processEnvelope{schema, version, codec.Identity(), artifact})
		if err != nil {
			t.Fatal(err)
		}
		for _, limit := range []uint64{uint64(len(want) - 1), uint64(len(want)), uint64(len(want) + 1)} {
			codec.maxBytes = limit
			got, err := codec.Encode(artifact)
			if limit < uint64(len(want)) {
				if !errors.Is(err, ErrProcessArtifactTooLarge) || got != nil {
					t.Fatalf("oversize admission: %v", err)
				}
			} else if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("canonical bytes changed, storage=%t: %v", storage, err)
			}
		}
	}
}

func TestProcessJSONMapBoundsNilEmptyAndUnsignedKeys(t *testing.T) {
	for _, maps := range []Artifact{
		{},
		{PackageIdentities: map[hir.PackageID]packagegraph.InstanceID{}, PackageCoordinates: map[hir.PackageID]packagegraph.CoordinateID{}},
		{PackageIdentities: map[hir.PackageID]packagegraph.InstanceID{0: {1}, 9: {2}, 10: {3}, ^hir.PackageID(0): {4}}, PackageCoordinates: map[hir.PackageID]packagegraph.CoordinateID{100: {1}, ^hir.PackageID(0): {2}}},
	} {
		value := processEnvelope{Artifact: maps}
		want, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, limit := range []uint64{uint64(len(want) - 1), uint64(len(want)), ^uint64(0)} {
			exceeds, err := stableJSONExceedsLimit(value, limit)
			if err != nil || exceeds != (limit < uint64(len(want))) {
				t.Fatalf("map accounting at %d bytes: %t, %v", limit, exceeds, err)
			}
		}
	}
}

func TestProcessJSONRejectsLargePayloadBeforeAllocation(t *testing.T) {
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifacts[0]
	artifact.Functions[0].Body.Span.Source = strings.Repeat("x", 16<<20)
	codec := fixedProcessCodec(t, 1)
	codec.maxBytes = 1024
	if _, err := codec.Encode(artifact); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatal(err)
	}
	// Measure the marshal boundary, excluding independently verified artifact
	// construction and the verifier's own graph clones.
	envelope := processEnvelope{ProcessCodecSchema, ProcessCodecVersion, codec.Identity(), artifact}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err = marshalProcessEnvelope(envelope, codec.maxBytes)
	runtime.ReadMemStats(&after)
	if !errors.Is(err, ErrProcessArtifactTooLarge) || after.TotalAlloc-before.TotalAlloc > 1<<20 {
		t.Fatalf("oversized JSON allocated a payload buffer: %d bytes, %v", after.TotalAlloc-before.TotalAlloc, err)
	}
	// Verification still precedes byte rejection at the public encoding boundary.
	artifact.Package = packagegraph.InstanceID{}
	if _, err := codec.Encode(artifact); err == nil || errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("invalid artifact hidden by byte admission: %v", err)
	}
}
