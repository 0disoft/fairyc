package symbolcatalog

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
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestEncodeIsCanonicalChecksummedBoundedAndTextFree(t *testing.T) {
	t.Parallel()
	root, index := catalogFixture(t)
	request := atlasmap.SymbolCatalogRequest{
		Kinds:  atlasmap.AllDeclarationKinds(),
		Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 2, MaxSelectionBytes: 1 << 20},
	}
	first, err := Encode(index, request, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encode(index, request, 2<<20)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("repeated catalog differs: %v", err)
	}
	if !json.Valid(first) || bytes.Contains(first, []byte(root)) || bytes.Contains(first, []byte("return leaf")) || bytes.HasSuffix(first, []byte("\n")) {
		t.Fatalf("catalog leaked invalid, path, source, or newline data: %s", first)
	}
	var decoded envelope
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.CatalogVersion != CatalogVersion || decoded.Encoding != CatalogEncoding ||
		decoded.ChecksumAlgorithm != ChecksumAlgorithm || len(decoded.Catalog.Symbols) != 2 || decoded.Catalog.Complete {
		t.Fatalf("catalog envelope = %+v", decoded)
	}
	stored, err := ParseChecksum(decoded.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if computed := independentChecksum(t, decoded.Catalog); stored != computed {
		t.Fatalf("checksum = %s, computed = %s", stored, computed)
	}
	if _, err := Encode(index, request, uint64(len(first)-1)); !errors.Is(err, ErrCatalogTooLarge) {
		t.Fatalf("small final bound error = %v", err)
	}
	if _, err := Encode(index, request, 0); err == nil {
		t.Fatal("zero final bound succeeded")
	}
	if _, err := ParseChecksum(ChecksumPrefix + strings.Repeat("A", 64)); err == nil {
		t.Fatal("uppercase checksum succeeded")
	}
	if _, err := ParseChecksum("fyscat0-sha256-" + strings.Repeat("0", 64)); err == nil {
		t.Fatal("v0 checksum succeeded under the v1 parser")
	}
	namedRequest := request
	namedRequest.Name = "leaf"
	named, err := Encode(index, namedRequest, 2<<20)
	if err != nil {
		t.Fatalf("exact-name catalog = %s, %v", named, err)
	}
	var decodedNamed envelope
	if err := json.Unmarshal(named, &decodedNamed); err != nil {
		t.Fatal(err)
	}
	if decodedNamed.Catalog.Name != "leaf" || decodedNamed.Catalog.Total != 1 ||
		len(decodedNamed.Catalog.Symbols) != 1 || decodedNamed.Catalog.Symbols[0].Name != "leaf" {
		t.Fatalf("exact-name catalog = %+v", decodedNamed.Catalog)
	}
	namedChecksum, err := ParseChecksum(decodedNamed.Checksum)
	if err != nil || namedChecksum != independentChecksum(t, decodedNamed.Catalog) {
		t.Fatalf("exact-name checksum = %s, %v", namedChecksum, err)
	}
	emptyRequest := request
	emptyRequest.Kinds = []semanticsnapshot.DeclarationKind{semanticsnapshot.DeclarationEnum}
	empty, err := Encode(index, emptyRequest, 2<<20)
	if err != nil || !bytes.Contains(empty, []byte(`"symbols":[]`)) || !bytes.Contains(empty, []byte(`"omissions":[]`)) {
		t.Fatalf("empty catalog arrays = %s, %v", empty, err)
	}
}

func catalogFixture(t *testing.T) (string, *atlasmap.QueryIndex) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeCatalogFile(t, filepath.Join(root, "fairy.toml"), `schema = "fairyc.manifest/v1"
[package]
uid = "test.symbol-catalog"
name = "symbol-catalog"
version = "1.0.0"
language = "fairyc-language/14"
source = "src"
`)
	writeCatalogFile(t, filepath.Join(root, "src", "main.fy"), `module main;
fn i32 leaf() { return 40; }
fn i32 middle() { return leaf() + 1; }
fn i32 main() { return middle() + 1; }
`)
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
		compilerBuild[position] = byte(0x70 + position)
	}
	_, partitions, err := atlasmap.PrepareWorkspace(workspace, checked, compilerBuild)
	if err != nil || len(partitions) != 1 {
		t.Fatalf("PrepareWorkspace() = %d, %v", len(partitions), err)
	}
	expectations, err := atlasmap.CurrentWorkspaceExpectations(workspace, checked, compilerBuild)
	if err != nil {
		t.Fatal(err)
	}
	partition := partitions[0]
	index, err := atlasmap.NewQueryIndex(partition, expectations[partition.Header().PackageInstance])
	if err != nil {
		t.Fatal(err)
	}
	return root, index
}

func writeCatalogFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func independentChecksum(t *testing.T, catalog atlasmap.SymbolCatalogResult) Checksum {
	t.Helper()
	payload, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("fairyc.symbol-catalog-checksum"))
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
