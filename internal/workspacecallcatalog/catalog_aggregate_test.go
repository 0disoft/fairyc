package workspacecallcatalog

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

func TestCatalogV2PreservesCleanupOwnershipAndChecksum(t *testing.T) {
	root, index, source, target, full := workspaceCallCatalogFixtureMode(t, true, true)
	for _, old := range []func(*atlasmap.WorkspaceCallIndex, string, string, atlasmap.QueryDirection, uint64) ([]byte, error){Encode, EncodeV1} {
		if data, err := old(index, source.PackageInstance, source.Symbol, atlasmap.DirectionOutgoing, 2<<20); err == nil || data != nil {
			t.Fatal("legacy encoder accepted v3 facts")
		}
	}
	for _, direction := range []atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth} {
		subject, selector := source, source.Symbol
		if direction == atlasmap.DirectionIncoming {
			subject, selector = target, full
		}
		data, err := EncodeV2(index, subject.PackageInstance, selector, direction, 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := EncodeV2(index, subject.PackageInstance, selector, direction, uint64(len(data)))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatalf("canonical exact limit: %v", err)
		}
		if truncated, err := EncodeV2(index, subject.PackageInstance, selector, direction, uint64(len(data)-1)); !errors.Is(err, ErrCatalogTooLarge) || truncated != nil {
			t.Fatal("final byte limit leaked partial catalog")
		}
		if bytes.Contains(data, []byte(root)) || bytes.Contains(data, []byte("let T observed")) || bytes.HasSuffix(data, []byte("\n")) {
			t.Fatal("source/path/newline leaked")
		}
		var decoded envelope
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.CatalogVersion != CatalogVersionV2 || decoded.Encoding != CatalogEncodingV2 || decoded.ChecksumAlgorithm != ChecksumAlgorithmV2 || decoded.Catalog.Context.LogicalSchema != atlasmap.LogicalSchemaV3 {
			t.Fatal("version identity lost")
		}
		catalog := decoded.Catalog
		if len(catalog.Calls) != 2 || catalog.Usage.SelectedCalls != 2 || !catalog.Complete || catalog.Subject.Specialization != "" {
			t.Fatal("selection lost cleanup calls")
		}
		ids := map[string]bool{}
		for _, call := range catalog.Calls {
			if call.SourceBody != "cleanup" || call.Source.Specialization == "" || call.Target.Specialization == "" || call.Source.Symbol != source.Symbol || call.Target.Symbol != target.Symbol {
				t.Fatalf("ownership lost: %+v", call)
			}
			ids[call.Source.Specialization] = true
		}
		if len(ids) != 2 {
			t.Fatal("shared-span implementations merged")
		}
		unknown := false
		for _, fact := range catalog.Context.Completeness {
			if fact.Relation == "generic-template-bodies" && fact.Status == semanticsnapshot.CompletenessUnavailable {
				unknown = true
			}
		}
		if !unknown {
			t.Fatal("template uncertainty lost")
		}
		checksum, err := ParseChecksumV2(decoded.Checksum)
		if err != nil || checksum.String() != decoded.Checksum {
			t.Fatalf("checksum parse: %v", err)
		}
		payload, err := json.Marshal(catalog)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		digest.Write([]byte("fairyc.workspace-call-catalog-checksum\x00\x02"))
		for _, value := range [][]byte{[]byte("fairyc.workspace-call-catalog/v2"), []byte("fairyc.workspace-call-catalog-json/v2"), payload} {
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(value)))
			digest.Write(length[:])
			digest.Write(value)
		}
		if !bytes.Equal(checksum[:], digest.Sum(nil)) {
			t.Fatal("independent checksum differs")
		}
		if _, err := ParseChecksum(decoded.Checksum); err == nil {
			t.Fatal("v0 accepted v2 checksum")
		}
		if _, err := ParseChecksumV1(decoded.Checksum); err == nil {
			t.Fatal("v1 accepted v2 checksum")
		}
		catalog.Calls[0].SourceBody = ""
		changed, err := computeVersionedChecksum(catalog, CatalogVersionV2, CatalogEncodingV2, 2)
		if err != nil || bytes.Equal(changed[:], checksum[:]) {
			t.Fatal("source body not checksum bound")
		}
	}
	data, err := EncodeV2(index, source.PackageInstance, source.Symbol, atlasmap.DirectionIncoming, 2<<20)
	if err != nil || !bytes.Contains(data, []byte(`"calls":[]`)) {
		t.Fatalf("empty selection: %v", err)
	}
}

func TestCatalogV2VersionIsolationAndSourceBodyOrdering(t *testing.T) {
	for _, generic := range []bool{false, true} {
		_, index, source, _, _ := workspaceCallCatalogFixtureVersion(t, generic)
		if _, err := EncodeV2(index, source.PackageInstance, source.Symbol, atlasmap.DirectionBoth, 2<<20); err == nil {
			t.Fatal("v2 accepted legacy maps")
		}
		encoder := Encode
		if generic {
			encoder = EncodeV1
		}
		data, err := encoder(index, source.PackageInstance, source.Symbol, atlasmap.DirectionBoth, 2<<20)
		if err != nil || bytes.Contains(data, []byte(`"source_body"`)) {
			t.Fatal("legacy output gained source body")
		}
	}
	for _, invalid := range []string{ChecksumPrefix + strings.Repeat("0", 64), ChecksumPrefixV1 + strings.Repeat("0", 64), ChecksumPrefixV2 + strings.Repeat("A", 64), ChecksumPrefixV2 + "00"} {
		if _, err := ParseChecksumV2(invalid); err == nil {
			t.Fatal("invalid v2 checksum accepted")
		}
	}
	first := atlasmap.JoinedCall{}
	second := first
	second.SourceBody = "cleanup"
	if compareJoinedCall(first, second) >= 0 || compareJoinedCall(second, first) <= 0 {
		t.Fatal("ordering ignores source body")
	}
}
