package packageartifact

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestPackageArtifactsReassembleWithFreshSymbolsAndPreserveExecution(t *testing.T) {
	t.Parallel()
	workspace, original, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(original, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) < 2 {
		t.Fatalf("ExtractAll() returned %d artifacts", len(artifacts))
	}
	maxOriginal := hir.SymbolID(0)
	for _, function := range original.Functions {
		maxOriginal = max(maxOriginal, function.Symbol)
	}
	for _, artifact := range artifacts {
		if err := Verify(artifact); err != nil {
			t.Fatalf("Verify(package %d): %v", artifact.PackageID, err)
		}
	}
	reassembled, err := Reassemble(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(reassembled.Functions) != len(original.Functions) {
		t.Fatalf("reassembled %d functions, want %d", len(reassembled.Functions), len(original.Functions))
	}
	for _, function := range reassembled.Functions {
		if function.Symbol <= maxOriginal {
			t.Fatalf("reassembled function %q reused symbol %d at or below original maximum %d", function.Name, function.Symbol, maxOriginal)
		}
	}
	rootPackage, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	entry := hir.QualifiedModulePath(rootPackage, "main") + "::main"
	outcome, err := interpreter.Execute(reassembled, entry)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("reassembled execution = %+v", outcome)
	}
}

func TestPackageArtifactRejectsCorruptBodyAndIdentity(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	corruptBody := cloneArtifactForTest(artifacts[0])
	corruptBody.Functions[0].Body.Entry = 999
	if err := Verify(corruptBody); err == nil || !strings.Contains(err.Error(), "verify package MIR closure") {
		t.Fatalf("corrupt body error = %v", err)
	}
	corruptID := cloneArtifactForTest(artifacts[0])
	corruptID.Functions[0].ID[0] ^= 0xff
	if err := Verify(corruptID); err == nil || !strings.Contains(err.Error(), "identity does not match") {
		t.Fatalf("corrupt identity error = %v", err)
	}
}

func TestPackageArtifactReassemblyRejectsMissingProviderAndContextDrift(t *testing.T) {
	t.Parallel()
	_, program, identities, coordinates := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	consumerIndex := slices.IndexFunc(artifacts, func(artifact Artifact) bool { return len(artifact.Imports) != 0 })
	if consumerIndex < 0 {
		t.Fatal("workspace artifacts have no internal import")
	}
	consumer := artifacts[consumerIndex]
	if _, err := Reassemble([]Artifact{consumer}); err == nil || !strings.Contains(err.Error(), "unresolved") {
		t.Fatalf("missing provider error = %v", err)
	}

	drifted := cloneArtifactsForTest(artifacts)
	driftIndex := 1
	drifted[driftIndex].Context.StructTypes = append(drifted[driftIndex].Context.StructTypes, mir.StructType{
		Type: hir.ParseType(hir.QualifiedModulePath(drifted[driftIndex].PackageID, "drift") + "::Type"),
		Name: "Type", Span: hir.Span{Package: drifted[driftIndex].PackageID}, Entity: "struct-type:drift-type",
	})
	drifted[driftIndex].StableContext, err = DescribeContext(
		drifted[driftIndex].Context,
		drifted[driftIndex].PackageCoordinates,
		artifactBindings(drifted[driftIndex]),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Reassemble(drifted); err == nil || !strings.Contains(err.Error(), "stable type context") {
		t.Fatalf("context drift error = %v", err)
	}
}

func checkedWorkspaceProgram(t *testing.T) (workspaceloader.Workspace, mir.Program, map[hir.PackageID]packagegraph.InstanceID, map[hir.PackageID]packagegraph.CoordinateID) {
	t.Helper()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "examples", "workspace_quickstart"))
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	identities, err := workspace.PackageIdentities()
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatalf("workspace is not verified: %+v", result.Semantic)
	}
	return workspace, result.MIR, identities, coordinates
}

func cloneArtifactsForTest(values []Artifact) []Artifact {
	result := make([]Artifact, len(values))
	for index := range values {
		result[index] = cloneArtifactForTest(values[index])
	}
	return result
}

func cloneArtifactForTest(value Artifact) Artifact {
	result := value
	result.PackageIdentities = cloneIdentities(value.PackageIdentities)
	result.PackageCoordinates = cloneCoordinates(value.PackageCoordinates)
	result.StableContext = cloneContextDescriptor(value.StableContext)
	result.Context = mir.Clone(value.Context)
	result.Functions = make([]FunctionRecord, len(value.Functions))
	for index, record := range value.Functions {
		result.Functions[index] = FunctionRecord{ID: record.ID, Descriptor: cloneDescriptor(record.Descriptor), Body: cloneFunction(record.Body)}
	}
	result.Imports = make([]FunctionImport, len(value.Imports))
	for index, imported := range value.Imports {
		result.Imports[index] = FunctionImport{ID: imported.ID, Descriptor: cloneDescriptor(imported.Descriptor), Signature: cloneSignature(imported.Signature)}
	}
	return result
}
