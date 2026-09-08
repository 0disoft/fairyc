package evidencebundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/contextpack"
)

func TestAggregateBundleKeepsChildrenAuthorityAndLimits(t *testing.T) {
	f := newEvidenceFixtureMode(t, true, true)
	r := atlasmap.RelationRequest{Direction: atlasmap.DirectionOutgoing, Relations: []atlasmap.RelationKind{atlasmap.RelationCalls}, Budget: atlasmap.QueryBudget{MaxRecords: 64, MaxRelations: 64, MaxSelectionBytes: 1 << 20}}
	b := attributedtext.Budget{MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20}
	l := Limits{FactsBytes: 2 << 20, TextDocumentBytes: 2 << 20, BundleBytes: 8 << 20}
	build := func(symbol string, limits Limits) ([]byte, error) {
		return EncodeShowV2(f.workspace, f.index, f.snapshot, symbol, r, b, limits)
	}
	data, err := build(f.main.Visible, l)
	if err != nil {
		t.Fatal(err)
	}
	full, err := build(f.main.Full, l)
	if err != nil || !bytes.Equal(full, data) {
		t.Fatal("full ID changed output")
	}
	var doc bundleEnvelope
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.BundleVersion != BundleVersionV2 || doc.Encoding != BundleEncodingV2 || doc.ChecksumAlgorithm != ChecksumAlgorithmV2 || doc.Bundle.Authority.Bundle != AuthorityBundleNone || doc.Bundle.Authority.Facts != AuthorityCompilerFact || doc.Bundle.Authority.Source != attributedtext.AuthorityNone || doc.Bundle.Authority.Documentation != attributedtext.AuthorityNone {
		t.Fatal("version or authority changed")
	}
	facts, err := contextpack.EncodeShowV3(f.index, f.main.Visible, r, l.FactsBytes)
	if err != nil || !bytes.Equal(facts, doc.Bundle.Components.Facts) {
		t.Fatal("facts child changed")
	}
	if !bytes.Contains(facts, []byte(`"source_body":"cleanup"`)) || !bytes.Contains(facts, []byte(`"aggregate":`)) || bytes.Contains(facts, []byte(evidencePromptLikeText)) {
		t.Fatal("facts lost aggregate ownership or acquired text")
	}
	for _, kind := range []attributedtext.Kind{attributedtext.KindSourceExcerpt, attributedtext.KindDeclarationDocumentation} {
		request := attributedtext.Request{SymbolID: f.main.Visible, Kinds: []attributedtext.Kind{kind}, Origins: []attributedtext.Origin{attributedtext.OriginWorkspaceAuthored}, Budget: b}
		var child, want []byte
		if kind == attributedtext.KindSourceExcerpt {
			child = doc.Bundle.Components.Source
			want, err = attributedtext.EncodeWorkspaceSource(f.workspace, f.index, request, l.TextDocumentBytes)
		} else {
			child = doc.Bundle.Components.Documentation
			want, err = attributedtext.EncodeWorkspaceDocumentation(f.workspace, f.index, f.snapshot, request, l.TextDocumentBytes)
		}
		if err != nil || !bytes.Equal(child, want) {
			t.Fatal("attributed child changed")
		}
	}
	stored, err := ParseChecksumV2(doc.Checksum)
	if err != nil || stored.String() != doc.Checksum {
		t.Fatal("checksum parse")
	}
	payload, err := json.Marshal(doc.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	digest.Write([]byte("fairyc.atlas-evidence-checksum\x00\x02"))
	for _, value := range [][]byte{[]byte("fairyc.atlas-evidence/v2"), []byte("fairyc.atlas-evidence-json/v2"), payload} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		digest.Write(length[:])
		digest.Write(value)
	}
	if !bytes.Equal(stored[:], digest.Sum(nil)) {
		t.Fatal("independent checksum differs")
	}
	if _, err := ParseChecksum(doc.Checksum); err == nil {
		t.Fatal("v0 accepted v2 checksum")
	}
	if _, err := ParseChecksumV1(doc.Checksum); err == nil {
		t.Fatal("v1 accepted v2 checksum")
	}
	for _, version := range []byte{0, 1} {
		if data, err := encodeShow(f.workspace, f.index, f.snapshot, f.main.Visible, r, b, l, version); err == nil || data != nil {
			t.Fatal("legacy bundle admitted v3")
		}
		old := newEvidenceFixtureVersion(t, version == 1)
		if data, err := EncodeShowV2(old.workspace, old.index, old.snapshot, old.main.Visible, r, b, l); err == nil || data != nil {
			t.Fatal("v2 admitted legacy map")
		}
	}
	other := newEvidenceFixture(t)
	if data, err := EncodeShowV2(f.workspace, f.index, other.snapshot, f.main.Visible, r, b, l); err == nil || data != nil {
		t.Fatal("mismatched snapshot accepted")
	}
	for _, item := range []struct {
		limits Limits
		want   error
	}{{Limits{1, l.TextDocumentBytes, l.BundleBytes}, contextpack.ErrPackTooLarge}, {Limits{l.FactsBytes, 1, l.BundleBytes}, attributedtext.ErrDocumentTooLarge}, {Limits{l.FactsBytes, l.TextDocumentBytes, 1}, ErrBundleTooLarge}} {
		if data, err := build(f.main.Visible, item.limits); !errors.Is(err, item.want) || data != nil {
			t.Fatalf("limit: %v", err)
		}
	}
	b.MaxFragmentBytes, b.MaxTextBytes = 1, 1
	data, err = build(f.main.Visible, l)
	if err != nil || bytes.Count(data, []byte(`"code":"fragment-byte-limit"`)) != 2 {
		t.Fatal("child omissions lost")
	}
}
