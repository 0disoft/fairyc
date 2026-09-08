package symbolcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestAggregateCatalogListsOriginalKindsAndChecksum(t *testing.T) {
	root, index := catalogFixtureMode(t, true, true)
	request := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 1, MaxSelectionBytes: 1 << 20}}
	for _, old := range []func(*atlasmap.QueryIndex, atlasmap.SymbolCatalogRequest, uint64) ([]byte, error){Encode, EncodeV2} {
		if data, err := old(index, request, 2<<20); err == nil || data != nil {
			t.Fatal("legacy encoder accepted v3 map")
		}
	}
	for name, kind := range map[string]semanticsnapshot.DeclarationKind{"Cell": semanticsnapshot.DeclarationStruct, "Token": semanticsnapshot.DeclarationResource, "Unused": semanticsnapshot.DeclarationStruct, "UnusedToken": semanticsnapshot.DeclarationResource} {
		request.Name, request.Kinds = name, []semanticsnapshot.DeclarationKind{kind}
		data, err := EncodeV3(index, request, 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := EncodeV3(index, request, uint64(len(data)))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatal("canonical exact bound")
		}
		if data, err := EncodeV3(index, request, uint64(len(data)-1)); !errors.Is(err, ErrCatalogTooLarge) || data != nil {
			t.Fatal("partial bytes at output limit")
		}
		if bytes.Contains(data, []byte(root)) || bytes.Contains(data, []byte(".payload =")) || bytes.HasSuffix(data, []byte("\n")) {
			t.Fatal("path/source/newline leaked")
		}
		var decoded envelope
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.CatalogVersion != CatalogVersionV3 || decoded.Encoding != CatalogEncodingV3 || decoded.ChecksumAlgorithm != ChecksumAlgorithmV3 || decoded.Catalog.Context.LogicalSchema != atlasmap.LogicalSchemaV3 || decoded.Catalog.Context.GenericInterfaceHash == "" {
			t.Fatal("v3 identity missing")
		}
		if decoded.Catalog.Total != 1 || len(decoded.Catalog.Symbols) != 1 || decoded.Catalog.Symbols[0].Name != name || decoded.Catalog.Symbols[0].Kind != kind || !decoded.Catalog.Complete {
			t.Fatal("original aggregate duplicated or missing")
		}
		stored, err := ParseChecksumV3(decoded.Checksum)
		if err != nil || stored.String() != decoded.Checksum {
			t.Fatalf("checksum: %v", err)
		}
		payload, err := json.Marshal(decoded.Catalog)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		digest.Write([]byte("fairyc.symbol-catalog-checksum\x00\x02"))
		for _, value := range [][]byte{[]byte("fairyc.symbol-catalog/v3"), []byte("fairyc.symbol-catalog-json/v3"), payload} {
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(value)))
			digest.Write(length[:])
			digest.Write(value)
		}
		if !bytes.Equal(stored[:], digest.Sum(nil)) {
			t.Fatal("independent checksum differs")
		}
		if _, err := ParseChecksum(decoded.Checksum); err == nil {
			t.Fatal("v1 accepted v3 checksum")
		}
		if _, err := ParseChecksumV2(decoded.Checksum); err == nil {
			t.Fatal("v2 accepted v3 checksum")
		}
	}
	request.Name, request.Kinds = "", atlasmap.AllDeclarationKinds()
	names := map[string]bool{}
	for {
		data, err := EncodeV3(index, request, 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		var page envelope
		if err := json.Unmarshal(data, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Catalog.Symbols) != 1 || page.Catalog.Total != 6 || names[page.Catalog.Symbols[0].Name] {
			t.Fatal("bad aggregate pagination")
		}
		names[page.Catalog.Symbols[0].Name] = true
		if page.Catalog.Complete {
			break
		}
		if len(names) >= 6 || page.Catalog.NextAfter == "" {
			t.Fatal("cursor did not advance")
		}
		request.After = page.Catalog.NextAfter
	}
	if len(names) != 6 {
		t.Fatal("missing source declarations")
	}
	request.After, request.Name = "", "missing"
	data, err := EncodeV3(index, request, 2<<20)
	if err != nil || !bytes.Contains(data, []byte(`"symbols":[]`)) {
		t.Fatal("empty normalization")
	}
}

func TestAggregateCatalogRejectsOtherVersions(t *testing.T) {
	request := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 8, MaxSelectionBytes: 1 << 20}}
	for _, generic := range []bool{false, true} {
		_, index := catalogFixtureVersion(t, generic)
		if data, err := EncodeV3(index, request, 2<<20); err == nil || data != nil {
			t.Fatal("v3 accepted legacy index")
		}
	}
	for _, invalid := range []string{ChecksumPrefix + strings.Repeat("0", 64), ChecksumPrefixV2 + strings.Repeat("0", 64), ChecksumPrefixV3 + strings.Repeat("A", 64), ChecksumPrefixV3 + "00"} {
		if _, err := ParseChecksumV3(invalid); err == nil {
			t.Fatal("invalid checksum accepted")
		}
	}
}
