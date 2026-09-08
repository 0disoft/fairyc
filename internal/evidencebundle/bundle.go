// Package evidencebundle owns a canonical output-only composition of independently checksummed
// Atlas compiler facts and attributed text. It creates no new facts, text, authority, or input
// decoder and never merges untrusted text into the compiler-fact Context Pack.
package evidencebundle

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/contextpack"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	BundleVersion     = "fairyc.atlas-evidence/v0"
	BundleEncoding    = "fairyc.atlas-evidence-json/v0"
	ChecksumAlgorithm = "fairyc.atlas-evidence-checksum/v0-sha256"
	ChecksumPrefix    = "fyevidence0-sha256-"
	checksumDomain    = "fairyc.atlas-evidence-checksum"
)

const (
	AuthorityBundleNone   = "none"
	AuthorityCompilerFact = "compiler-fact"
)

var ErrBundleTooLarge = errors.New("Atlas Evidence Bundle exceeds the caller byte limit")

// Limits independently bound the three nested documents and the final composite. FactsBytes
// applies to the selected Context Pack; TextDocumentBytes applies separately to each text document.
type Limits struct {
	FactsBytes        uint64 `json:"facts_bytes"`
	TextDocumentBytes uint64 `json:"text_document_bytes"`
	BundleBytes       uint64 `json:"bundle_bytes"`
}

// Checksum binds the exact nested canonical documents, authority labels, and subject identity.
type Checksum [sha256.Size]byte

func (checksum Checksum) String() string {
	return ChecksumPrefix + hex.EncodeToString(checksum[:])
}

func ParseChecksum(value string) (Checksum, error) {
	var result Checksum
	if !strings.HasPrefix(value, ChecksumPrefix) {
		return result, fmt.Errorf("Atlas Evidence Bundle checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("Atlas Evidence Bundle checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode Atlas Evidence Bundle checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// EncodeShow compiles no input itself. It consumes one already exact-current query index, retained
// workspace, and matching semantic snapshot, then builds three child outputs through their owning
// encoders before composing them. Caller-authored JSON or text is never accepted.
func EncodeShow(
	workspace workspaceloader.Workspace,
	index *atlasmap.QueryIndex,
	snapshot semanticsnapshot.Snapshot,
	symbolID string,
	request atlasmap.RelationRequest,
	textBudget attributedtext.Budget,
	limits Limits,
) ([]byte, error) {
	return encodeShow(workspace, index, snapshot, symbolID, request, textBudget, limits, 0)
}

func encodeShow(
	workspace workspaceloader.Workspace,
	index *atlasmap.QueryIndex,
	snapshot semanticsnapshot.Snapshot,
	symbolID string,
	request atlasmap.RelationRequest,
	textBudget attributedtext.Budget,
	limits Limits,
	bundleVersion byte,
) ([]byte, error) {
	if limits.FactsBytes == 0 || limits.TextDocumentBytes == 0 || limits.BundleBytes == 0 {
		return nil, fmt.Errorf("Atlas Evidence Bundle requires positive component and final byte limits")
	}
	subject, err := index.ResolveAttributionSubject(symbolID)
	if err != nil {
		return nil, err
	}
	encodeFacts := contextpack.EncodeShow
	switch bundleVersion {
	case 0:
	case 1:
		encodeFacts = contextpack.EncodeShowV2
	case 2:
		encodeFacts = contextpack.EncodeShowV3
	default:
		return nil, fmt.Errorf("unsupported Atlas Evidence Bundle version %d", bundleVersion)
	}
	facts, err := encodeFacts(index, symbolID, request, limits.FactsBytes)
	if err != nil {
		return nil, err
	}
	baseTextRequest := attributedtext.Request{
		SymbolID: symbolID,
		Origins:  []attributedtext.Origin{attributedtext.OriginWorkspaceAuthored},
		Budget:   textBudget,
	}
	sourceRequest := baseTextRequest
	sourceRequest.Kinds = []attributedtext.Kind{attributedtext.KindSourceExcerpt}
	sourceDocument, err := attributedtext.EncodeWorkspaceSource(
		workspace, index, sourceRequest, limits.TextDocumentBytes,
	)
	if err != nil {
		return nil, err
	}
	documentationRequest := baseTextRequest
	documentationRequest.Kinds = []attributedtext.Kind{attributedtext.KindDeclarationDocumentation}
	documentationDocument, err := attributedtext.EncodeWorkspaceDocumentation(
		workspace, index, snapshot, documentationRequest, limits.TextDocumentBytes,
	)
	if err != nil {
		return nil, err
	}
	document := bundleDocument{
		Authority: authorityDescriptor{
			Bundle: AuthorityBundleNone, Facts: AuthorityCompilerFact,
			Source: attributedtext.AuthorityNone, Documentation: attributedtext.AuthorityNone,
		},
		Subject: subjectDescriptor{
			SymbolVisible: subject.Subject.Symbol.Visible, SymbolFull: subject.Subject.Symbol.Full,
			PackageUID: subject.Context.PackageUID, PackageInstance: subject.Context.PackageInstance,
		},
		Components: componentDocuments{
			Facts:         append(json.RawMessage(nil), facts...),
			Source:        append(json.RawMessage(nil), sourceDocument...),
			Documentation: append(json.RawMessage(nil), documentationDocument...),
		},
		Limits: limits,
	}
	version, encoding, algorithm, prefix := BundleVersion, BundleEncoding, ChecksumAlgorithm, ChecksumPrefix
	if bundleVersion == 1 {
		version, encoding, algorithm, prefix = BundleVersionV1, BundleEncodingV1, ChecksumAlgorithmV1, ChecksumPrefixV1
	} else if bundleVersion == 2 {
		version, encoding, algorithm, prefix = BundleVersionV2, BundleEncodingV2, ChecksumAlgorithmV2, ChecksumPrefixV2
	}
	checksum, err := computeChecksumVersion(document, bundleVersion)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(bundleEnvelope{
		BundleVersion: version, Encoding: encoding,
		ChecksumAlgorithm: algorithm, Checksum: prefix + hex.EncodeToString(checksum[:]), Bundle: document,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Atlas Evidence Bundle: %w", err)
	}
	if uint64(len(encoded)) > limits.BundleBytes {
		return nil, ErrBundleTooLarge
	}
	return encoded, nil
}

type bundleEnvelope struct {
	BundleVersion     string         `json:"bundle_version"`
	Encoding          string         `json:"encoding"`
	ChecksumAlgorithm string         `json:"checksum_algorithm"`
	Checksum          string         `json:"checksum"`
	Bundle            bundleDocument `json:"bundle"`
}

type bundleDocument struct {
	Authority  authorityDescriptor `json:"authority"`
	Subject    subjectDescriptor   `json:"subject"`
	Components componentDocuments  `json:"components"`
	Limits     Limits              `json:"limits"`
}

type authorityDescriptor struct {
	Bundle        string                   `json:"bundle"`
	Facts         string                   `json:"facts"`
	Source        attributedtext.Authority `json:"source"`
	Documentation attributedtext.Authority `json:"documentation"`
}

type subjectDescriptor struct {
	SymbolVisible   string `json:"symbol_visible"`
	SymbolFull      string `json:"symbol_full"`
	PackageUID      string `json:"package_uid"`
	PackageInstance string `json:"package_instance"`
}

type componentDocuments struct {
	Facts         json.RawMessage `json:"facts"`
	Source        json.RawMessage `json:"source"`
	Documentation json.RawMessage `json:"documentation"`
}

func computeChecksum(document bundleDocument) (Checksum, error) {
	return computeChecksumVersion(document, 0)
}

func computeChecksumVersion(document bundleDocument, bundleVersion byte) (Checksum, error) {
	payload, err := json.Marshal(document)
	if err != nil {
		return Checksum{}, fmt.Errorf("encode Atlas Evidence Bundle checksum payload: %w", err)
	}
	digest := sha256.New()
	version, encoding := BundleVersion, BundleEncoding
	if bundleVersion != 0 {
		_, _ = digest.Write([]byte(checksumDomain))
		_, _ = digest.Write([]byte{0, bundleVersion})
		if bundleVersion == 1 {
			version, encoding = BundleVersionV1, BundleEncodingV1
		} else if bundleVersion == 2 {
			version, encoding = BundleVersionV2, BundleEncodingV2
		} else {
			return Checksum{}, fmt.Errorf("unsupported Atlas Evidence Bundle checksum version %d", bundleVersion)
		}
	} else {
		writeChecksumHeader(digest)
	}
	writeChecksumBytes(digest, []byte(version))
	writeChecksumBytes(digest, []byte(encoding))
	writeChecksumBytes(digest, payload)
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result, nil
}

func writeChecksumHeader(destination hash.Hash) {
	_, _ = destination.Write([]byte(checksumDomain))
	_, _ = destination.Write([]byte{0, 0})
}

func writeChecksumBytes(destination hash.Hash, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = destination.Write(length[:])
	_, _ = destination.Write(value)
}
