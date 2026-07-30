package atlasmap

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestPrepareWorkspaceBuildsDeterministicCompletePackagePlans(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	compilerBuild := testCompilerBuild()
	snapshot, first, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatalf("PrepareWorkspace() error = %v", err)
	}
	_, second, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil {
		t.Fatalf("PrepareWorkspace(second) error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("package plans are nondeterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if len(first) != 2 {
		t.Fatalf("partitions = %+v", first)
	}
	if !slices.IsSortedFunc(first, func(left, right Partition) int {
		return strings.Compare(left.Header().PackageInstance, right.Header().PackageInstance)
	}) {
		t.Fatalf("partitions are not package-instance sorted: %+v", first)
	}

	totalFiles := 0
	totalModules := 0
	totalSymbols := 0
	totalFunctions := 0
	for _, partition := range first {
		header := partition.Header()
		if header.LogicalSchema != LogicalSchema || header.SnapshotModel != semanticsnapshot.ModelVersion ||
			header.CompilerBuild != compilerBuild.String() || header.CompilerBuildAlgorithm != buildidentity.Encoding ||
			header.PackageInstanceAlgorithm != packagegraph.InstanceIDEncoding ||
			header.PackageCoordinateAlgorithm != packagegraph.CoordinateIDEncoding ||
			header.PackageSourceAlgorithm != packagecontent.DigestEncoding ||
			header.SourceFileAlgorithm != packagecontent.FileDigestEncoding ||
			header.LineMapAlgorithm != packagecontent.LineMapEncoding ||
			header.SymbolAlgorithm != symbolidentity.Algorithm || header.InterfaceAlgorithm != moduleinterface.HashEncoding ||
			header.TargetScope != TargetScopeIndependent || header.Target != "" || header.GenerationStatus != GenerationPartial {
			t.Fatalf("header compatibility = %+v", header)
		}
		if _, err := packagegraph.ParseInstanceID(header.PackageInstance); err != nil {
			t.Fatalf("package instance = %q: %v", header.PackageInstance, err)
		}
		if _, err := packagegraph.ParseCoordinateID(header.PackageCoordinate); err != nil {
			t.Fatalf("package coordinate = %q: %v", header.PackageCoordinate, err)
		}
		if len(header.PackageReferences) != len(header.Dependencies)+1 || !slices.IsSortedFunc(header.PackageReferences, func(left, right PackageReference) int {
			return strings.Compare(left.Instance, right.Instance)
		}) {
			t.Fatalf("package references = %+v", header.PackageReferences)
		}
		selfReference := false
		for _, reference := range header.PackageReferences {
			if _, err := packagegraph.ParseInstanceID(reference.Instance); err != nil {
				t.Fatalf("reference instance = %q: %v", reference.Instance, err)
			}
			if _, err := packagegraph.ParseCoordinateID(reference.Coordinate); err != nil {
				t.Fatalf("reference coordinate = %q: %v", reference.Coordinate, err)
			}
			if reference.Instance == header.PackageInstance {
				selfReference = reference.UID == header.PackageUID && reference.Coordinate == header.PackageCoordinate
			}
		}
		if !selfReference {
			t.Fatalf("package references lack exact self identity: %+v", header.PackageReferences)
		}
		if _, err := packagecontent.ParseDigest(header.PackageSource); err != nil {
			t.Fatalf("package source = %q: %v", header.PackageSource, err)
		}
		if _, err := moduleinterface.ParseHash(header.InterfaceHash); err != nil {
			t.Fatalf("interface hash = %q: %v", header.InterfaceHash, err)
		}
		if !slices.Contains(header.Omissions, "reads:unavailable:not-implemented") ||
			!slices.Contains(header.Omissions, "test-calls:unavailable:test-metadata-not-retained") ||
			!slices.Contains(header.Omissions, "writes:unavailable:not-implemented") {
			t.Fatalf("header omissions = %+v", header.Omissions)
		}
		if header.Counts.Files != len(partition.Files()) || header.Counts.Modules != len(partition.Modules()) ||
			header.Counts.Symbols != len(partition.Symbols()) || header.Counts.Functions != len(partition.Functions()) ||
			header.Counts.Completeness != len(partition.Completeness()) {
			t.Fatalf("header counts = %+v", header.Counts)
		}
		if len(partition.Symbols()) != len(partition.Declarations()) {
			t.Fatalf("symbol/declaration coverage = %d/%d", len(partition.Symbols()), len(partition.Declarations()))
		}
		for _, file := range partition.Files() {
			if filepath.IsAbs(file.Path) || strings.Contains(file.Path, "\\") {
				t.Fatalf("partition leaked host path: %+v", file)
			}
		}
		for _, symbol := range partition.Symbols() {
			if _, err := symbolidentity.ParseVisibleID(symbol.Visible); err != nil {
				t.Fatalf("visible SymbolId = %q: %v", symbol.Visible, err)
			}
			if _, err := symbolidentity.ParseFullDigest(symbol.Full); err != nil {
				t.Fatalf("full SymbolId = %q: %v", symbol.Full, err)
			}
		}
		totalFiles += header.Counts.Files
		totalModules += header.Counts.Modules
		totalSymbols += header.Counts.Symbols
		totalFunctions += header.Counts.Functions
	}
	if totalFiles != len(snapshot.Files()) || totalModules != len(snapshot.Modules()) ||
		totalSymbols != len(snapshot.Declarations()) || totalFunctions != len(snapshot.Functions()) {
		t.Fatalf("partition totals = files %d modules %d symbols %d functions %d", totalFiles, totalModules, totalSymbols, totalFunctions)
	}
}

func TestDocumentationSidecarDoesNotEnterFymapV1PlanOrEncoding(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `schema = "fairyc.manifest/v1"

[package]
uid = "test.atlas.documentation-sidecar"
name = "documentation-sidecar"
version = "1.0.0"
language = "fairyc-language/5"
source = "src"
`
	sourceText := "module main;\n/// Returns the answer.\nfn i32 main() { return 42; }\n"
	if err := os.WriteFile(filepath.Join(root, "fairy.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.fy"), []byte(sourceText), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil || !result.Verified {
		t.Fatalf("CheckPackages() = verified %t error %v", result.Verified, err)
	}
	compilerBuild := testCompilerBuild()
	withDocumentation, withPlans, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil || len(withDocumentation.DocumentationSpans()) != 1 {
		t.Fatalf("documented PrepareWorkspace() = (%d spans, %v)", len(withDocumentation.DocumentationSpans()), err)
	}
	for moduleIndex := range result.Semantic.HIR.Modules {
		for functionIndex := range result.Semantic.HIR.Modules[moduleIndex].Functions {
			result.Semantic.HIR.Modules[moduleIndex].Functions[functionIndex].DocumentationSpan = hir.Span{}
		}
	}
	withoutDocumentation, withoutPlans, err := PrepareWorkspace(workspace, result, compilerBuild)
	if err != nil || len(withoutDocumentation.DocumentationSpans()) != 0 || len(withPlans) != len(withoutPlans) {
		t.Fatalf("undocumented PrepareWorkspace() = (%d spans, %d/%d plans, %v)", len(withoutDocumentation.DocumentationSpans()), len(withPlans), len(withoutPlans), err)
	}
	for index := range withPlans {
		withBytes, err := Encode(withPlans[index], 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		withoutBytes, err := Encode(withoutPlans[index], 2<<20)
		if err != nil || !bytes.Equal(withBytes, withoutBytes) {
			t.Fatalf(".fymap v1 changed when only the documentation sidecar changed: %v", err)
		}
	}
}

func TestPartitionAccessorsAreDetached(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	_, partitions, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	partition := partitions[0]
	header := partition.Header()
	files := partition.Files()
	symbols := partition.Symbols()
	completeness := partition.Completeness()
	if len(files) == 0 || len(symbols) == 0 || len(completeness) == 0 {
		t.Fatalf("partition lacks mutation fixtures: %+v", partition)
	}
	header.Features = append(header.Features, "corrupted")
	header.PackageReferences[0].UID = "corrupted"
	header.Omissions = nil
	files[0].Path = "corrupted.fy"
	symbols[0].Full = "corrupted"
	completeness[0].Reasons = append(completeness[0].Reasons, "corrupted")
	freshHeader := partition.Header()
	if slices.Contains(freshHeader.Features, "corrupted") || freshHeader.PackageReferences[0].UID == "corrupted" || len(freshHeader.Omissions) == 0 ||
		partition.Files()[0].Path == "corrupted.fy" || partition.Symbols()[0].Full == "corrupted" ||
		slices.Contains(partition.Completeness()[0].Reasons, "corrupted") {
		t.Fatalf("partition accessor mutation escaped into immutable plan")
	}
}

func TestPartitionSemanticRecordAccessorsDeepDetach(t *testing.T) {
	t.Parallel()
	owner := semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypePointer, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}}}
	partition := Partition{data: partitionData{
		declarations: []semanticsnapshot.Declaration{{
			Key:         semanticsnapshot.DeclarationKey{Name: "answer"},
			Type:        semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeResult, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}}},
			Owner:       &owner,
			Parameters:  []semanticsnapshot.Parameter{{Name: "value", Type: semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypePointer, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}}}}},
			Effects:     []string{"io"},
			Fields:      []semanticsnapshot.Field{{Name: "field", Type: semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypePointer, Arguments: []semanticsnapshot.TypeRef{{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}}}}},
			Variants:    []semanticsnapshot.Variant{{Name: "some", Fields: []semanticsnapshot.Field{{Name: "payload", Type: semanticsnapshot.TypeRef{Kind: semanticsnapshot.TypeBuiltin, Name: "i32"}}}}},
			DropEffects: []string{"drop"},
		}},
		functions: []semanticsnapshot.FunctionSummary{{
			Function:       semanticsnapshot.DeclarationKey{Name: "answer"},
			Calls:          []semanticsnapshot.Call{{Target: semanticsnapshot.DeclarationKey{Name: "callee"}}},
			TrapCategories: []string{"overflow"},
			UnsafeSites:    []semanticsnapshot.Span{{Source: "main.fy"}},
			Borrows:        []semanticsnapshot.Borrow{{Kind: "shared"}},
		}},
	}}

	declarations := partition.Declarations()
	functions := partition.Functions()
	declarations[0].Type.Arguments[0].Name = "bool"
	declarations[0].Owner.Arguments[0].Name = "bool"
	declarations[0].Parameters[0].Type.Arguments[0].Name = "bool"
	declarations[0].Effects[0] = "corrupted"
	declarations[0].Fields[0].Type.Arguments[0].Name = "bool"
	declarations[0].Variants[0].Fields[0].Name = "corrupted"
	declarations[0].DropEffects[0] = "corrupted"
	functions[0].Calls[0].Target.Name = "corrupted"
	functions[0].TrapCategories[0] = "corrupted"
	functions[0].UnsafeSites[0].Source = "corrupted.fy"
	functions[0].Borrows[0].Kind = "corrupted"

	freshDeclaration := partition.Declarations()[0]
	freshFunction := partition.Functions()[0]
	if freshDeclaration.Type.Arguments[0].Name != "i32" ||
		freshDeclaration.Owner.Arguments[0].Name != "i32" ||
		freshDeclaration.Parameters[0].Type.Arguments[0].Name != "i32" ||
		freshDeclaration.Effects[0] != "io" || freshDeclaration.Fields[0].Type.Arguments[0].Name != "i32" ||
		freshDeclaration.Variants[0].Fields[0].Name != "payload" || freshDeclaration.DropEffects[0] != "drop" ||
		freshFunction.Calls[0].Target.Name != "callee" || freshFunction.TrapCategories[0] != "overflow" ||
		freshFunction.UnsafeSites[0].Source != "main.fy" || freshFunction.Borrows[0].Kind != "shared" {
		t.Fatalf("nested semantic record mutation escaped into immutable plan")
	}
}

func TestBuildPlanFailsClosedOnMissingOrContradictoryCompatibilityInputs(t *testing.T) {
	t.Parallel()
	workspace, result := loadQuickstart(t)
	snapshot, partitions, err := PrepareWorkspace(workspace, result, testCompilerBuild())
	if err != nil {
		t.Fatal(err)
	}
	interfaces := make(map[string]moduleinterface.Hash, len(partitions))
	for _, partition := range partitions {
		header := partition.Header()
		hash, parseErr := moduleinterface.ParseHash(header.InterfaceHash)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		interfaces[header.PackageInstance] = hash
	}
	if _, err := BuildPlan(snapshot, Input{Interfaces: interfaces}); err == nil || !strings.Contains(err.Error(), "compiler build") {
		t.Fatalf("zero compiler BuildPlan() error = %v", err)
	}
	missing := cloneInterfaces(interfaces)
	for instance := range missing {
		delete(missing, instance)
		break
	}
	if _, err := BuildPlan(snapshot, Input{CompilerBuild: testCompilerBuild(), Interfaces: missing}); err == nil || !strings.Contains(err.Error(), "one public-interface hash") {
		t.Fatalf("missing interface BuildPlan() error = %v", err)
	}
	extra := cloneInterfaces(interfaces)
	extra["fypi1-sha256-"+strings.Repeat("0", 64)] = moduleinterface.Hash{1}
	if _, err := BuildPlan(snapshot, Input{CompilerBuild: testCompilerBuild(), Interfaces: extra}); err == nil || !strings.Contains(err.Error(), "one public-interface hash") {
		t.Fatalf("extra interface BuildPlan() error = %v", err)
	}
	zero := cloneInterfaces(interfaces)
	for instance := range zero {
		zero[instance] = moduleinterface.Hash{}
		break
	}
	if _, err := BuildPlan(snapshot, Input{CompilerBuild: testCompilerBuild(), Interfaces: zero}); err == nil || !strings.Contains(err.Error(), "no exact public-interface hash") {
		t.Fatalf("zero interface BuildPlan() error = %v", err)
	}
}

func loadQuickstart(t *testing.T) (workspaceloader.Workspace, compiler.Result) {
	t.Helper()
	workspace, err := workspaceloader.Load(filepath.Join("..", "..", "examples", "workspace_quickstart"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatalf("SemanticInput() error = %v", err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil || !result.Verified {
		t.Fatalf("CheckPackages() = verified %t error %v", result.Verified, err)
	}
	return workspace, result
}

func testCompilerBuild() buildidentity.Identity {
	var result buildidentity.Identity
	for index := range result {
		result[index] = byte(index + 1)
	}
	return result
}

func cloneInterfaces(values map[string]moduleinterface.Hash) map[string]moduleinterface.Hash {
	result := make(map[string]moduleinterface.Hash, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
