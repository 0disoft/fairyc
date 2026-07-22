package attributedtext

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	DocumentVersion           = PolicyVersion
	DocumentEncoding          = "fairyc.attributed-text-json/v0"
	DocumentChecksumAlgorithm = "fairyc.attributed-text-checksum/v0-sha256"
	DocumentChecksumPrefix    = "fytextdoc0-sha256-"
	documentChecksumDomain    = "fairyc.attributed-text-checksum"
)

var ErrDocumentTooLarge = errors.New("Atlas attributed-text document exceeds the caller byte limit")

// DocumentChecksum binds the canonical logical document to its version and JSON encoding. It is
// integrity evidence only and carries no origin, review, authentication, or execution authority.
type DocumentChecksum [sha256.Size]byte

func (checksum DocumentChecksum) String() string {
	return DocumentChecksumPrefix + hex.EncodeToString(checksum[:])
}

func ParseDocumentChecksum(value string) (DocumentChecksum, error) {
	var result DocumentChecksum
	if !strings.HasPrefix(value, DocumentChecksumPrefix) {
		return result, fmt.Errorf("Atlas attributed-text document checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, DocumentChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("Atlas attributed-text document checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode Atlas attributed-text document checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

type encodedEnvelope struct {
	TextVersion       string          `json:"text_version"`
	Encoding          string          `json:"encoding"`
	ChecksumAlgorithm string          `json:"checksum_algorithm"`
	Checksum          string          `json:"checksum"`
	Text              encodedDocument `json:"text"`
}

type encodedDocument struct {
	Authority Authority  `json:"authority"`
	Subject   Subject    `json:"subject"`
	Selection Selection  `json:"selection"`
	Budget    Budget     `json:"budget"`
	Usage     Usage      `json:"usage"`
	Records   []View     `json:"records"`
	Omissions []Omission `json:"omissions"`
}

// EncodeWorkspaceSource performs exact-current extraction itself and emits one compact canonical
// document. It never accepts a caller-authored Result that could impersonate compiler-selected text.
func EncodeWorkspaceSource(
	workspace workspaceloader.Workspace,
	index *atlasmap.QueryIndex,
	request Request,
	maxBytes uint64,
) ([]byte, error) {
	if maxBytes == 0 {
		return nil, fmt.Errorf("Atlas attributed-text document byte limit must be positive")
	}
	result, err := ExtractWorkspaceSource(workspace, index, request)
	if err != nil {
		return nil, err
	}
	document, err := documentFromResult(result)
	if err != nil {
		return nil, err
	}
	checksum, err := computeDocumentChecksum(document)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(encodedEnvelope{
		TextVersion: DocumentVersion, Encoding: DocumentEncoding,
		ChecksumAlgorithm: DocumentChecksumAlgorithm, Checksum: checksum.String(), Text: document,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Atlas attributed-text document: %w", err)
	}
	if uint64(len(encoded)) > maxBytes {
		return nil, ErrDocumentTooLarge
	}
	return encoded, nil
}

func documentFromResult(result Result) (encodedDocument, error) {
	if err := validateResult(result); err != nil {
		return encodedDocument{}, err
	}
	records := make([]View, 0, len(result.Records))
	for _, record := range result.Records {
		records = append(records, record.View())
	}
	return encodedDocument{
		Authority: AuthorityNone,
		Subject:   result.Subject,
		Selection: Selection{Kinds: slices.Clone(result.Selection.Kinds), Origins: slices.Clone(result.Selection.Origins)},
		Budget:    result.Budget,
		Usage:     result.Usage,
		Records:   records,
		Omissions: append(make([]Omission, 0, len(result.Omissions)), result.Omissions...),
	}, nil
}

func validateResult(result Result) error {
	if err := ValidateBudget(result.Budget); err != nil {
		return err
	}
	if len(result.Selection.Kinds) != 1 || result.Selection.Kinds[0] != KindSourceExcerpt ||
		len(result.Selection.Origins) != 1 || result.Selection.Origins[0] != OriginWorkspaceAuthored {
		return fmt.Errorf("Atlas attributed-text result has a noncanonical selection")
	}
	if err := validateSubject(result.Subject); err != nil {
		return err
	}
	if len(result.Records)+len(result.Omissions) != 1 || len(result.Records) > result.Budget.MaxFragments {
		return fmt.Errorf("Atlas attributed-text result must contain exactly one record or omission")
	}
	var textBytes uint64
	for _, record := range result.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("validate Atlas attributed-text result record: %w", err)
		}
		view := record.View()
		if !recordMatchesSubject(view, result.Subject) || view.Kind != KindSourceExcerpt ||
			view.Origin != OriginWorkspaceAuthored || view.Authority != AuthorityNone {
			return fmt.Errorf("Atlas attributed-text result record does not match its subject and selection")
		}
		length := uint64(len(view.Text))
		if length > result.Budget.MaxFragmentBytes || length > result.Budget.MaxTextBytes-textBytes {
			return fmt.Errorf("Atlas attributed-text result exceeds its declared text budget")
		}
		textBytes += length
	}
	if result.Usage.Fragments != len(result.Records) || result.Usage.TextBytes != textBytes {
		return fmt.Errorf("Atlas attributed-text result usage does not match its records")
	}
	for _, omission := range result.Omissions {
		if !validOmissionCode(omission.Code) || omission.Kind != KindSourceExcerpt ||
			omission.SymbolVisible != result.Subject.SymbolVisible || omission.Bytes != result.Subject.EndByte-result.Subject.StartByte {
			return fmt.Errorf("Atlas attributed-text result has an invalid omission")
		}
		if omission.Code == OmissionOriginNotAdmitted {
			if omission.Origin != OriginCompilerBundled {
				return fmt.Errorf("Atlas attributed-text origin omission has an invalid actual origin")
			}
		} else if omission.Origin != OriginWorkspaceAuthored {
			return fmt.Errorf("Atlas attributed-text omission has an invalid workspace origin")
		}
	}
	return nil
}

func validateSubject(subject Subject) error {
	visible, err := symbolidentity.ParseVisibleID(subject.SymbolVisible)
	if err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid visible SymbolId")
	}
	full, err := symbolidentity.ParseFullDigest(subject.SymbolFull)
	if err != nil || !bytes.Equal(visible[:], full[:len(visible)]) {
		return fmt.Errorf("Atlas attributed-text subject has an invalid SymbolId pair")
	}
	if !projectmanifest.ValidPackageUID(subject.PackageUID) {
		return fmt.Errorf("Atlas attributed-text subject has an invalid package UID")
	}
	if _, err := packagegraph.ParseInstanceID(subject.PackageInstance); err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid package instance")
	}
	if _, err := packagegraph.ParseCoordinateID(subject.PackageCoordinate); err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid package coordinate")
	}
	if _, err := packagecontent.ParseDigest(subject.PackageSource); err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid package source identity")
	}
	if err := packagecontent.ValidatePath(subject.Path); err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid source path")
	}
	if subject.FileKind != semanticsnapshot.FileKindSource && subject.FileKind != semanticsnapshot.FileKindTest {
		return fmt.Errorf("Atlas attributed-text subject has an invalid file kind")
	}
	if _, err := packagecontent.ParseFileDigest(subject.FileIdentity); err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid file identity")
	}
	if _, err := packagecontent.ParseLineMapID(subject.LineMapIdentity); err != nil {
		return fmt.Errorf("Atlas attributed-text subject has an invalid line-map identity")
	}
	if subject.EndByte <= subject.StartByte {
		return fmt.Errorf("Atlas attributed-text subject has an invalid byte span")
	}
	return nil
}

func recordMatchesSubject(view View, subject Subject) bool {
	return view.SymbolVisible == subject.SymbolVisible && view.SymbolFull == subject.SymbolFull &&
		view.PackageUID == subject.PackageUID && view.PackageInstance == subject.PackageInstance &&
		view.PackageCoordinate == subject.PackageCoordinate && view.PackageSource == subject.PackageSource &&
		view.Path == subject.Path && view.FileKind == subject.FileKind && view.FileIdentity == subject.FileIdentity &&
		view.LineMapIdentity == subject.LineMapIdentity && view.StartByte == subject.StartByte && view.EndByte == subject.EndByte
}

func validOmissionCode(code OmissionCode) bool {
	return code == OmissionOriginNotAdmitted || code == OmissionInvalidUTF8 ||
		code == OmissionDisallowedControl || code == OmissionFragmentByteLimit
}

func computeDocumentChecksum(document encodedDocument) (DocumentChecksum, error) {
	payload, err := json.Marshal(document)
	if err != nil {
		return DocumentChecksum{}, fmt.Errorf("encode Atlas attributed-text checksum payload: %w", err)
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte(documentChecksumDomain))
	_, _ = digest.Write([]byte{0, 0})
	writeChecksumBytes(digest, []byte(DocumentVersion))
	writeChecksumBytes(digest, []byte(DocumentEncoding))
	writeChecksumBytes(digest, payload)
	var result DocumentChecksum
	copy(result[:], digest.Sum(nil))
	return result, nil
}
