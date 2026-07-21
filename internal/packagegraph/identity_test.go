package packagegraph

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestEncodeIdentityMatchesCanonicalV1Vector(t *testing.T) {
	t.Parallel()
	identity := Identity{
		PackageUID:      "pkg",
		SourceIdentity:  "src",
		VersionIdentity: "1",
		Features:        []string{"base", "tls"},
	}
	encoded, err := EncodeIdentity(identity)
	if err != nil {
		t.Fatalf("EncodeIdentity() error = %v", err)
	}
	want := []byte("fairyc.package-instance\x00\x01" +
		"u\x00\x00\x00\x00\x00\x00\x00\x03pkg" +
		"s\x00\x00\x00\x00\x00\x00\x00\x03src" +
		"v\x00\x00\x00\x00\x00\x00\x00\x011" +
		"f\x00\x00\x00\x00\x00\x00\x00\x02" +
		"\x00\x00\x00\x00\x00\x00\x00\x04base" +
		"\x00\x00\x00\x00\x00\x00\x00\x03tls")
	if string(encoded) != string(want) {
		t.Fatalf("EncodeIdentity() = %s, want %s", hex.EncodeToString(encoded), hex.EncodeToString(want))
	}

	id, err := ComputeInstanceID(identity)
	if err != nil {
		t.Fatalf("ComputeInstanceID() error = %v", err)
	}
	wantDigest := sha256.Sum256(want)
	if id != InstanceID(wantDigest) {
		t.Fatalf("ComputeInstanceID() = %s, want %s", id, InstanceID(wantDigest))
	}
	if !strings.HasPrefix(id.String(), InstanceIDPrefix) || len(id.String()) != len(InstanceIDPrefix)+sha256.Size*2 {
		t.Fatalf("InstanceID.String() = %q", id.String())
	}
}

func TestIdentityEncodingPreservesComponentBoundaries(t *testing.T) {
	t.Parallel()
	left, err := ComputeInstanceID(Identity{PackageUID: "ab", SourceIdentity: "c", VersionIdentity: "d"})
	if err != nil {
		t.Fatalf("left identity error = %v", err)
	}
	right, err := ComputeInstanceID(Identity{PackageUID: "a", SourceIdentity: "bc", VersionIdentity: "d"})
	if err != nil {
		t.Fatalf("right identity error = %v", err)
	}
	if left == right {
		t.Fatalf("component-boundary identities collided: %s", left)
	}

	featureLeft, err := ComputeInstanceID(Identity{PackageUID: "p", SourceIdentity: "s", VersionIdentity: "v", Features: []string{"a", "bc"}})
	if err != nil {
		t.Fatalf("left feature identity error = %v", err)
	}
	featureRight, err := ComputeInstanceID(Identity{PackageUID: "p", SourceIdentity: "s", VersionIdentity: "v", Features: []string{"ab", "c"}})
	if err != nil {
		t.Fatalf("right feature identity error = %v", err)
	}
	if featureLeft == featureRight {
		t.Fatalf("feature-boundary identities collided: %s", featureLeft)
	}
}

func TestParseInstanceIDRequiresCanonicalText(t *testing.T) {
	t.Parallel()
	id, err := ComputeInstanceID(testIdentity("parse"))
	if err != nil {
		t.Fatalf("ComputeInstanceID() error = %v", err)
	}
	parsed, err := ParseInstanceID(id.String())
	if err != nil || parsed != id {
		t.Fatalf("ParseInstanceID() = %s, %v; want %s", parsed, err, id)
	}

	for _, value := range []string{
		strings.TrimPrefix(id.String(), InstanceIDPrefix),
		InstanceIDPrefix + strings.ToUpper(strings.TrimPrefix(id.String(), InstanceIDPrefix)),
		id.String() + "00",
		InstanceIDPrefix + strings.Repeat("z", sha256.Size*2),
	} {
		if _, err := ParseInstanceID(value); err == nil {
			t.Fatalf("ParseInstanceID(%q) succeeded", value)
		}
	}
}

func TestComputeInstanceIDRejectsInvalidIdentity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		identity Identity
		reason   string
	}{
		{name: "empty component", identity: Identity{SourceIdentity: "s", VersionIdentity: "v"}, reason: IdentityReasonEmptyComponent},
		{name: "invalid component utf8", identity: Identity{PackageUID: string([]byte{0xff}), SourceIdentity: "s", VersionIdentity: "v"}, reason: IdentityReasonInvalidUTF8},
		{name: "unsorted features", identity: Identity{PackageUID: "p", SourceIdentity: "s", VersionIdentity: "v", Features: []string{"tls", "base"}}, reason: IdentityReasonFeaturesNotSortedOrUnique},
		{name: "duplicate features", identity: Identity{PackageUID: "p", SourceIdentity: "s", VersionIdentity: "v", Features: []string{"base", "base"}}, reason: IdentityReasonFeaturesNotSortedOrUnique},
		{name: "empty feature", identity: Identity{PackageUID: "p", SourceIdentity: "s", VersionIdentity: "v", Features: []string{""}}, reason: IdentityReasonEmptyFeature},
		{name: "invalid feature utf8", identity: Identity{PackageUID: "p", SourceIdentity: "s", VersionIdentity: "v", Features: []string{string([]byte{0xff})}}, reason: IdentityReasonInvalidUTF8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ComputeInstanceID(test.identity)
			var validationError *IdentityValidationError
			if !errors.As(err, &validationError) || validationError.Reason != test.reason {
				t.Fatalf("ComputeInstanceID() error = %v, want reason %q", err, test.reason)
			}
		})
	}
}
