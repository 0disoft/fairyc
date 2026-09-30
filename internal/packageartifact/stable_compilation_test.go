package packageartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableCompilationArtifactRoundTripsAcrossStatePackageRenumbering(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	process := processArtifacts[0]
	stateBodies, err := contractstatemir.DescribeStableHIRStateBodies(
		hir.Package{}, []hir.PackageID{process.PackageID},
		map[hir.PackageID]packagegraph.CoordinateID{process.PackageID: process.PackageCoordinate},
	)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := DescribeStableCompilationArtifact(process, stateBodies[process.PackageID], stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableCompilationArtifact(artifact, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("capability")) {
		t.Fatal("stable compilation artifact serialized capability authority")
	}
	decoded, err := DecodeStableCompilationArtifact(encoded, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, artifact) {
		t.Fatalf("decoded stable compilation differs:\n got %#v\nwant %#v", decoded, artifact)
	}
	reencoded, err := EncodeStableCompilationArtifact(decoded, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reencoded, encoded) {
		t.Fatal("stable compilation artifact did not re-encode byte-for-byte")
	}

	freshPackage := hir.PackageID(900)
	freshBodies, err := contractstatemir.DescribeStableHIRStateBodies(
		hir.Package{}, []hir.PackageID{freshPackage},
		map[hir.PackageID]packagegraph.CoordinateID{freshPackage: process.PackageCoordinate},
	)
	if err != nil {
		t.Fatal(err)
	}
	renumbered := artifact
	renumbered.StateBody = freshBodies[freshPackage]
	renumberedBytes, err := EncodeStableCompilationArtifact(renumbered, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(renumberedBytes, encoded) {
		t.Fatal("state package renumbering changed stable compilation bytes")
	}

	digest := sha256.Sum256(encoded)
	const wantDigest = "6c00664f3f794b411c50d07620e77ce3010d6dd61448c810ea5a44b4bd92dbb9"
	if got := hex.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("stable compilation digest = %s, want %s", got, wantDigest)
	}
}

func TestStableCompilationArtifactRejectsBoundsEncodingAndOwnershipDrift(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	process := processArtifacts[0]
	stateBodies, err := contractstatemir.DescribeStableHIRStateBodies(
		hir.Package{}, []hir.PackageID{process.PackageID},
		map[hir.PackageID]packagegraph.CoordinateID{process.PackageID: process.PackageCoordinate},
	)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := DescribeStableCompilationArtifact(process, stateBodies[process.PackageID], stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableCompilationArtifact(artifact, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeStableCompilationArtifact(artifact, 0); err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("zero encode bound error = %v", err)
	}
	if _, err := DecodeStableCompilationArtifact(encoded, 0); err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("zero decode bound error = %v", err)
	}
	if _, err := EncodeStableCompilationArtifact(artifact, uint64(len(encoded)-1)); !errors.Is(err, ErrStableCompilationTooLarge) {
		t.Fatalf("small encode bound error = %v", err)
	}
	if _, err := DecodeStableCompilationArtifact(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrStableCompilationTooLarge) {
		t.Fatalf("small decode bound error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)
	if _, err := DecodeStableCompilationArtifact(unknown, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	if _, err := DecodeStableCompilationArtifact(append([]byte(" "), encoded...), stablePackageArtifactLimit); !errors.Is(err, ErrStableCompilationEncoding) {
		t.Fatalf("noncanonical error = %v", err)
	}
	if _, err := DecodeStableCompilationArtifact(append(append([]byte(nil), encoded...), []byte(`{}`)...), stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
		t.Fatalf("trailing-value error = %v", err)
	}
	wrongVersion := bytes.Replace(encoded, []byte(StableCompilationVersion), []byte("fairyc-stable-package-compilation/99"), 1)
	if _, err := DecodeStableCompilationArtifact(wrongVersion, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("wrong-version error = %v", err)
	}

	otherPackage := process.PackageID + 1000
	otherCoordinate := testCoordinateID(t, "stable-compilation-other")
	otherBodies, err := contractstatemir.DescribeStableHIRStateBodies(
		hir.Package{}, []hir.PackageID{otherPackage},
		map[hir.PackageID]packagegraph.CoordinateID{otherPackage: otherCoordinate},
	)
	if err != nil {
		t.Fatal(err)
	}
	mismatched := artifact
	mismatched.StateBody = otherBodies[otherPackage]
	if _, err := EncodeStableCompilationArtifact(mismatched, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "different package coordinates") {
		t.Fatalf("coordinate mismatch error = %v", err)
	}
	corrupt := artifact
	corrupt.StateBody.Hash[0] ^= 0xff
	if _, err := EncodeStableCompilationArtifact(corrupt, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "hash does not match") {
		t.Fatalf("state body corruption error = %v", err)
	}
}
