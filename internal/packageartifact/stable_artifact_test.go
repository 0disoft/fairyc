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
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const stablePackageArtifactLimit = 64 << 20

func TestStablePackageArtifactsDecodeRebindExecuteAndReencodeAcrossFreshProcessNumbers(t *testing.T) {
	t.Parallel()
	workspace, original, identities, coordinates := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(original, identities, coordinates)
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
	freshCoordinates := make(map[hir.PackageID]packagegraph.CoordinateID, len(coordinates))
	oldToFresh := make(map[hir.PackageID]hir.PackageID, len(identities))
	for index, old := range oldPackageIDs {
		fresh := hir.PackageID(100 + index)
		freshIdentities[fresh] = identities[old]
		freshCoordinates[fresh] = coordinates[old]
		oldToFresh[old] = fresh
	}
	rebound, err := RebindStableArtifacts(decoded, freshCoordinates, 1000, 10000, 500)
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

	reextracted, err := ExtractAll(rebound, freshIdentities, freshCoordinates)
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
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(program, identities, coordinates)
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
	old := bytes.Replace(encoded, []byte(StableArtifactVersion), []byte("fairyc-stable-package-mir/4"), 1)
	if _, err := DecodeStableArtifact(old, stablePackageArtifactLimit); err == nil {
		t.Fatal("accepted previous lossy stable MIR version")
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

func TestStablePackageArtifactBytesIgnoreUnreferencedWorkspaceDeclarations(t *testing.T) {
	t.Parallel()
	workspace, original, identities, coordinates := checkedWorkspaceProgram(t)
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := ExtractAll(original, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	baselineBytes := encodeStableArtifactsByPackage(t, baseline)

	mutated := mir.Clone(original)
	mutated.StructTypes = append(mutated.StructTypes, mir.StructType{
		Type: hir.ParseType(hir.QualifiedModulePath(rootPackage, "private::CacheNoise")),
		Name: "CacheNoise",
		Span: hir.Span{Package: rootPackage},
	})
	if failures := mir.Verify(mutated); len(failures) != 0 {
		t.Fatalf("verify private-declaration mutation: %v", failures[0])
	}
	changed, err := ExtractAll(mutated, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	changedBytes := encodeStableArtifactsByPackage(t, changed)
	if !reflect.DeepEqual(changedBytes, baselineBytes) {
		t.Fatalf("unreferenced private declaration changed stable package bytes")
	}
}

func TestStablePackageArtifactsMergeDistinctPackageContexts(t *testing.T) {
	t.Parallel()
	workspace, program, identities, coordinates := checkedWorkspaceProgram(t)
	processArtifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	stable := make([]StableArtifact, 0, len(processArtifacts))
	for _, artifact := range processArtifacts {
		projected, describeErr := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
		if describeErr != nil {
			t.Fatal(describeErr)
		}
		stable = append(stable, projected)
	}
	if len(stable) < 2 {
		t.Fatal("test workspace must contain at least two package artifacts")
	}
	marker := ContextStruct{
		Type: "pkg[" + stable[0].Coordinate.String() + "]::private::MergeMarker",
		Name: "MergeMarker",
	}
	stable[0].Context.Structs = append(stable[0].Context.Structs, marker)
	sortContextDescriptor(&stable[0].Context)
	contextID, err := ComputeContextID(stable[0].Context, stablePackageArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	for index := range stable[0].Functions {
		stable[0].Functions[index].ContextID = contextID
	}
	if reflect.DeepEqual(stable[0].Context, stable[1].Context) {
		t.Fatal("test setup did not create distinct package contexts")
	}

	rebound, err := RebindStableArtifacts(stable, coordinates, 1000, 10000, 500)
	if err != nil {
		t.Fatal(err)
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(rebound, hir.QualifiedModulePath(rootPackage, "main")+"::main")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Value != 42 {
		t.Fatalf("mixed-context rebound value = %d, want 42", outcome.Value)
	}
}

func TestMergeContextDescriptorsRejectsConflictingDefinitions(t *testing.T) {
	t.Parallel()
	_, _, _, coordinates := checkedWorkspaceProgram(t)
	var coordinate packagegraph.CoordinateID
	for _, candidate := range coordinates {
		coordinate = candidate
		break
	}
	typeName := "pkg[" + coordinate.String() + "]::shared::Value"
	left := ContextDescriptor{Structs: []ContextStruct{{
		Type: typeName, Name: "Value", Fields: []ContextField{{Name: "value", Type: "i32"}},
	}}}
	right := cloneContextDescriptor(left)
	right.Structs[0].Fields[0].Type = "u32"
	if _, err := MergeContextDescriptors([]ContextDescriptor{left, right}); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("conflicting package contexts error = %v", err)
	}
}

func encodeStableArtifactsByPackage(t *testing.T, artifacts []Artifact) map[packagegraph.InstanceID]string {
	t.Helper()
	result := make(map[packagegraph.InstanceID]string, len(artifacts))
	for _, artifact := range artifacts {
		stable, err := DescribeStableArtifact(artifact, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeStableArtifact(stable, stablePackageArtifactLimit)
		if err != nil {
			t.Fatal(err)
		}
		result[stable.Package] = string(encoded)
	}
	return result
}
