package contractstatemir

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableStateBodyArtifactRoundTripsWithoutLocalAuthority(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-e8756812ba7956f1f997a35a522b7f1735d8679c99c48b140570f4cd690c1a91")
	base := stableStateBodyPackage(t, 2, 17, 33)
	artifacts, err := DescribeStableHIRStateBodies(base, []hir.PackageID{2}, map[hir.PackageID]packagegraph.CoordinateID{2: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	hashes, err := ComputeHIRStateBodyHashes(base, []hir.PackageID{2}, map[hir.PackageID]packagegraph.CoordinateID{2: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[2]
	if body.Hash != hashes[2] {
		t.Fatalf("artifact hash = %s, computed hash %s", body.Hash, hashes[2])
	}

	encoded, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte("$p2"), []byte("$f17"), []byte("capability")} {
		if bytes.Contains(encoded, forbidden) {
			t.Fatalf("encoded artifact contains build-local authority %q", forbidden)
		}
	}
	decoded, err := DecodeStableStateBody(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, body) {
		t.Fatalf("decoded artifact differs:\n got %#v\nwant %#v", decoded, body)
	}
	reencoded, err := EncodeStableStateBody(decoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reencoded, encoded) {
		t.Fatal("stable artifact did not re-encode byte-for-byte")
	}

	renumbered := stableStateBodyPackage(t, 9, 501, 700)
	renumberedArtifacts, err := DescribeStableHIRStateBodies(renumbered, []hir.PackageID{9}, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	renumberedEncoded, err := EncodeStableStateBody(renumberedArtifacts[9], 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(renumberedEncoded, encoded) {
		t.Fatal("build-local package and function renumbering changed stable payload bytes")
	}

	digest := sha256.Sum256(encoded)
	const wantDigest = "2b00beb8c83793a8664f0095cb9c707b24ffe7345f5f63f705d1a932e2a4958f"
	if got := hex.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("encoded artifact digest = %s, want %s", got, wantDigest)
	}
}

func TestStableStateBodyArtifactEnforcesBoundsAndCanonicalJSON(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-903a7b9974986cccf6166e8978ca9cb26273d4776af6d1a5dd3b04e7060e2f41")
	artifacts, err := DescribeStableHIRStateBodies(stableStateBodyPackage(t, 1, 7, 9), []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[1]
	encoded, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeStableStateBody(body, 0); err == nil {
		t.Fatal("EncodeStableStateBody accepted a zero byte limit")
	}
	if _, err := DecodeStableStateBody(encoded, 0); err == nil {
		t.Fatal("DecodeStableStateBody accepted a zero byte limit")
	}
	if _, err := EncodeStableStateBody(body, uint64(len(encoded)-1)); !errors.Is(err, ErrStableStateBodyTooLarge) {
		t.Fatalf("bounded encode error = %v", err)
	}
	if _, err := DecodeStableStateBody(encoded, uint64(len(encoded)-1)); !errors.Is(err, ErrStableStateBodyTooLarge) {
		t.Fatalf("bounded decode error = %v", err)
	}

	noncanonical := append([]byte(" "), encoded...)
	if _, err := DecodeStableStateBody(noncanonical, 1<<20); !errors.Is(err, ErrStableStateBodyEncoding) {
		t.Fatalf("noncanonical decode error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`"schema":`), []byte(`"unknown":0,"schema":`), 1)
	if _, err := DecodeStableStateBody(unknown, 1<<20); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field decode error = %v", err)
	}
	wrongVersion := bytes.Replace(encoded, []byte(StableStateBodyVersion), []byte("fairyc-contract-state-package-body-artifact/2"), 1)
	if _, err := DecodeStableStateBody(wrongVersion, 1<<20); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("wrong-version decode error = %v", err)
	}
	if _, err := DecodeStableStateBody(append(slices.Clone(encoded), []byte("null")...), 1<<20); err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
		t.Fatalf("trailing-value decode error = %v", err)
	}
}

func TestStableStateBodyArtifactRejectsMalformedPayloads(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-04fc4400f5e875c5c22ab2d50765d721df38a12a19ef3508deb432c7f4da92f3")
	artifacts, err := DescribeStableHIRStateBodies(stableStateBodyPackage(t, 1, 7, 9), []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	base := artifacts[1]
	tests := []struct {
		name   string
		mutate func(*StableStateBodyArtifact)
	}{
		{name: "missing identity", mutate: func(value *StableStateBodyArtifact) { value.Hash = StateBodyHash{} }},
		{name: "nil transactions", mutate: func(value *StableStateBodyArtifact) { value.Transactions = nil }},
		{name: "nil layout fields", mutate: func(value *StableStateBodyArtifact) { value.Layout.Fields = nil }},
		{name: "duplicate function", mutate: func(value *StableStateBodyArtifact) {
			value.Transactions = append(value.Transactions, value.Transactions[0])
		}},
		{name: "trailing descriptor byte", mutate: func(value *StableStateBodyArtifact) {
			value.Transactions[0].Function = append(value.Transactions[0].Function, 0)
		}},
		{name: "noncanonical descriptor type", mutate: func(value *StableStateBodyArtifact) {
			descriptor, err := decodeStableStateFunctionDescriptor(value.Transactions[0].Function)
			if err != nil {
				t.Fatal(err)
			}
			descriptor.parameterTypes = []string{"Result<i32,u32>"}
			value.Transactions[0].Function = encodeStableStateFunctionDescriptor(descriptor)
		}},
		{name: "effect mismatch", mutate: func(value *StableStateBodyArtifact) { value.Transactions[0].Effects = []string{} }},
		{name: "unsupported operation", mutate: func(value *StableStateBodyArtifact) { value.Transactions[0].Operations[0].Kind = "peek" }},
		{name: "noncanonical value", mutate: func(value *StableStateBodyArtifact) {
			value.Transactions[0].Operations[0].Value.Encoded = append(value.Transactions[0].Operations[0].Value.Encoded, 0)
		}},
		{name: "trap metadata on commit", mutate: func(value *StableStateBodyArtifact) { value.Transactions[0].TrapCategory = "panic" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			malformed := cloneStableStateBodyArtifact(base)
			test.mutate(&malformed)
			if _, err := EncodeStableStateBody(malformed, 1<<20); err == nil {
				t.Fatal("EncodeStableStateBody accepted malformed payload")
			}
		})
	}
}

func TestStableStateBodyArtifactSupportsEmptyPackageBody(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-0f05e8aec70e99fa5420f3c82f7bbb082bf9126349965e2414300320236eeeda")
	artifacts, err := DescribeStableHIRStateBodies(hir.Package{}, []hir.PackageID{3}, map[hir.PackageID]packagegraph.CoordinateID{3: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[3]
	if body.Layout != nil || body.Transactions == nil || len(body.Transactions) != 0 || body.Hash == (StateBodyHash{}) {
		t.Fatalf("empty package artifact = %#v", body)
	}
	encoded, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeStableStateBody(encoded, uint64(len(encoded))); err != nil {
		t.Fatal(err)
	}
}

func stableStateBodyPackage(t *testing.T, packageID hir.PackageID, writeSymbol, readSymbol hir.SymbolID) hir.Package {
	t.Helper()
	result := stateBodyPackage(t, packageID, writeSymbol)
	result.Modules[0].Functions = append(result.Modules[0].Functions, hir.Function{
		Symbol: readSymbol, Name: "observe", ResultType: hir.TypeVoid, Effects: []hir.Effect{hir.EffectStorageRead},
	})
	result.StateTransactions = append(result.StateTransactions, hir.StateTransaction{
		Package: packageID, Function: readSymbol, Exit: hir.StateTransactionReject,
		Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}},
	})
	result.StateTransactions[0], result.StateTransactions[1] = result.StateTransactions[1], result.StateTransactions[0]
	return result
}

func cloneStableStateBodyArtifact(value StableStateBodyArtifact) StableStateBodyArtifact {
	result := value
	if value.Layout != nil {
		layout := *value.Layout
		layout.Fields = slices.Clone(value.Layout.Fields)
		result.Layout = &layout
	}
	result.Transactions = make([]StableStateTransaction, len(value.Transactions))
	for index, transaction := range value.Transactions {
		result.Transactions[index] = transaction
		result.Transactions[index].Function = slices.Clone(transaction.Function)
		result.Transactions[index].Effects = slices.Clone(transaction.Effects)
		result.Transactions[index].Operations = make([]StableStateOperation, len(transaction.Operations))
		for operationIndex, operation := range transaction.Operations {
			result.Transactions[index].Operations[operationIndex] = operation
			result.Transactions[index].Operations[operationIndex].Key.Encoded = slices.Clone(operation.Key.Encoded)
			result.Transactions[index].Operations[operationIndex].Value.Encoded = slices.Clone(operation.Value.Encoded)
		}
	}
	return result
}
