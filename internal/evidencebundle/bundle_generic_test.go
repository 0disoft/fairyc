package evidencebundle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/contextpack"
)

func TestGenericBundlePreservesExactChildrenAndVersionBoundaries(t *testing.T) {
	f := newEvidenceFixtureVersion(t, true)
	r := atlasmap.RelationRequest{Direction: atlasmap.DirectionOutgoing, Relations: []atlasmap.RelationKind{atlasmap.RelationCalls}, Budget: atlasmap.QueryBudget{MaxRecords: 64, MaxRelations: 64, MaxSelectionBytes: 1 << 20}}
	b := attributedtext.Budget{MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20}
	l := Limits{FactsBytes: 2 << 20, TextDocumentBytes: 2 << 20, BundleBytes: 8 << 20}
	encode := func(symbol string, limits Limits) ([]byte, error) {
		return EncodeShowV1(f.workspace, f.index, f.snapshot, symbol, r, b, limits)
	}
	data, err := encode(f.main.Visible, l)
	if err != nil {
		t.Fatal(err)
	}
	full, err := encode(f.main.Full, l)
	if err != nil || !bytes.Equal(data, full) {
		t.Fatal("full ID changed output")
	}
	var doc bundleEnvelope
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.BundleVersion != BundleVersionV1 || doc.Encoding != BundleEncodingV1 || doc.ChecksumAlgorithm != ChecksumAlgorithmV1 || doc.Bundle.Authority.Facts != AuthorityCompilerFact || doc.Bundle.Authority.Source != attributedtext.AuthorityNone || doc.Bundle.Authority.Documentation != attributedtext.AuthorityNone || doc.Bundle.Authority.Bundle != AuthorityBundleNone {
		t.Fatal("version or authority lost")
	}
	facts, err := contextpack.EncodeShowV2(f.index, f.main.Visible, r, l.FactsBytes)
	if err != nil || !bytes.Equal(facts, doc.Bundle.Components.Facts) {
		t.Fatal("facts child changed")
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
			t.Fatal("text child changed")
		}
	}
	var pack struct {
		Pack struct {
			Facts struct {
				Show struct {
					Subject atlasmap.QueryNode `json:"subject"`
				} `json:"show"`
			} `json:"facts"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(facts, &pack); err != nil {
		t.Fatal(err)
	}
	if pack.Pack.Facts.Show.Subject.Function != nil || len(pack.Pack.Facts.Show.Subject.Specializations) != 2 {
		t.Fatal("template uncertainty or implementations lost")
	}
	if bytes.Contains(facts, []byte(evidencePromptLikeText)) || bytes.Contains(data, []byte(f.root)) || bytes.HasSuffix(data, []byte("\n")) {
		t.Fatal("text/host path leaked or noncanonical output")
	}
	checksum, err := ParseChecksumV1(doc.Checksum)
	if err != nil || checksum.String() != doc.Checksum {
		t.Fatal("checksum invalid")
	}
	payload, _ := json.Marshal(doc.Bundle)
	digest := sha256.New()
	digest.Write([]byte("fairyc.atlas-evidence-checksum"))
	digest.Write([]byte{0, 1})
	for _, value := range [][]byte{[]byte("fairyc.atlas-evidence/v1"), []byte("fairyc.atlas-evidence-json/v1"), payload} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		digest.Write(length[:])
		digest.Write(value)
	}
	if !bytes.Equal(checksum[:], digest.Sum(nil)) {
		t.Fatal("independent checksum differs")
	}
	if _, err := ParseChecksum(doc.Checksum); err == nil {
		t.Fatal("v0 accepted v1 checksum")
	}
	for _, bad := range []string{ChecksumPrefix + strings.Repeat("0", 64), ChecksumPrefixV1 + strings.Repeat("A", 64), ChecksumPrefixV1 + "00"} {
		if _, err := ParseChecksumV1(bad); err == nil {
			t.Fatal("invalid checksum accepted")
		}
	}
	if got, err := EncodeShow(f.workspace, f.index, f.snapshot, f.main.Visible, r, b, l); err == nil || got != nil {
		t.Fatal("v0 accepted v2 query")
	}
	old := newEvidenceFixture(t)
	if got, err := EncodeShowV1(old.workspace, old.index, old.snapshot, old.main.Visible, r, b, l); err == nil || got != nil {
		t.Fatal("v1 accepted v1 query")
	}
	other := newEvidenceFixture(t)
	if got, err := EncodeShowV1(f.workspace, f.index, other.snapshot, f.main.Visible, r, b, l); err == nil || got != nil {
		t.Fatal("mismatched snapshot accepted")
	}
	writeEvidenceFile(t, filepath.Join(f.root, "src", "main.fy"), "module changed;")
	retained, err := encode(f.main.Visible, l)
	if err != nil || !bytes.Equal(data, retained) {
		t.Fatal("disk reread changed bundle")
	}
	for _, item := range []struct {
		limits Limits
		want   error
	}{{Limits{1, l.TextDocumentBytes, l.BundleBytes}, contextpack.ErrPackTooLarge}, {Limits{l.FactsBytes, 1, l.BundleBytes}, attributedtext.ErrDocumentTooLarge}, {Limits{l.FactsBytes, l.TextDocumentBytes, 1}, ErrBundleTooLarge}} {
		if got, err := encode(f.main.Visible, item.limits); !errors.Is(err, item.want) || got != nil {
			t.Fatalf("limit: %v", err)
		}
	}
	b.MaxFragmentBytes = 1
	b.MaxTextBytes = 1
	omitted, err := encode(f.main.Visible, l)
	if err != nil || bytes.Count(omitted, []byte(`"code":"fragment-byte-limit"`)) != 2 {
		t.Fatal("omissions lost")
	}
	b.MaxFragmentBytes = 1 << 20
	b.MaxTextBytes = 1 << 20
	unused, err := encode(f.helper.Visible, l)
	if err != nil || !bytes.Contains(unused, []byte(`"code":"kind-unavailable"`)) {
		t.Fatal("unused template documentation fabricated")
	}
}
