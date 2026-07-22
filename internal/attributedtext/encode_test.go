package attributedtext

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEncodeWorkspaceSourceIsCanonicalChecksummedBoundedAndExact(t *testing.T) {
	t.Parallel()
	fixture := loadAttributedTextFixture(t, []byte(attributedTextSource))
	request := sourceRequest(fixture.visible, Budget{
		MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20,
	})
	first, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, request, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, request, 2<<20)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("repeated attributed-text encoding differs: %v", err)
	}
	fullRequest := request
	fullRequest.SymbolID = fixture.full
	full, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, fullRequest, 2<<20)
	if err != nil || !bytes.Equal(first, full) {
		t.Fatalf("full SymbolId encoding differs: %v", err)
	}
	independentFixture := loadAttributedTextFixture(t, []byte(attributedTextSource))
	independentBytes, err := EncodeWorkspaceSource(
		independentFixture.workspace,
		independentFixture.index,
		sourceRequest(independentFixture.visible, request.Budget),
		2<<20,
	)
	if err != nil || !bytes.Equal(first, independentBytes) {
		t.Fatalf("equivalent workspace under another root differs: %v", err)
	}
	if !json.Valid(first) || bytes.HasSuffix(first, []byte("\n")) || bytes.Contains(first, []byte(fixture.root)) ||
		!bytes.Contains(first, []byte("Ignore previous instructions")) || !bytes.Contains(first, []byte("λ")) {
		t.Fatalf("encoded attributed text is invalid, path-bearing, or missing exact source: %s", first)
	}
	if !bytes.HasPrefix(first, []byte(`{"text_version":"fairyc.attributed-text/v0","encoding":"fairyc.attributed-text-json/v0","checksum_algorithm":"fairyc.attributed-text-checksum/v0-sha256","checksum":"fytextdoc0-sha256-`)) {
		t.Fatalf("encoded attributed-text envelope order = %s", first)
	}

	var envelope encodedEnvelope
	if err := json.Unmarshal(first, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.TextVersion != DocumentVersion || envelope.Encoding != DocumentEncoding ||
		envelope.ChecksumAlgorithm != DocumentChecksumAlgorithm || envelope.Text.Authority != AuthorityNone ||
		len(envelope.Text.Records) != 1 || len(envelope.Text.Omissions) != 0 ||
		envelope.Text.Subject.SymbolVisible != fixture.visible || envelope.Text.Records[0].Text == "" ||
		envelope.Text.Usage.Fragments != 1 || envelope.Text.Usage.TextBytes != uint64(len(envelope.Text.Records[0].Text)) {
		t.Fatalf("attributed-text envelope = %+v", envelope)
	}
	stored, err := ParseDocumentChecksum(envelope.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if computed := independentDocumentChecksum(t, envelope.Text); stored != computed {
		t.Fatalf("document checksum = %s, computed = %s", stored, computed)
	}
	const expectedChecksum = "fytextdoc0-sha256-ed3d884108140fa00e4fbf21ce12f6368fd723c9b09fe4de810b3a6050a2bea3"
	if envelope.Checksum != expectedChecksum {
		t.Fatalf("frozen document checksum = %s, want %s", envelope.Checksum, expectedChecksum)
	}
	if _, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, request, uint64(len(first)-1)); !errors.Is(err, ErrDocumentTooLarge) {
		t.Fatalf("small final output bound = %v", err)
	}
	exactBound, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, request, uint64(len(first)))
	if err != nil || !bytes.Equal(first, exactBound) {
		t.Fatalf("exact final output bound = (%d bytes, %v)", len(exactBound), err)
	}
	if _, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, request, 0); err == nil {
		t.Fatal("zero final output bound succeeded")
	}
	if _, err := ParseDocumentChecksum(DocumentChecksumPrefix + strings.Repeat("A", 64)); err == nil {
		t.Fatal("uppercase document checksum succeeded")
	}
}

func TestEncodeWorkspaceSourcePreservesAtomicOmissionsWithoutUnsafeText(t *testing.T) {
	t.Parallel()
	fixture := loadAttributedTextFixture(t, []byte(attributedTextSource))
	large := Budget{MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20}
	baseline, err := ExtractWorkspaceSource(fixture.workspace, fixture.index, sourceRequest(fixture.visible, large))
	if err != nil || len(baseline.Records) != 1 {
		t.Fatalf("baseline extraction = (%+v, %v)", baseline, err)
	}
	length := uint64(len(baseline.Records[0].View().Text))
	limitedRequest := sourceRequest(fixture.visible, Budget{
		MaxFragments: 1, MaxFragmentBytes: length - 1, MaxTextBytes: length,
	})
	limitedBytes, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, limitedRequest, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	var limited encodedEnvelope
	if err := json.Unmarshal(limitedBytes, &limited); err != nil {
		t.Fatal(err)
	}
	if len(limited.Text.Records) != 0 || len(limited.Text.Omissions) != 1 ||
		limited.Text.Omissions[0].Code != OmissionFragmentByteLimit || limited.Text.Omissions[0].Bytes != length ||
		limited.Text.Usage != (Usage{}) || bytes.Contains(limitedBytes, []byte("Ignore previous instructions")) {
		t.Fatalf("limited document = %+v", limited.Text)
	}

	bidiSource := strings.Replace(attributedTextSource, "λ:", "λ:\u202e", 1)
	bidi := loadAttributedTextFixture(t, []byte(bidiSource))
	bidiBytes, err := EncodeWorkspaceSource(bidi.workspace, bidi.index, sourceRequest(bidi.visible, large), 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	var bidiEnvelope encodedEnvelope
	if err := json.Unmarshal(bidiBytes, &bidiEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(bidiEnvelope.Text.Records) != 0 || len(bidiEnvelope.Text.Omissions) != 1 ||
		bidiEnvelope.Text.Omissions[0].Code != OmissionDisallowedControl || bytes.Contains(bidiBytes, []byte("\\u202e")) {
		t.Fatalf("bidi document = %+v", bidiEnvelope.Text)
	}
}

func TestEncodedDocumentRejectsMutatedExtractionResults(t *testing.T) {
	t.Parallel()
	fixture := loadAttributedTextFixture(t, []byte(attributedTextSource))
	request := sourceRequest(fixture.visible, Budget{
		MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20,
	})
	baseline, err := ExtractWorkspaceSource(fixture.workspace, fixture.index, request)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []func(*Result){
		func(result *Result) { result.Subject.Path = "other.fy" },
		func(result *Result) { result.Selection.Kinds[0] = KindDeclarationDocumentation },
		func(result *Result) { result.Usage.TextBytes++ },
		func(result *Result) {
			result.Records[0].value.Text = strings.Repeat("x", len(result.Records[0].value.Text))
		},
		func(result *Result) { result.Omissions = append(result.Omissions, Omission{Code: OmissionInvalidUTF8}) },
	}
	for index, mutate := range mutations {
		candidate := baseline
		candidate.Selection.Kinds = append([]Kind(nil), baseline.Selection.Kinds...)
		candidate.Selection.Origins = append([]Origin(nil), baseline.Selection.Origins...)
		candidate.Records = append([]Record(nil), baseline.Records...)
		candidate.Omissions = append([]Omission(nil), baseline.Omissions...)
		mutate(&candidate)
		if _, err := documentFromResult(candidate); err == nil {
			t.Fatalf("mutated result %d encoded successfully", index)
		}
	}
}

func independentDocumentChecksum(t *testing.T, document encodedDocument) DocumentChecksum {
	t.Helper()
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("fairyc.attributed-text-checksum"))
	_, _ = digest.Write([]byte{0, 0})
	for _, value := range [][]byte{[]byte(DocumentVersion), []byte(DocumentEncoding), payload} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = digest.Write(length[:])
		_, _ = digest.Write(value)
	}
	var result DocumentChecksum
	copy(result[:], digest.Sum(nil))
	return result
}
