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
)

func TestCatalogV1PreservesGenericEndpointsAndVersionedChecksum(t *testing.T) {
	root, index, source, target, targetFull := workspaceCallCatalogFixtureVersion(t, true)
	if data, err := Encode(index, source.PackageInstance, source.Symbol, atlasmap.DirectionOutgoing, 2<<20); err == nil || data != nil {
		t.Fatal("v0 silently accepted generic workspace facts")
	}
	var outgoing []byte
	for _, direction := range []atlasmap.QueryDirection{atlasmap.DirectionOutgoing, atlasmap.DirectionIncoming, atlasmap.DirectionBoth} {
		subject, selector := source, source.Symbol
		if direction == atlasmap.DirectionIncoming {
			subject, selector = target, targetFull
		}
		data, err := EncodeV1(index, subject.PackageInstance, selector, direction, 2<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := EncodeV1(index, subject.PackageInstance, selector, direction, uint64(len(data)))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatalf("noncanonical or exact bound failed: %v", err)
		}
		if data, err := EncodeV1(index, subject.PackageInstance, selector, direction, uint64(len(data)-1)); !errors.Is(err, ErrCatalogTooLarge) || data != nil {
			t.Fatal("final byte limit did not fail closed")
		}
		if bytes.Contains(data, []byte(root)) || bytes.Contains(data, []byte("return answer")) || bytes.HasSuffix(data, []byte("\n")) {
			t.Fatal("output contains text or newline")
		}
		var decoded envelope
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.CatalogVersion != CatalogVersionV1 || decoded.Encoding != CatalogEncodingV1 || decoded.ChecksumAlgorithm != ChecksumAlgorithmV1 || decoded.Catalog.Context.LogicalSchema != atlasmap.LogicalSchemaV2 {
			t.Fatal("wrong version identity")
		}
		want := 2
		if direction == atlasmap.DirectionBoth {
			want = 4
		}
		if len(decoded.Catalog.Calls) != want || decoded.Catalog.Usage.SelectedCalls != want || !decoded.Catalog.Complete || decoded.Catalog.Subject.Specialization != "" {
			t.Fatal("declaration selection lost calls")
		}
		ids := map[string]bool{}
		for _, call := range decoded.Catalog.Calls {
			if call.Source.PackageInstance == call.Target.PackageInstance {
				continue
			}
			if call.Source.Specialization == "" || call.Target.Specialization == "" {
				t.Fatal("concrete endpoint missing")
			}
			ids[call.Target.Specialization] = true
		}
		if len(ids) != 2 {
			t.Fatal("implementations collapsed")
		}
		checksum, err := ParseChecksumV1(decoded.Checksum)
		if err != nil || checksum.String() != decoded.Checksum {
			t.Fatalf("checksum: %v", err)
		}
		payload, err := json.Marshal(decoded.Catalog)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		digest.Write([]byte("fairyc.workspace-call-catalog-checksum\x00\x01"))
		for _, value := range [][]byte{[]byte("fairyc.workspace-call-catalog/v1"), []byte("fairyc.workspace-call-catalog-json/v1"), payload} {
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(value)))
			digest.Write(length[:])
			digest.Write(value)
		}
		if !bytes.Equal(checksum[:], digest.Sum(nil)) {
			t.Fatal("independent checksum disagrees")
		}
		if _, err := ParseChecksum(decoded.Checksum); err == nil {
			t.Fatal("v0 accepted v1 checksum")
		}
		if direction == atlasmap.DirectionOutgoing {
			outgoing = data
		}
	}
	if len(outgoing) == 0 {
		t.Fatal("no outgoing output")
	}
	_, legacy, legacySource, _, _ := workspaceCallCatalogFixture(t)
	if _, err := EncodeV1(legacy, legacySource.PackageInstance, legacySource.Symbol, atlasmap.DirectionBoth, 2<<20); err == nil {
		t.Fatal("v1 accepted v0 join")
	}
	for _, invalid := range []string{ChecksumPrefix + strings.Repeat("0", 64), ChecksumPrefixV1 + strings.Repeat("A", 64), ChecksumPrefixV1 + "00"} {
		if _, err := ParseChecksumV1(invalid); err == nil {
			t.Fatal("invalid checksum accepted")
		}
	}
}

func TestJoinedCallOrderingIncludesImplementationIdentity(t *testing.T) {
	first := atlasmap.JoinedCall{SourceSpecialization: "g0_a", TargetSpecialization: "g0_a"}
	second := first
	second.SourceSpecialization = "g0_b"
	if compareJoinedCall(first, second) >= 0 {
		t.Fatal("source implementation ignored")
	}
	second = first
	second.TargetSpecialization = "g0_b"
	if compareJoinedCall(first, second) >= 0 {
		t.Fatal("target implementation ignored")
	}
}
