package semanticsnapshot_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	rootSource = `module main;
import dep as dep;

pub fn i32 main() {
    assert(dep::answer() == 42);
    return dep::answer();
}
`
	dependencySource = `module dep;
pub fn i32 answer() { return 42; }
`
)

func TestBuildIsImmutableDeterministicAndIndependentOfBuildLocalPackageNumbers(t *testing.T) {
	t.Parallel()
	leftResult, leftPackages := compileResolvedClosure(t, 1, 2)
	rightResult, rightPackages := compileResolvedClosure(t, 91, 7)

	left, err := semanticsnapshot.Build(leftResult.Semantic, leftResult.MIR, leftPackages)
	if err != nil {
		t.Fatalf("Build(left) error = %v", err)
	}
	right, err := semanticsnapshot.Build(rightResult.Semantic, rightResult.MIR, rightPackages)
	if err != nil {
		t.Fatalf("Build(right) error = %v", err)
	}
	assertSnapshotsEqual(t, left, right)

	packagesBefore := left.Packages()
	filesBefore := left.Files()
	modulesBefore := left.Modules()
	declarationsBefore := left.Declarations()
	functionsBefore := left.Functions()
	completenessBefore := left.Completeness()
	if len(packagesBefore) != 2 || len(filesBefore) != 2 || len(modulesBefore) != 2 || len(declarationsBefore) != 2 || len(functionsBefore) != 2 {
		t.Fatalf("snapshot counts = packages %d files %d modules %d declarations %d functions %d", len(packagesBefore), len(filesBefore), len(modulesBefore), len(declarationsBefore), len(functionsBefore))
	}
	for _, file := range filesBefore {
		if file.Digest == "" || file.LineMap == "" || file.Kind != semanticsnapshot.FileKindSource || strings.Contains(file.Path, "\\") {
			t.Fatalf("snapshot file = %+v", file)
		}
	}
	main := findFunction(t, functionsBefore, "main", "main")
	if len(main.Calls) != 2 || main.Calls[0].TargetKind != semanticsnapshot.CallTargetResolved ||
		main.Calls[0].Certainty != semanticsnapshot.CertaintyExact || main.Calls[0].Target.Module != "dep" ||
		!slices.Contains(main.TrapCategories, "assertion-failed") {
		t.Fatalf("main summary = %+v", main)
	}
	calls := findCompleteness(t, completenessBefore, "calls")
	if calls.Status != semanticsnapshot.CompletenessExact || calls.Resolved != 2 || calls.External != 0 || calls.Unknown != 0 {
		t.Fatalf("call completeness = %+v", calls)
	}

	// Mutating both the original compiler results and every detached accessor result cannot mutate
	// the immutable snapshot.
	leftResult.Semantic.HIR.Modules[0].Path = "corrupted"
	leftResult.MIR.Functions[0].Name = "corrupted"
	leftPackages[0].Files[0].Path = "corrupted-input.fy"
	packagesBefore[0].Features = append(packagesBefore[0].Features, "corrupted")
	filesBefore[0].Path = "corrupted.fy"
	declarationsBefore[0].Effects = append(declarationsBefore[0].Effects, "corrupted")
	functionsBefore[0].Calls = nil
	completenessBefore[0].Reasons = append(completenessBefore[0].Reasons, "corrupted")
	assertSnapshotsEqual(t, left, right)

	if left.Version() != semanticsnapshot.ModelVersion {
		t.Fatalf("snapshot version = %q", left.Version())
	}
	rendered := strings.Join(snapshotStrings(left), "\n")
	if strings.Contains(rendered, "$p1") || strings.Contains(rendered, "$p2") || strings.Contains(rendered, "$p91") || strings.Contains(rendered, "C:\\") {
		t.Fatalf("snapshot leaked build-local or host identity: %q", rendered)
	}
}

func TestBuildRetainsExactDeclarationDocumentationAsDetachedNonDeclarationSidecar(t *testing.T) {
	t.Parallel()
	text := `module docs;
/// Packet docs.
struct Packet { i32 value; }
/// Error docs.
enum Error { Failed; }
/// Handle docs.
resource Handle { ptr void raw; cleanup {} }
/// Foreign docs.
unsafe extern C fn i32 raw(i32 value);
/// Function docs.
fn i32 run() { return 0; }
`
	result := compileSinglePackage(t, hir.DefaultPackage, "docs.fy", text)
	input := packageInput(t, hir.DefaultPackage, "example.snapshot.docs", "docs", "1.0.0", "docs.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	documentation := snapshot.DocumentationSpans()
	if len(documentation) != 5 || len(snapshot.Declarations()) != 5 {
		t.Fatalf("documentation/declarations = %d/%d: %+v", len(documentation), len(snapshot.Declarations()), documentation)
	}
	wants := map[semanticsnapshot.DeclarationKind]string{
		semanticsnapshot.DeclarationStruct:          "/// Packet docs.",
		semanticsnapshot.DeclarationEnum:            "/// Error docs.",
		semanticsnapshot.DeclarationResource:        "/// Handle docs.",
		semanticsnapshot.DeclarationForeignFunction: "/// Foreign docs.",
		semanticsnapshot.DeclarationFunction:        "/// Function docs.",
	}
	for _, item := range documentation {
		want, found := wants[item.Declaration.Kind]
		if !found || item.DeclarationSpan.PackageInstance != input.Instance.String() ||
			item.Documentation.PackageInstance != input.Instance.String() ||
			item.Documentation.Source != "docs.fy" || item.Documentation.EndByte > item.DeclarationSpan.StartByte ||
			text[item.Documentation.StartByte:item.Documentation.EndByte] != want {
			t.Fatalf("documentation sidecar = %+v, want %q", item, want)
		}
	}
	documentation[0].Documentation.Source = "corrupted.fy"
	if snapshot.DocumentationSpans()[0].Documentation.Source == "corrupted.fy" {
		t.Fatal("documentation accessor mutation escaped into immutable snapshot")
	}

	mutated := result.Semantic
	mutated.HIR.Modules[0].Functions[0].DocumentationSpan.EndByte = mutated.HIR.Modules[0].Functions[0].Span.StartByte + 1
	if _, err := semanticsnapshot.Build(mutated, result.MIR, []semanticsnapshot.PackageInput{input}); err == nil ||
		!strings.Contains(err.Error(), "not a preceding span") {
		t.Fatalf("overlapping documentation Build() error = %v", err)
	}
}

func TestBuildOmitsModuleConstantsWithoutReclassifyingThem(t *testing.T) {
	t.Parallel()
	text := `module constants;
@const_eval
fn i32 square(i32 value) { return value * value; }
const i32 ANSWER = square(6) + 6;
static_assert(ANSWER == 42);
pub fn i32 answer() { return ANSWER; }
`
	result := compileSinglePackage(t, hir.DefaultPackage, "constants.fy", text)
	input := packageInput(t, hir.DefaultPackage, "example.snapshot.constants", "constants", "1.0.0", "constants.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	declarations := snapshot.Declarations()
	if len(declarations) != 2 {
		t.Fatalf("declarations = %+v, want only the two ordinary functions", declarations)
	}
	findDeclaration(t, declarations, semanticsnapshot.DeclarationFunction, "square")
	findDeclaration(t, declarations, semanticsnapshot.DeclarationFunction, "answer")
	if strings.Contains(strings.Join(snapshotStrings(snapshot), "\n"), "ANSWER") {
		t.Fatalf("snapshot reclassified or exposed module constant: %+v", declarations)
	}
}

func TestBuildMarksForeignCallsOpaqueWithoutDegradingLocalRelations(t *testing.T) {
	t.Parallel()
	text := `module main;
@link_name("fixture_answer")
unsafe extern C fn i32 answer();
fn i32 main() { return unsafe { answer() }; }
`
	result := compileSinglePackage(t, hir.DefaultPackage, "main.fy", text)
	packages := []semanticsnapshot.PackageInput{packageInput(t, hir.DefaultPackage, "example.snapshot.foreign", "foreign", "1.0.0", "main.fy", text, "root")}
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, packages)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	main := findFunction(t, snapshot.Functions(), "main", "main")
	if len(main.Calls) != 1 || main.Calls[0].TargetKind != semanticsnapshot.CallTargetExternal ||
		main.Calls[0].Certainty != semanticsnapshot.CertaintyOpaque || main.Calls[0].ABI != "C" ||
		main.Calls[0].LinkName != "fixture_answer" || len(main.UnsafeSites) != 1 {
		t.Fatalf("foreign main summary = %+v", main)
	}
	calls := findCompleteness(t, snapshot.Completeness(), "calls")
	if calls.Status != semanticsnapshot.CompletenessPartial || calls.Resolved != 0 || calls.External != 1 ||
		!reflect.DeepEqual(calls.Reasons, []string{"external-ffi"}) {
		t.Fatalf("foreign call completeness = %+v", calls)
	}
	if unsafe := findCompleteness(t, snapshot.Completeness(), "uses-unsafe"); unsafe.Status != semanticsnapshot.CompletenessExact {
		t.Fatalf("unsafe completeness = %+v", unsafe)
	}
}

func TestBuildRetainsStructuredDeclarationTypesAndHidesCompilerDropGlue(t *testing.T) {
	t.Parallel()
	text := `module api;
pub struct Packet { i32 value; }
pub resource Handle {
    ptr void raw;
    cleanup {}
}

pub unsafe fn i32 inspect(ref Packet packet, ptr mut u8 output, u8[4] bytes, Option<Packet> maybe) {
    return 0;
}
`
	result := compileSinglePackage(t, hir.DefaultPackage, "api.fy", text)
	input := packageInput(t, hir.DefaultPackage, "example.snapshot.types", "types", "1.0.0", "api.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	declarations := snapshot.Declarations()
	if len(declarations) != 3 {
		t.Fatalf("declarations = %+v", declarations)
	}
	packet := findDeclaration(t, declarations, semanticsnapshot.DeclarationStruct, "Packet")
	if packet.Type.Kind != semanticsnapshot.TypeNominal || packet.Type.Name != "api::Packet" ||
		packet.Type.PackageCoordinate != input.Coordinate.String() || len(packet.Fields) != 1 ||
		packet.Fields[0].Type.Kind != semanticsnapshot.TypeBuiltin || packet.Fields[0].Type.Name != "i32" {
		t.Fatalf("Packet declaration = %+v", packet)
	}
	handle := findDeclaration(t, declarations, semanticsnapshot.DeclarationResource, "Handle")
	if handle.DropKind != string(hir.ResourceDropEmpty) || len(handle.Fields) != 1 ||
		handle.Fields[0].Type.Kind != semanticsnapshot.TypePointer || handle.Fields[0].Type.Mutable ||
		len(handle.Fields[0].Type.Arguments) != 1 || handle.Fields[0].Type.Arguments[0].Name != "void" {
		t.Fatalf("Handle declaration = %+v", handle)
	}
	inspect := findDeclaration(t, declarations, semanticsnapshot.DeclarationFunction, "inspect")
	if !inspect.Public || !inspect.Unsafe || len(inspect.Parameters) != 4 ||
		inspect.Parameters[0].Type.Kind != semanticsnapshot.TypeBorrow || inspect.Parameters[0].Type.Mutable ||
		inspect.Parameters[0].Type.Arguments[0].Name != "api::Packet" ||
		inspect.Parameters[1].Type.Kind != semanticsnapshot.TypePointer || !inspect.Parameters[1].Type.Mutable ||
		inspect.Parameters[2].Type.Kind != semanticsnapshot.TypeFixedArray || inspect.Parameters[2].Type.Length != 4 ||
		inspect.Parameters[3].Type.Kind != semanticsnapshot.TypeOption || inspect.Parameters[3].Type.Arguments[0].Name != "api::Packet" {
		t.Fatalf("inspect declaration = %+v", inspect)
	}
	if len(snapshot.Functions()) != 1 || snapshot.Functions()[0].Function.Name != "inspect" {
		t.Fatalf("function summaries expose compiler drop glue: %+v", snapshot.Functions())
	}
}

func TestBuildAttachesBorrowFactsToTheirSourceFunction(t *testing.T) {
	t.Parallel()
	text := `module main;

fn i32 read(ref i32 value) {
    return 1;
}

fn i32 update() {
    var i32 value = 1;
    let ref i32 view = ref value;
    let i32 before = read(view);
    value = 2;
    return before + value;
}
`
	result := compileSinglePackage(t, hir.DefaultPackage, "main.fy", text)
	input := packageInput(t, hir.DefaultPackage, "example.snapshot.borrow", "borrow", "1.0.0", "main.fy", text, "root")
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	read := findFunction(t, snapshot.Functions(), "main", "read")
	if len(read.Borrows) != 0 {
		t.Fatalf("read borrows = %+v", read.Borrows)
	}
	update := findFunction(t, snapshot.Functions(), "main", "update")
	if len(update.Borrows) != 1 || update.Borrows[0].Kind != "shared" || update.Borrows[0].Escapes ||
		update.Borrows[0].Span.Source != "main.fy" {
		t.Fatalf("update borrows = %+v", update.Borrows)
	}
	if completeness := findCompleteness(t, snapshot.Completeness(), "borrows"); completeness.Status != semanticsnapshot.CompletenessExact {
		t.Fatalf("borrow completeness = %+v", completeness)
	}
}

func TestBuildConsumesTheRealWorkspaceMetadataBoundary(t *testing.T) {
	t.Parallel()
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
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatalf("SemanticPackages() error = %v", err)
	}
	fileMetadata, err := workspace.SemanticFiles()
	if err != nil {
		t.Fatalf("SemanticFiles() error = %v", err)
	}
	filesByPackage := make(map[hir.PackageID][]semanticsnapshot.FileInput)
	for _, file := range fileMetadata {
		filesByPackage[file.Package] = append(filesByPackage[file.Package], semanticsnapshot.FileInput{
			Path: file.Path, Digest: file.Digest, LineMap: file.LineMap, Kind: semanticsnapshot.FileKind(file.Kind),
		})
	}
	packages := make([]semanticsnapshot.PackageInput, 0, len(metadata))
	for _, value := range metadata {
		packages = append(packages, semanticsnapshot.PackageInput{
			BuildID: value.ID, UID: value.PackageUID, Name: value.Name, Version: value.Version,
			Language: value.Language, Instance: value.Instance, Coordinate: value.Coordinate,
			Source: value.Source, SourceKind: value.SourceKind, RuntimeABI: value.RuntimeABI,
			Role: value.Role, Features: slices.Clone(value.Features), Dependencies: slices.Clone(value.Dependencies),
			Files: slices.Clone(filesByPackage[value.ID]),
		})
	}
	snapshot, err := semanticsnapshot.Build(result.Semantic, result.MIR, packages)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(snapshot.Packages()) != 2 || len(snapshot.Files()) != 2 || len(snapshot.Modules()) != 2 || len(snapshot.Functions()) != 2 {
		t.Fatalf("workspace snapshot counts = packages %d files %d modules %d functions %d", len(snapshot.Packages()), len(snapshot.Files()), len(snapshot.Modules()), len(snapshot.Functions()))
	}
	for _, module := range snapshot.Modules() {
		if strings.HasPrefix(module.Path, "tests::") {
			t.Fatalf("ordinary workspace snapshot included isolated test module %+v", module)
		}
	}
}

func TestBuildCanonicalizesAndRequiresCompleteFileIdentityCoverage(t *testing.T) {
	t.Parallel()
	texts := map[string]string{
		"a.fy": "module a;\nfn i32 alpha() { return 1; }\n",
		"z.fy": "module z;\nfn i32 zed() { return 2; }\n",
	}
	result, err := compiler.CheckPackages(semantic.PackageInput{Files: []semantic.File{
		{Package: hir.DefaultPackage, Path: "z.fy", Module: analyzeSyntax(t, "z.fy", texts["z.fy"])},
		{Package: hir.DefaultPackage, Path: "a.fy", Module: analyzeSyntax(t, "a.fy", texts["a.fy"])},
	}})
	if err != nil || !result.Verified {
		t.Fatalf("CheckPackages() = verified %t error %v", result.Verified, err)
	}
	digestFiles := []packagecontent.File{
		{Path: "z.fy", Bytes: []byte(texts["z.fy"])},
		{Path: "a.fy", Bytes: []byte(texts["a.fy"])},
	}
	digest, err := packagecontent.Compute(digestFiles)
	if err != nil {
		t.Fatal(err)
	}
	identity := packagegraph.Identity{PackageUID: "example.snapshot.files", SourceIdentity: digest.String(), VersionIdentity: "1.0.0"}
	instance, err := packagegraph.ComputeInstanceID(identity)
	if err != nil {
		t.Fatal(err)
	}
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(identity))
	if err != nil {
		t.Fatal(err)
	}
	files := make([]semanticsnapshot.FileInput, 0, len(digestFiles))
	for _, file := range digestFiles {
		fileDigest, lineMap := packagecontent.ComputeFileIdentities(file.Bytes)
		files = append(files, semanticsnapshot.FileInput{Path: file.Path, Digest: fileDigest, LineMap: lineMap, Kind: semanticsnapshot.FileKindSource})
	}
	input := semanticsnapshot.PackageInput{
		BuildID: hir.DefaultPackage, UID: identity.PackageUID, Name: "files", Version: identity.VersionIdentity,
		Language: "fairyc-language/7", Instance: instance, Coordinate: coordinate, Source: digest,
		SourceKind: "local", Role: "root", Files: files,
	}
	left, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{input})
	if err != nil {
		t.Fatalf("Build(left) error = %v", err)
	}
	rightInput := input
	rightInput.Files = slices.Clone(files)
	slices.Reverse(rightInput.Files)
	right, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{rightInput})
	if err != nil {
		t.Fatalf("Build(right) error = %v", err)
	}
	assertSnapshotsEqual(t, left, right)
	if actual := left.Files(); len(actual) != 2 || actual[0].Path != "a.fy" || actual[1].Path != "z.fy" {
		t.Fatalf("canonical files = %+v", actual)
	}
	missing := input
	missing.Files = slices.Clone(files[:1])
	if _, err := semanticsnapshot.Build(result.Semantic, result.MIR, []semanticsnapshot.PackageInput{missing}); err == nil || !strings.Contains(err.Error(), "typed module has no exact file identity") {
		t.Fatalf("partial-file Build() error = %v", err)
	}
}

func TestBuildFailsClosedBeforePublishingUnverifiedOrUnmappedFacts(t *testing.T) {
	t.Parallel()
	result, packages := compileResolvedClosure(t, 1, 2)

	if _, err := semanticsnapshot.Build(semantic.Result{}, result.MIR, packages); err == nil || !strings.Contains(err.Error(), "complete Flow") {
		t.Fatalf("incomplete semantic Build() error = %v", err)
	}
	if _, err := semanticsnapshot.Build(result.Semantic, result.MIR, packages[1:]); err == nil || !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("missing package Build() error = %v", err)
	}
	extra := packageInput(t, 99, "example.snapshot.extra", "extra", "1.0.0", "extra.fy", "module extra;\n", "dependency")
	if _, err := semanticsnapshot.Build(result.Semantic, result.MIR, append(slices.Clone(packages), extra)); err == nil || !strings.Contains(err.Error(), "has no typed source module") {
		t.Fatalf("extra package Build() error = %v", err)
	}

	mutated := cloneProgram(result.MIR)
	changed := false
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			terminator := &mutated.Functions[functionIndex].Blocks[blockIndex].Terminator
			if terminator.Kind == mir.TerminatorCall {
				terminator.Callee = hir.SymbolID(1 << 30)
				changed = true
				break
			}
		}
		if changed {
			break
		}
	}
	if !changed {
		t.Fatal("test program has no call terminator to mutate")
	}
	if _, err := semanticsnapshot.Build(result.Semantic, mutated, packages); err == nil || !strings.Contains(err.Error(), "verified MIR") {
		t.Fatalf("unverified MIR Build() error = %v", err)
	}

	standaloneText := "module main;\nfn i32 main() { return 0; }\n"
	standalone := compileSinglePackage(t, hir.DefaultPackage, "main.fy", standaloneText)
	standaloneProgram := cloneProgram(standalone.MIR)
	standaloneProgram.Functions = nil
	standalonePackage := packageInput(t, hir.DefaultPackage, "example.snapshot.missing", "missing", "1.0.0", "main.fy", standaloneText, "root")
	if failures := mir.Verify(standaloneProgram); len(failures) != 0 {
		t.Fatalf("function-elided MIR unexpectedly fails its own structural verifier: %+v", failures)
	}
	if _, err := semanticsnapshot.Build(standalone.Semantic, standaloneProgram, []semanticsnapshot.PackageInput{standalonePackage}); err == nil || !strings.Contains(err.Error(), "has no verified MIR summary") {
		t.Fatalf("function-elided Build() error = %v", err)
	}

	missingFile := standalonePackage
	missingFile.Files = nil
	if _, err := semanticsnapshot.Build(standalone.Semantic, standalone.MIR, []semanticsnapshot.PackageInput{missingFile}); err == nil || !strings.Contains(err.Error(), "has no file identities") {
		t.Fatalf("missing-file Build() error = %v", err)
	}
	extraFile := standalonePackage
	extraDigest, extraLineMap := packagecontent.ComputeFileIdentities([]byte("module extra;\n"))
	extraFile.Files = append(slices.Clone(extraFile.Files), semanticsnapshot.FileInput{
		Path: "extra.fy", Digest: extraDigest, LineMap: extraLineMap, Kind: semanticsnapshot.FileKindSource,
	})
	if _, err := semanticsnapshot.Build(standalone.Semantic, standalone.MIR, []semanticsnapshot.PackageInput{extraFile}); err == nil || !strings.Contains(err.Error(), "has no typed source module") {
		t.Fatalf("extra-file Build() error = %v", err)
	}
}

func compileResolvedClosure(t *testing.T, rootID, dependencyID hir.PackageID) (compiler.Result, []semanticsnapshot.PackageInput) {
	t.Helper()
	rootAST := analyzeSyntax(t, "main.fy", rootSource)
	dependencyAST := analyzeSyntax(t, "dep.fy", dependencySource)
	result, err := compiler.CheckPackages(semantic.PackageInput{
		Files: []semantic.File{
			{Package: rootID, Path: "main.fy", Module: rootAST},
			{Package: dependencyID, Path: "dep.fy", Module: dependencyAST},
		},
		Imports: []semantic.PackageImport{{FromPackage: rootID, ImportPath: "dep", ToPackage: dependencyID}},
	})
	if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		t.Fatalf("CheckPackages() = verified %t error %v diagnostics %+v unsupported %+v", result.Verified, err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	root := packageInput(t, rootID, "example.snapshot.app", "app", "1.0.0", "main.fy", rootSource, "root")
	dependency := packageInput(t, dependencyID, "example.snapshot.dep", "dep", "1.0.0", "dep.fy", dependencySource, "dependency")
	root.Dependencies = []hir.PackageID{dependencyID}
	return result, []semanticsnapshot.PackageInput{dependency, root}
}

func compileSinglePackage(t *testing.T, packageID hir.PackageID, path, text string) compiler.Result {
	t.Helper()
	result, err := compiler.CheckPackages(semantic.PackageInput{Files: []semantic.File{{Package: packageID, Path: path, Module: analyzeSyntax(t, path, text)}}})
	if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		t.Fatalf("CheckPackages() = verified %t error %v diagnostics %+v unsupported %+v", result.Verified, err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	return result
}

func analyzeSyntax(t *testing.T, path, text string) *ast.Module {
	t.Helper()
	result, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
	if err != nil || result.AST == nil || len(result.Tree.Diagnostics) != 0 {
		t.Fatalf("AnalyzeSyntax(%s) = error %v diagnostics %+v", path, err, result.Tree.Diagnostics)
	}
	return result.AST
}

func packageInput(t *testing.T, id hir.PackageID, uid, name, version, path, text, role string) semanticsnapshot.PackageInput {
	t.Helper()
	digest, err := packagecontent.Compute([]packagecontent.File{{Path: path, Bytes: []byte(text)}})
	if err != nil {
		t.Fatal(err)
	}
	identity := packagegraph.Identity{PackageUID: uid, SourceIdentity: digest.String(), VersionIdentity: version}
	instance, err := packagegraph.ComputeInstanceID(identity)
	if err != nil {
		t.Fatal(err)
	}
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(identity))
	if err != nil {
		t.Fatal(err)
	}
	fileDigest, lineMap := packagecontent.ComputeFileIdentities([]byte(text))
	return semanticsnapshot.PackageInput{
		BuildID: id, UID: uid, Name: name, Version: version, Language: "fairyc-language/7",
		Instance: instance, Coordinate: coordinate, Source: digest, SourceKind: "local", Role: role,
		Files: []semanticsnapshot.FileInput{{Path: path, Digest: fileDigest, LineMap: lineMap, Kind: semanticsnapshot.FileKindSource}},
	}
}

func findFunction(t *testing.T, values []semanticsnapshot.FunctionSummary, module, name string) semanticsnapshot.FunctionSummary {
	t.Helper()
	for _, value := range values {
		if value.Function.Module == module && value.Function.Name == name {
			return value
		}
	}
	t.Fatalf("function %s::%s missing from %+v", module, name, values)
	return semanticsnapshot.FunctionSummary{}
}

func findDeclaration(t *testing.T, values []semanticsnapshot.Declaration, kind semanticsnapshot.DeclarationKind, name string) semanticsnapshot.Declaration {
	t.Helper()
	for _, value := range values {
		if value.Key.Kind == kind && value.Key.Name == name {
			return value
		}
	}
	t.Fatalf("declaration %s %s missing from %+v", kind, name, values)
	return semanticsnapshot.Declaration{}
}

func findCompleteness(t *testing.T, values []semanticsnapshot.Completeness, relation string) semanticsnapshot.Completeness {
	t.Helper()
	for _, value := range values {
		if value.Relation == relation {
			return value
		}
	}
	t.Fatalf("completeness %q missing from %+v", relation, values)
	return semanticsnapshot.Completeness{}
}

func assertSnapshotsEqual(t *testing.T, left, right semanticsnapshot.Snapshot) {
	t.Helper()
	if !reflect.DeepEqual(left.Packages(), right.Packages()) || !reflect.DeepEqual(left.Files(), right.Files()) || !reflect.DeepEqual(left.Modules(), right.Modules()) ||
		!reflect.DeepEqual(left.Declarations(), right.Declarations()) || !reflect.DeepEqual(left.DocumentationSpans(), right.DocumentationSpans()) ||
		!reflect.DeepEqual(left.Functions(), right.Functions()) ||
		!reflect.DeepEqual(left.Completeness(), right.Completeness()) {
		t.Fatalf("snapshots differ:\nleft=%s\nright=%s", strings.Join(snapshotStrings(left), "\n"), strings.Join(snapshotStrings(right), "\n"))
	}
}

func snapshotStrings(value semanticsnapshot.Snapshot) []string {
	return []string{
		fmt.Sprintf("%+v", value.Packages()),
		fmt.Sprintf("%+v", value.Files()),
		fmt.Sprintf("%+v", value.Modules()),
		fmt.Sprintf("%+v", value.Declarations()),
		fmt.Sprintf("%+v", value.DocumentationSpans()),
		fmt.Sprintf("%+v", value.Functions()),
		fmt.Sprintf("%+v", value.Completeness()),
	}
}

func cloneProgram(value mir.Program) mir.Program {
	result := value
	result.StructTypes = slices.Clone(value.StructTypes)
	result.EnumTypes = slices.Clone(value.EnumTypes)
	result.ResourceTypes = slices.Clone(value.ResourceTypes)
	result.ForeignFunctions = slices.Clone(value.ForeignFunctions)
	result.Functions = slices.Clone(value.Functions)
	for functionIndex := range result.Functions {
		result.Functions[functionIndex].Blocks = slices.Clone(value.Functions[functionIndex].Blocks)
		for blockIndex := range result.Functions[functionIndex].Blocks {
			result.Functions[functionIndex].Blocks[blockIndex].Operations = slices.Clone(value.Functions[functionIndex].Blocks[blockIndex].Operations)
		}
	}
	return result
}
