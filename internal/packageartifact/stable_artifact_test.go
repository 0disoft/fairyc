package packageartifact

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const stablePackageArtifactLimit = 64 << 20

func TestStablePackageArtifactsDecodeRebindExecuteAndReencodeAcrossFreshProcessNumbers(t *testing.T) {
	t.Parallel()
	workspace, original, identities := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(original, identities)
	if err != nil {
		t.Fatal(err)
	}
	encodedByPackage := make(map[packagegraph.InstanceID][]byte, len(processArtifacts))
	decoded := make([]StableArtifact, 0, len(processArtifacts))
	for _, artifact := range processArtifacts {
		stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		encodedByPackage[stable.Package] = encoded
		stable, err = DecodeStableArtifact(encoded, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		decoded = append(decoded, stable)
	}

	oldPackageIDs := make([]hir.PackageID, 0, len(identities))
	for packageID := range identities {
		oldPackageIDs = append(oldPackageIDs, packageID)
	}
	slices.Sort(oldPackageIDs)
	freshIdentities := make(map[hir.PackageID]packagegraph.InstanceID, len(identities))
	oldToFresh := make(map[hir.PackageID]hir.PackageID, len(identities))
	for index, old := range oldPackageIDs {
		fresh := hir.PackageID(100 + index)
		freshIdentities[fresh] = identities[old]
		oldToFresh[old] = fresh
	}
	rebound, err := RebindStableArtifacts(decoded, freshIdentities, 1000, 10000, 500)
	if err != nil {
		t.Fatal(err)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	originalEntry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	freshEntry := hir.QualifiedModulePath(oldToFresh[rootPackage], "main") + "::main"
	originalOutcome, err := interpreter.Execute(original, originalEntry)
	if err != nil {
		t.Fatal(err)
	}
	reboundOutcome, err := interpreter.Execute(rebound, freshEntry)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reboundOutcome, originalOutcome) || reboundOutcome.Value != 42 {
		t.Fatalf("rebound stable packages = %+v, original = %+v", reboundOutcome, originalOutcome)
	}

	reextracted, err := ExtractAll(rebound, freshIdentities)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range reextracted {
		stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, encodedByPackage[stable.Package]) {
			t.Fatalf("stable package %s bytes changed after independent decode and rebind", stable.Package)
		}
	}
}

func TestStablePackageArtifactCodecRejectsBoundsUnknownFieldsAndImportCorruption(t *testing.T) {
	t.Parallel()
	_, program, identities := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(program, identities)
	if err != nil {
		t.Fatal(err)
	}
	stable, err := DescribeStableArtifact(processArtifacts[0], stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeStableArtifact(stable, uint64(len(encoded)-1)); !errors.Is(err, ErrStableArtifactTooLarge) {
		t.Fatalf("small encode bound error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1)
	if _, err := DecodeStableArtifact(unknown, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	withWhitespace := append(append([]byte(nil), encoded...), '\n')
	if _, err := DecodeStableArtifact(withWhitespace, stablePackageArtifactLimit); !errors.Is(err, ErrStableArtifactEncoding) {
		t.Fatalf("noncanonical error = %v", err)
	}
	if len(stable.Imports) != 0 {
		corrupt := stable
		corrupt.Imports = slices.Clone(stable.Imports)
		corrupt.Imports[0][0] ^= 0xff
		if _, err := EncodeStableArtifact(corrupt, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "neither owned nor imported") {
			t.Fatalf("corrupt import error = %v", err)
		}
	}
}
