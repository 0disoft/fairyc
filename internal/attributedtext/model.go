// Package attributedtext owns the closed, non-authoritative logical record, budget contract, exact
// retained-workspace source extraction gate, and canonical output-only JSON encoding for Atlas
// text. It performs no path reopening, decoding, command handling, or model integration.
package attributedtext

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/projectmanifest"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

const (
	PolicyVersion = "fairyc.attributed-text/v0"
	AuthorityNone = Authority("untrusted-text")
)

const (
	MaxFragmentsGuard     = 4_096
	MaxFragmentBytesGuard = uint64(8 << 20)
	MaxTextBytesGuard     = uint64(64 << 20)
)

type Authority string
type Kind string
type Origin string
type Format string

const (
	KindDeclarationDocumentation Kind = "declaration-documentation"
	KindSourceExcerpt            Kind = "source-excerpt"
)

const (
	OriginWorkspaceAuthored  Origin = "workspace-authored"
	OriginCompilerBundled    Origin = "compiler-bundled"
	OriginExternalDependency Origin = "external-dependency"
	OriginGenerated          Origin = "generated"
)

const (
	FormatFairyDeclarationDocUTF8 Format = "fairyc-declaration-doc-utf8"
	FormatFairySourceUTF8         Format = "fairyc-source-utf8"
)

const (
	ReasonInvalidKind        = "invalid-kind"
	ReasonInvalidOrigin      = "invalid-origin"
	ReasonInvalidPolicy      = "invalid-policy"
	ReasonInvalidAuthority   = "invalid-authority"
	ReasonInvalidSymbol      = "invalid-symbol"
	ReasonSymbolPairMismatch = "symbol-pair-mismatch"
	ReasonInvalidPackage     = "invalid-package"
	ReasonInvalidPath        = "invalid-path"
	ReasonInvalidFileKind    = "invalid-file-kind"
	ReasonInvalidFileID      = "invalid-file-identity"
	ReasonInvalidLineMapID   = "invalid-line-map-identity"
	ReasonInvalidFormat      = "invalid-format"
	ReasonInvalidSpan        = "invalid-span"
	ReasonEmptyText          = "empty-text"
	ReasonSpanLengthMismatch = "span-length-mismatch"
	ReasonInvalidUTF8        = "invalid-utf8"
	ReasonDisallowedControl  = "disallowed-control"
	ReasonInvalidChecksum    = "invalid-checksum"
	ReasonInvalidBudget      = "invalid-budget"
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid Atlas attributed text %s: %s", e.Field, e.Reason)
}

type Input struct {
	Kind              Kind
	Origin            Origin
	SymbolVisible     string
	SymbolFull        string
	PackageUID        string
	PackageInstance   string
	PackageCoordinate string
	PackageSource     string
	Path              string
	FileKind          semanticsnapshot.FileKind
	FileIdentity      string
	LineMapIdentity   string
	StartByte         uint64
	EndByte           uint64
	Format            Format
	Text              string
}

type View struct {
	Policy            string                    `json:"policy"`
	Authority         Authority                 `json:"authority"`
	Kind              Kind                      `json:"kind"`
	Origin            Origin                    `json:"origin"`
	SymbolVisible     string                    `json:"symbol_visible"`
	SymbolFull        string                    `json:"symbol_full"`
	PackageUID        string                    `json:"package_uid"`
	PackageInstance   string                    `json:"package_instance"`
	PackageCoordinate string                    `json:"package_coordinate"`
	PackageSource     string                    `json:"package_source"`
	Path              string                    `json:"path"`
	FileKind          semanticsnapshot.FileKind `json:"file_kind"`
	FileIdentity      string                    `json:"file_identity"`
	LineMapIdentity   string                    `json:"line_map_identity"`
	StartByte         uint64                    `json:"start_byte"`
	EndByte           uint64                    `json:"end_byte"`
	Format            Format                    `json:"format"`
	Text              string                    `json:"text"`
	Checksum          string                    `json:"checksum"`
}

type Record struct {
	value    View
	checksum Checksum
}

type Budget struct {
	MaxFragments     int    `json:"max_fragments"`
	MaxFragmentBytes uint64 `json:"max_fragment_bytes"`
	MaxTextBytes     uint64 `json:"max_text_bytes"`
}

func New(input Input) (Record, error) {
	value := View{
		Policy: PolicyVersion, Authority: AuthorityNone,
		Kind: input.Kind, Origin: input.Origin,
		SymbolVisible: input.SymbolVisible, SymbolFull: input.SymbolFull,
		PackageUID: input.PackageUID, PackageInstance: input.PackageInstance,
		PackageCoordinate: input.PackageCoordinate, PackageSource: input.PackageSource,
		Path: input.Path, FileKind: input.FileKind,
		FileIdentity: input.FileIdentity, LineMapIdentity: input.LineMapIdentity,
		StartByte: input.StartByte, EndByte: input.EndByte,
		Format: input.Format, Text: input.Text,
	}
	if err := validateValue(value); err != nil {
		return Record{}, err
	}
	checksum := computeChecksum(value)
	value.Checksum = checksum.String()
	return Record{value: value, checksum: checksum}, nil
}

func (record Record) View() View {
	return record.value
}

func (record Record) Checksum() Checksum {
	return record.checksum
}

func (record Record) Validate() error {
	if err := validateValue(record.value); err != nil {
		return err
	}
	parsed, err := ParseChecksum(record.value.Checksum)
	if err != nil || parsed != record.checksum || parsed != computeChecksum(record.value) {
		return &ValidationError{Field: "checksum", Reason: ReasonInvalidChecksum}
	}
	return nil
}

func ValidateBudget(budget Budget) error {
	if budget.MaxFragments <= 0 || budget.MaxFragments > MaxFragmentsGuard ||
		budget.MaxFragmentBytes == 0 || budget.MaxFragmentBytes > MaxFragmentBytesGuard ||
		budget.MaxTextBytes == 0 || budget.MaxTextBytes > MaxTextBytesGuard ||
		budget.MaxFragmentBytes > budget.MaxTextBytes {
		return &ValidationError{Field: "budget", Reason: ReasonInvalidBudget}
	}
	return nil
}

func validateValue(value View) error {
	if value.Policy != PolicyVersion {
		return &ValidationError{Field: "policy", Reason: ReasonInvalidPolicy}
	}
	if value.Authority != AuthorityNone {
		return &ValidationError{Field: "authority", Reason: ReasonInvalidAuthority}
	}
	if value.Kind != KindSourceExcerpt && value.Kind != KindDeclarationDocumentation {
		return &ValidationError{Field: "kind", Reason: ReasonInvalidKind}
	}
	if value.Origin != OriginWorkspaceAuthored && value.Origin != OriginCompilerBundled &&
		value.Origin != OriginExternalDependency && value.Origin != OriginGenerated {
		return &ValidationError{Field: "origin", Reason: ReasonInvalidOrigin}
	}
	visible, err := symbolidentity.ParseVisibleID(value.SymbolVisible)
	if err != nil {
		return &ValidationError{Field: "symbolVisible", Reason: ReasonInvalidSymbol}
	}
	full, err := symbolidentity.ParseFullDigest(value.SymbolFull)
	if err != nil {
		return &ValidationError{Field: "symbolFull", Reason: ReasonInvalidSymbol}
	}
	if !bytes.Equal(visible[:], full[:len(visible)]) {
		return &ValidationError{Field: "symbol", Reason: ReasonSymbolPairMismatch}
	}
	if !projectmanifest.ValidPackageUID(value.PackageUID) {
		return &ValidationError{Field: "packageUid", Reason: ReasonInvalidPackage}
	}
	if _, err := packagegraph.ParseInstanceID(value.PackageInstance); err != nil {
		return &ValidationError{Field: "packageInstance", Reason: ReasonInvalidPackage}
	}
	if _, err := packagegraph.ParseCoordinateID(value.PackageCoordinate); err != nil {
		return &ValidationError{Field: "packageCoordinate", Reason: ReasonInvalidPackage}
	}
	if _, err := packagecontent.ParseDigest(value.PackageSource); err != nil {
		return &ValidationError{Field: "packageSource", Reason: ReasonInvalidPackage}
	}
	if err := packagecontent.ValidatePath(value.Path); err != nil {
		return &ValidationError{Field: "path", Reason: ReasonInvalidPath}
	}
	if value.FileKind != semanticsnapshot.FileKindSource && value.FileKind != semanticsnapshot.FileKindTest {
		return &ValidationError{Field: "fileKind", Reason: ReasonInvalidFileKind}
	}
	if _, err := packagecontent.ParseFileDigest(value.FileIdentity); err != nil {
		return &ValidationError{Field: "fileIdentity", Reason: ReasonInvalidFileID}
	}
	if _, err := packagecontent.ParseLineMapID(value.LineMapIdentity); err != nil {
		return &ValidationError{Field: "lineMapIdentity", Reason: ReasonInvalidLineMapID}
	}
	if value.EndByte <= value.StartByte {
		return &ValidationError{Field: "span", Reason: ReasonInvalidSpan}
	}
	if value.Text == "" {
		return &ValidationError{Field: "text", Reason: ReasonEmptyText}
	}
	if uint64(len(value.Text)) != value.EndByte-value.StartByte {
		return &ValidationError{Field: "text", Reason: ReasonSpanLengthMismatch}
	}
	if !utf8.ValidString(value.Text) {
		return &ValidationError{Field: "text", Reason: ReasonInvalidUTF8}
	}
	for _, character := range value.Text {
		if isDisallowedControl(character) {
			return &ValidationError{Field: "text", Reason: ReasonDisallowedControl}
		}
	}
	if value.Kind == KindSourceExcerpt && value.Format != FormatFairySourceUTF8 ||
		value.Kind == KindDeclarationDocumentation && value.Format != FormatFairyDeclarationDocUTF8 {
		return &ValidationError{Field: "format", Reason: ReasonInvalidFormat}
	}
	return nil
}

func isDisallowedControl(character rune) bool {
	if character == '\t' || character == '\n' || character == '\r' {
		return false
	}
	if unicode.IsControl(character) {
		return true
	}
	return character == '\u061c' || character == '\u200e' || character == '\u200f' ||
		character >= '\u202a' && character <= '\u202e' ||
		character >= '\u2066' && character <= '\u2069'
}
