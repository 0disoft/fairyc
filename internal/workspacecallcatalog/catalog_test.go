package workspacecallcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestEncodeIsCanonicalChecksummedBoundedAndTextFree(t *testing.T) {
	t.Parallel()
	root, index, source, target, targetFull := workspaceCallCatalogFixture(t)
	first, err := Encode(index, source.PackageInstance, source.Symbol, atlasmap.DirectionOutgoing, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encode(index, source.PackageInstance, source.Symbol, atlasmap.DirectionOutgoing, 2<<20)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("repeated workspace call catalog differs: %v", err)
	}
	if !json.Valid(first) || bytes.HasSuffix(first, []byte("\n")) || bytes.Contains(first, []byte(root)) ||
		bytes.Contains(first, []byte("answer::value")) {
		t.Fatalf("workspace call catalog leaked invalid, path, source, or newline data: %s", first)
	}
	var decoded envelope
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.CatalogVersion != CatalogVersion || decoded.Encoding != CatalogEncoding ||
		decoded.ChecksumAlgorithm != ChecksumAlgorithm || decoded.Catalog.Direction != atlasmap.DirectionOutgoing ||
		decoded.Catalog.Subject != (Symbol{PackageInstance: source.PackageInstance, Symbol: source.Symbol}) ||
		len(decoded.Catalog.Calls) != 1 || decoded.Catalog.Calls[0].Target != (Symbol{PackageInstance: target.PackageInstance, Symbol: target.Symbol}) ||
		decoded.Catalog.Usage != (Usage{Packages: 2, Calls: 1, CrossPackageCalls: 1, SelectedCalls: 1}) || !decoded.Catalog.Complete {
		t.Fatalf("workspace call catalog = %+v", decoded)
	}
	stored, err := ParseChecksum(decoded.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if computed := independentChecksum(t, decoded.Catalog); stored != computed {
		t.Fatalf("checksum = %s, computed = %s", stored, computed)
	}

	incoming, err := Encode(index, target.PackageInstance, targetFull, atlasmap.DirectionIncoming, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	var incomingEnvelope envelope
	if err := json.Unmarshal(incoming, &incomingEnvelope); err != nil {
		t.Fatal(err)
	}
	if incomingEnvelope.Catalog.Subject.Symbol != target.Symbol || len(incomingEnvelope.Catalog.Calls) != 1 ||
		incomingEnvelope.Catalog.Calls[0].Source.Symbol != source.Symbol {
		t.Fatalf("incoming workspace call catalog = %+v", incomingEnvelope.Catalog)
	}
	empty, err := Encode(index, target.PackageInstance, target.Symbol, atlasmap.DirectionOutgoing, 2<<20)
	if err != nil || !bytes.Contains(empty, []byte(`"calls":[]`)) || !bytes.Contains(empty, []byte(`"selected_calls":0`)) {
		t.Fatalf("empty workspace call catalog = %s, %v", empty, err)
	}
	if _, err := Encode(index, source.PackageInstance, source.Symbol, atlasmap.DirectionOutgoing, uint64(len(first)-1)); !errors.Is(err, ErrCatalogTooLarge) {
		t.Fatalf("small final bound error = %v", err)
	}
	if _, err := Encode(index, source.PackageInstance, source.Symbol, "sideways", 2<<20); err == nil {
		t.Fatal("invalid direction succeeded")
	}
	if _, err := Encode(index, source.PackageInstance, source.Symbol, atlasmap.DirectionOutgoing, 0); err == nil {
		t.Fatal("zero final bound succeeded")
	}
	if _, err := ParseChecksum(ChecksumPrefix + strings.Repeat("A", 64)); err == nil {
		t.Fatal("uppercase checksum succeeded")
	}
	if _, err := ParseChecksum("fywcall1-sha256-" + strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong-version checksum succeeded")
	}
}

func workspaceCallCatalogFixture(t *testing.T) (string, *atlasmap.WorkspaceCallIndex, atlasmap.WorkspaceSymbol, atlasmap.WorkspaceSymbol, string) {
	t.Helper()
	root := t.TempDir()
	writeCatalogFixture(t, filepath.Join(root, "fairy.toml"), `schema = "fairyc.manifest/v1"
[package]
uid = "test.workspace-call.app"
name = "app"
version = "1.0.0"
language = "fairyc-language/6"
source = "src"
[workspace]
members = ["libs/answer"]
[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
`)
	writeCatalogFixture(t, filepath.Join(root, "src", "main.fy"), "module main;\nimport answer as answer;\nfn i32 main() { return answer::value(); }\n")
	writeCatalogFixture(t, filepath.Join(root, "libs", "answer", "fairy.toml"), `schema = "fairyc.manifest/v1"
[package]
uid = "test.workspace-call.answer"
name = "answer"
version = "1.0.0"
language = "fairyc-language/6"
source = "src"
`)
	writeCatalogFixture(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), "module answer;\npub fn i32 value() { return 42; }\n")
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil || !checked.Verified {
		t.Fatalf("CheckPackages() = %t, %v", checked.Verified, err)
	}
	var compilerBuild buildidentity.Identity
	for position := range compilerBuild {
		compilerBuild[position] = byte(0x40 + position)
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, checked, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectations(workspace, checked, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	index, err := atlasmap.NewWorkspaceCallIndex(partitions, expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 4, MaxCalls: 8})
	if err != nil {
		t.Fatal(err)
	}
	calls := index.Calls()
	if len(calls) != 1 {
		t.Fatalf("joined calls = %+v", calls)
	}
	targetFull := ""
	for _, partition := range partitions {
		for _, symbol := range partition.Symbols() {
			if symbol.Visible == calls[0].Target.Symbol {
				targetFull = symbol.Full
			}
		}
	}
	if targetFull == "" {
		t.Fatal("joined target has no full SymbolId")
	}
	return root, index, calls[0].Source, calls[0].Target, targetFull
}

func writeCatalogFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func independentChecksum(t *testing.T, catalog Catalog) Checksum {
	t.Helper()
	payload, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("fairyc.workspace-call-catalog-checksum"))
	_, _ = digest.Write([]byte{0, 0})
	for _, value := range [][]byte{[]byte(CatalogVersion), []byte(CatalogEncoding), payload} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = digest.Write(length[:])
		_, _ = digest.Write(value)
	}
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result
}
