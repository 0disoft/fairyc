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
)

func TestGenericCatalogPreservesOriginalIdentitiesAndBounds(t *testing.T) {
	_, index := catalogFixtureVersion(t, true)
	request := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 1, MaxSelectionBytes: 1 << 20}}
	if data, err := Encode(index, request, 2<<20); err == nil || data != nil {
		t.Fatal("v1 accepted v2 map")
	}
	for _, name := range []string{"id", "unused"} {
		request.Name = name
		data, err := EncodeV2(index, request, 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := EncodeV2(index, request, uint64(len(data)))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatal("noncanonical output or exact byte bound")
		}
		if data, err := EncodeV2(index, request, uint64(len(data)-1)); !errors.Is(err, ErrCatalogTooLarge) || data != nil {
			t.Fatal("byte bound failed")
		}
		var decoded envelope
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.CatalogVersion != CatalogVersionV2 || decoded.Encoding != CatalogEncodingV2 || decoded.ChecksumAlgorithm != ChecksumAlgorithmV2 || decoded.Catalog.Context.LogicalSchema != atlasmap.LogicalSchemaV2 || decoded.Catalog.Context.GenericInterfaceHash == "" {
			t.Fatal("version or interface identity missing")
		}
		if decoded.Catalog.Total != 1 || len(decoded.Catalog.Symbols) != 1 || decoded.Catalog.Symbols[0].Name != name || !decoded.Catalog.Complete {
			t.Fatal("original declaration duplicated or missing")
		}
		stored, err := ParseChecksumV2(decoded.Checksum)
		if err != nil || stored.String() != decoded.Checksum {
			t.Fatalf("checksum: %v", err)
		}
		payload, err := json.Marshal(decoded.Catalog)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		digest.Write([]byte("fairyc.symbol-catalog-checksum\x00\x01"))
		for _, value := range [][]byte{[]byte("fairyc.symbol-catalog/v2"), []byte("fairyc.symbol-catalog-json/v2"), payload} {
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(value)))
			digest.Write(length[:])
			digest.Write(value)
		}
		if !bytes.Equal(stored[:], digest.Sum(nil)) {
			t.Fatal("independent checksum disagrees")
		}
		if _, err := ParseChecksum(decoded.Checksum); err == nil {
			t.Fatal("v1 accepted v2 checksum")
		}
	}
	request.Name = ""
	names := map[string]bool{}
	for {
		data, err := EncodeV2(index, request, 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		var page envelope
		if err := json.Unmarshal(data, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Catalog.Symbols) != 1 || page.Catalog.Total != 3 || names[page.Catalog.Symbols[0].Name] {
			t.Fatal("bad generic pagination")
		}
		names[page.Catalog.Symbols[0].Name] = true
		if page.Catalog.Complete {
			break
		}
		if len(names) >= 3 || page.Catalog.NextAfter == "" {
			t.Fatal("cursor did not progress")
		}
		request.After = page.Catalog.NextAfter
	}
	if len(names) != 3 {
		t.Fatal("generic declarations missing")
	}
	_, legacy := catalogFixture(t)
	if _, err := EncodeV2(legacy, atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: request.Budget}, 2<<20); err == nil {
		t.Fatal("v2 accepted v1 map")
	}
	for _, invalid := range []string{ChecksumPrefix + strings.Repeat("0", 64), ChecksumPrefixV2 + strings.Repeat("A", 64), ChecksumPrefixV2 + "00"} {
		if _, err := ParseChecksumV2(invalid); err == nil {
			t.Fatal("invalid checksum accepted")
		}
	}
}
