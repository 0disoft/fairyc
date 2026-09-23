package contextpack

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestContextPackEncodersAreDeterministicBoundedAndTextFree(t *testing.T) {
	t.Parallel()
	fixture := loadContextPackFixture(t)
	budget := contextPackQueryBudget(3)

	first, err := EncodeOverview(fixture.index, budget, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeOverview(fixture.index, budget, 16<<20)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("EncodeOverview() is nondeterministic: %v", err)
	}
	if !json.Valid(first) || bytes.Contains(first, []byte(fixture.root)) || bytes.Contains(first, []byte("return leaf")) {
		t.Fatalf("overview pack leaked invalid, host-path, or source-text data: %s", first)
	}
	var overview packEnvelope
	if err := json.Unmarshal(first, &overview); err != nil {
		t.Fatal(err)
	}
	assertEnvelopeIntegrity(t, overview)
	if overview.Pack.Query.Kind != QueryOverview || overview.Pack.Facts.Overview == nil ||
		overview.Pack.Facts.Show != nil || overview.Pack.Facts.Around != nil ||
		len(overview.Pack.Documentation) != 0 || len(overview.Pack.SourceExcerpts) != 0 || len(overview.Pack.Warnings) != 0 {
		t.Fatalf("overview envelope shape = %+v", overview)
	}

	request := atlasmap.RelationRequest{
		Direction: atlasmap.DirectionBoth, Relations: []atlasmap.RelationKind{atlasmap.RelationCalls}, Budget: budget,
	}
	showBytes, err := EncodeShow(fixture.index, fixture.symbol, request, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	var show packEnvelope
	if err := json.Unmarshal(showBytes, &show); err != nil {
		t.Fatal(err)
	}
	assertEnvelopeIntegrity(t, show)
	if show.Pack.Query.Kind != QueryShow || show.Pack.Query.Subject != fixture.symbol ||
		show.Pack.Facts.Show == nil || show.Pack.Facts.Overview != nil || show.Pack.Facts.Around != nil ||
		len(show.Pack.Relations) == 0 {
		t.Fatalf("show envelope shape = %+v", show)
	}

	aroundBytes, err := EncodeAround(fixture.index, fixture.symbol, request, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	var around packEnvelope
	if err := json.Unmarshal(aroundBytes, &around); err != nil {
		t.Fatal(err)
	}
	assertEnvelopeIntegrity(t, around)
	if around.Pack.Query.Kind != QueryAround || around.Pack.Facts.Around == nil ||
		around.Pack.Facts.Show != nil || around.Pack.Facts.Overview != nil ||
		len(around.Pack.Facts.Around.Nodes) < 2 || around.Pack.Usage.Depth == 0 {
		t.Fatalf("around envelope shape = %+v", around)
	}
	if reflect.DeepEqual(show.Checksum, around.Checksum) {
		t.Fatal("distinct query kinds produced the same Context Pack checksum")
	}

	if _, err := EncodeOverview(fixture.index, budget, uint64(len(first)-1)); !errors.Is(err, ErrPackTooLarge) {
		t.Fatalf("oversize pack error = %v", err)
	}
	if _, err := EncodeOverview(fixture.index, budget, 0); err == nil {
		t.Fatal("zero pack byte limit succeeded")
	}
}

func TestContextPackPropagatesExactQueryFailures(t *testing.T) {
	t.Parallel()
	fixture := loadContextPackFixture(t)
	request := atlasmap.RelationRequest{
		Direction: atlasmap.DirectionOutgoing, Relations: []atlasmap.RelationKind{atlasmap.RelationCalls},
		Budget: contextPackQueryBudget(1),
	}
	missing := "fysym1-sha256-128-" + strings.Repeat("0", 32)
	if _, err := EncodeShow(fixture.index, missing, request, 1<<20); !errors.Is(err, atlasmap.ErrSymbolNotFound) {
		t.Fatalf("missing symbol error = %v", err)
	}
	request.Direction = "sideways"
	if _, err := EncodeAround(fixture.index, fixture.symbol, request, 1<<20); err == nil {
		t.Fatal("invalid query direction produced a pack")
	}
	if _, err := ParseChecksum("fyctx1-sha256-" + strings.Repeat("A", 64)); err == nil {
		t.Fatal("uppercase Context Pack checksum succeeded")
	}
}

func assertEnvelopeIntegrity(t *testing.T, envelope packEnvelope) {
	t.Helper()
	if envelope.PackVersion != PackVersion || envelope.Encoding != PackEncoding ||
		envelope.ChecksumAlgorithm != ChecksumAlgorithm {
		t.Fatalf("envelope versions = %+v", envelope)
	}
	stored, err := ParseChecksum(envelope.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	computed := independentlyComputeChecksum(t, envelope.Pack)
	if stored != computed {
		t.Fatalf("checksum = %s, computed = %s", stored, computed)
	}
	if envelope.Pack.MapVersion != atlasmap.LogicalSchema || envelope.Pack.PackageIdentity.UID != "test.context-pack" ||
		envelope.Pack.PackageIdentity.Instance == "" || envelope.Pack.PackageIdentity.Coordinate == "" ||
		envelope.Pack.PackageIdentity.Source == "" || envelope.Pack.Compatibility.CompilerBuild == "" ||
		envelope.Pack.Compatibility.InterfaceHash == "" {
		t.Fatalf("pack identity = %+v, compatibility = %+v", envelope.Pack.PackageIdentity, envelope.Pack.Compatibility)
	}
	if envelope.Pack.Usage.Records > envelope.Pack.Budget.MaxRecords ||
		envelope.Pack.Usage.Relations > envelope.Pack.Budget.MaxRelations ||
		envelope.Pack.Usage.Depth > envelope.Pack.Budget.MaxDepth ||
		envelope.Pack.Usage.SelectionBytes > envelope.Pack.Budget.MaxSelectionBytes {
		t.Fatalf("pack usage exceeds query budget: %+v / %+v", envelope.Pack.Usage, envelope.Pack.Budget)
	}
}

func independentlyComputeChecksum(t *testing.T, document packDocument) Checksum {
	t.Helper()
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("fairyc.context-pack-checksum"))
	_, _ = digest.Write([]byte{0, 1})
	for _, value := range [][]byte{[]byte(PackVersion), []byte(PackEncoding), payload} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = digest.Write(length[:])
		_, _ = digest.Write(value)
	}
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result
}

type contextPackFixture struct {
	root   string
	index  *atlasmap.QueryIndex
	symbol string
}

func loadContextPackFixture(t *testing.T) contextPackFixture {
	return loadContextPackFixtureVersion(t, false)
}

func loadContextPackFixtureVersion(t *testing.T, generic bool) contextPackFixture {
	return loadContextPackFixtureMode(t, generic, false)
}

func loadContextPackFixtureMode(t *testing.T, generic, aggregate bool) contextPackFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeContextPackFixture(t, filepath.Join(root, "fairy.toml"), `schema = "fairyc.manifest/v1"

[package]
uid = "test.context-pack"
name = "context-pack"
version = "1.0.0"
language = "fairyc-language/18"
source = "src"
`)
	writeContextPackFixture(t, filepath.Join(root, "src", "main.fy"), `module main;

fn i32 leaf() {
    return 40;
}

fn i32 middle() {
    return leaf() + 1;
}

fn i32 main() {
    return middle() + 1;
}
`)
	if generic {
		writeContextPackFixture(t, filepath.Join(root, "src", "main.fy"), "module main; fn T leaf<T>(T x) { let ref T view = ref x; return *view; } fn T middle<T>(T x) { return leaf<T>(x); } fn T unused<T>(T x) { return x; } fn i32 main() { assert(middle<i32>(42) == 42); assert(middle<u32>(7) == 7); return 0; }")
	}
	if aggregate {
		writeContextPackFixture(t, filepath.Join(root, "src", "main.fy"), "module main; struct Cell<T> { T value; } resource Token<T> { Cell<T> payload; cleanup { let T copy = this.payload.value; let ref T view = ref copy; let T observed = leaf<T>(*view); } } fn T leaf<T>(T x) { return x; } fn Token<T> make<T>(T value) { return Token<T> { .payload = Cell<T> { .value = value } }; } fn i32 main() { let Token<i32> a = make<i32>(42); let Token<u32> b = make<u32>(7); return a.payload.value; }")
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	var result compiler.Result
	if generic {
		coordinates, coordinateErr := workspace.PackageCoordinates()
		if coordinateErr != nil {
			t.Fatal(coordinateErr)
		}
		result, _, err = compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	} else {
		result, err = compiler.CheckPackages(input)
	}
	if err != nil || !result.Verified {
		t.Fatalf("CheckPackages() = verified %t error %v", result.Verified, err)
	}
	compilerBuild := contextPackCompilerBuild()
	prepare, current, query := atlasmap.PrepareWorkspace, atlasmap.CurrentWorkspaceExpectations, atlasmap.NewQueryIndex
	if generic {
		prepare, current, query = atlasmap.PrepareWorkspaceV2, atlasmap.CurrentWorkspaceExpectationsV2, atlasmap.NewQueryIndexV2
	}
	if aggregate {
		prepare, current, query = atlasmap.PrepareWorkspaceV3, atlasmap.CurrentWorkspaceExpectationsV3, atlasmap.NewQueryIndexV3
	}
	_, partitions, err := prepare(workspace, result, compilerBuild)
	if err != nil || len(partitions) != 1 {
		t.Fatalf("PrepareWorkspace() = %d, %v", len(partitions), err)
	}
	expectations, err := current(workspace, result, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	partition := partitions[0]
	index, err := query(partition, expectations[partition.Header().PackageInstance])
	if err != nil {
		t.Fatal(err)
	}
	symbols := make(map[semanticsnapshot.DeclarationKey]string)
	for _, symbol := range partition.Symbols() {
		symbols[symbol.Declaration] = symbol.Visible
		if aggregate && symbol.Declaration.Name == "Token" || generic && !aggregate && symbol.Declaration.Name == "middle" {
			return contextPackFixture{root: root, index: index, symbol: symbol.Visible}
		}
	}
	for _, function := range partition.Functions() {
		if len(function.Calls) != 0 {
			return contextPackFixture{root: root, index: index, symbol: symbols[function.Function]}
		}
	}
	t.Fatal("Context Pack fixture has no calling function")
	return contextPackFixture{}
}

func writeContextPackFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func contextPackCompilerBuild() buildidentity.Identity {
	var result buildidentity.Identity
	for index := range result {
		result[index] = byte(0xa0 + index)
	}
	return result
}

func contextPackQueryBudget(depth int) atlasmap.QueryBudget {
	return atlasmap.QueryBudget{MaxRecords: 256, MaxRelations: 128, MaxDepth: depth, MaxSelectionBytes: 4 << 20}
}
