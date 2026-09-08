package contextpack

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestContextPackV3PreservesAggregateFieldsAndCleanup(t *testing.T) {
	fixture := loadContextPackFixtureMode(t, true, true)
	request := atlasmap.RelationRequest{Direction: atlasmap.DirectionBoth, Relations: atlasmap.AllRelationKinds(), Budget: contextPackQueryBudget(3)}
	for _, kind := range []QueryKind{QueryOverview, QueryShow, QueryAround} {
		build := func(version byte, limit uint64) ([]byte, error) {
			switch kind {
			case QueryOverview:
				return encodeOverview(fixture.index, request.Budget, limit, version)
			case QueryShow:
				return encodeShow(fixture.index, fixture.symbol, request, limit, version)
			default:
				return encodeAround(fixture.index, fixture.symbol, request, limit, version)
			}
		}
		data, err := build(3, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		again, err := build(3, uint64(len(data)))
		if err != nil || !bytes.Equal(data, again) {
			t.Fatal("canonical exact bound")
		}
		if data, err := build(3, uint64(len(data)-1)); !errors.Is(err, ErrPackTooLarge) || data != nil {
			t.Fatal("partial output at byte limit")
		}
		for _, old := range []byte{1, 2} {
			if data, err := build(old, 16<<20); err == nil || data != nil {
				t.Fatal("legacy pack accepted v3")
			}
		}
		var pack packEnvelope
		if err := json.Unmarshal(data, &pack); err != nil {
			t.Fatal(err)
		}
		if pack.PackVersion != PackVersionV3 || pack.Encoding != PackEncodingV3 || pack.ChecksumAlgorithm != ChecksumAlgorithmV3 || pack.Pack.MapVersion != atlasmap.LogicalSchemaV3 {
			t.Fatal("version identity lost")
		}
		if len(pack.Pack.Documentation) != 0 || len(pack.Pack.SourceExcerpts) != 0 || bytes.Contains(data, []byte(fixture.root)) {
			t.Fatal("free text channel populated")
		}
		if kind == QueryShow {
			node := pack.Pack.Facts.Show.Subject
			if node.Function != nil || node.Declaration.Key.Kind != "resource" || len(node.Specializations) != 2 {
				t.Fatal("resource template collapsed into function")
			}
			for _, value := range node.Specializations {
				if value.Function != nil || value.Aggregate == nil || value.Cleanup == nil || value.Aggregate.Fields[0].Type.Arguments[0].Name != value.TypeArguments[0].Name {
					t.Fatal("concrete fields or cleanup missing")
				}
			}
			if len(pack.Pack.Relations) != 4 {
				t.Fatalf("cleanup call/borrow count: %d", len(pack.Pack.Relations))
			}
			for _, relation := range pack.Pack.Relations {
				if relation.Source != fixture.symbol || relation.SourceBody != "cleanup" || relation.SourceSpecialization == "" {
					t.Fatal("cleanup ownership lost")
				}
			}
		}
		if kind == QueryAround && len(pack.Pack.Facts.Around.Nodes) < 2 {
			t.Fatal("cleanup helper absent from neighborhood")
		}
		checksum, err := ParseChecksumV3(pack.Checksum)
		if err != nil || checksum.String() != pack.Checksum {
			t.Fatalf("checksum: %v", err)
		}
		payload, err := json.Marshal(pack.Pack)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.New()
		digest.Write([]byte("fairyc.context-pack-checksum\x00\x03"))
		for _, value := range [][]byte{[]byte("fairyc.context-pack/v3"), []byte("fairyc.context-pack-json/v3"), payload} {
			var length [8]byte
			binary.BigEndian.PutUint64(length[:], uint64(len(value)))
			digest.Write(length[:])
			digest.Write(value)
		}
		if !bytes.Equal(checksum[:], digest.Sum(nil)) {
			t.Fatal("independent checksum differs")
		}
		if _, err := ParseChecksum(pack.Checksum); err == nil {
			t.Fatal("v1 parsed v3 checksum")
		}
		if _, err := ParseChecksumV2(pack.Checksum); err == nil {
			t.Fatal("v2 parsed v3 checksum")
		}
	}
	for _, generic := range []bool{false, true} {
		old := loadContextPackFixtureVersion(t, generic)
		if _, err := EncodeShowV3(old.index, old.symbol, request, 16<<20); err == nil {
			t.Fatal("v3 accepted legacy query")
		}
	}
	request.Budget.MaxSelectionBytes = 1
	if data, err := EncodeShowV3(fixture.index, fixture.symbol, request, 16<<20); !errors.Is(err, atlasmap.ErrQueryBudgetTooSmall) || data != nil {
		t.Fatal("subject field budget bypass")
	}
}
