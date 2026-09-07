package contextpack

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

func TestContextPackV2PreservesGenericFactsAndVersionBoundaries(t *testing.T) {
	fixture := loadContextPackFixtureVersion(t, true)
	legacy := loadContextPackFixture(t)
	request := atlasmap.RelationRequest{Direction: atlasmap.DirectionBoth, Relations: atlasmap.AllRelationKinds(), Budget: contextPackQueryBudget(3)}
	for _, kind := range []QueryKind{QueryOverview, QueryShow, QueryAround} {
		encode := func(value contextPackFixture, v2 bool, limit uint64) ([]byte, error) {
			switch kind {
			case QueryOverview:
				if v2 {
					return EncodeOverviewV2(value.index, request.Budget, limit)
				}
				return EncodeOverview(value.index, request.Budget, limit)
			case QueryShow:
				if v2 {
					return EncodeShowV2(value.index, value.symbol, request, limit)
				}
				return EncodeShow(value.index, value.symbol, request, limit)
			default:
				if v2 {
					return EncodeAroundV2(value.index, value.symbol, request, limit)
				}
				return EncodeAround(value.index, value.symbol, request, limit)
			}
		}
		data, err := encode(fixture, true, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := encode(fixture, true, uint64(len(data)))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatal("noncanonical pack or exact bound failed")
		}
		if output, err := encode(fixture, true, uint64(len(data)-1)); !errors.Is(err, ErrPackTooLarge) || output != nil {
			t.Fatal("pack output bound did not fail atomically")
		}
		if output, err := encode(fixture, false, 16<<20); err == nil || output != nil {
			t.Fatal("v1 accepted generic map")
		}
		if output, err := encode(legacy, true, 16<<20); err == nil || output != nil {
			t.Fatal("v2 accepted v1 map")
		}
		if bytes.Contains(data, []byte(fixture.root)) || bytes.Contains(data, []byte("return leaf")) || bytes.HasSuffix(data, []byte("\n")) {
			t.Fatal("pack leaked host path or text")
		}
		var pack packEnvelope
		if err := json.Unmarshal(data, &pack); err != nil {
			t.Fatal(err)
		}
		if pack.PackVersion != PackVersionV2 || pack.Encoding != PackEncodingV2 || pack.ChecksumAlgorithm != ChecksumAlgorithmV2 || pack.Pack.MapVersion != atlasmap.LogicalSchemaV2 || pack.Pack.Compatibility.GenericInterfaceHash == "" {
			t.Fatal("generic compatibility identity lost")
		}
		if len(pack.Pack.Documentation) != 0 || len(pack.Pack.SourceExcerpts) != 0 || len(pack.Pack.Warnings) != 0 {
			t.Fatal("text channels became populated")
		}
		unknown := false
		for _, record := range pack.Pack.Completeness {
			if record.Relation == "generic-template-bodies" && record.Status == semanticsnapshot.CompletenessUnavailable {
				unknown = true
			}
		}
		if !unknown {
			t.Fatal("template uncertainty lost")
		}
		if kind == QueryShow {
			node := pack.Pack.Facts.Show.Subject
			if node.Function != nil || len(node.Specializations) != 2 || len(node.TypeParameters) != 1 || node.TypeParameters[0] != "T" {
				t.Fatal("implementation facts collapsed into template")
			}
			for _, relation := range pack.Pack.Relations {
				if relation.Source == fixture.symbol && (relation.SourceSpecialization == "" || relation.Target == nil || relation.Target.Specialization == "") {
					t.Fatal("implementation endpoint lost")
				}
			}
		}
		if kind == QueryAround && len(pack.Pack.Facts.Around.Nodes) < 2 {
			t.Fatal("declaration neighborhood missing")
		}
		stored, err := ParseChecksumV2(pack.Checksum)
		if err != nil || stored.String() != pack.Checksum {
			t.Fatalf("checksum: %v", err)
		}
		payload, err := json.Marshal(pack.Pack)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		digest.Write([]byte("fairyc.context-pack-checksum\x00\x02"))
		for _, value := range [][]byte{[]byte("fairyc.context-pack/v2"), []byte("fairyc.context-pack-json/v2"), payload} {
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(value)))
			digest.Write(length[:])
			digest.Write(value)
		}
		if !bytes.Equal(stored[:], digest.Sum(nil)) {
			t.Fatal("independent checksum disagrees")
		}
		if _, err := ParseChecksum(pack.Checksum); err == nil {
			t.Fatal("v1 accepted v2 checksum")
		}
	}
	for _, invalid := range []string{ChecksumPrefix + strings.Repeat("0", 64), ChecksumPrefixV2 + strings.Repeat("A", 64), ChecksumPrefixV2 + "00"} {
		if _, err := ParseChecksumV2(invalid); err == nil {
			t.Fatal("invalid checksum accepted")
		}
	}
}
