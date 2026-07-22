// Package symbolidentity owns the provisional deterministic SymbolId contract consumed by Fairy
// Atlas. It derives identity only from stable semantic coordinates and canonical signatures.
package symbolidentity

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/source"
)

const (
	Algorithm         = "fairyc.symbol-id/v1-sha256-128+256"
	Encoding          = "fairyc.symbol-identity/v1"
	SignatureEncoding = "fairyc.symbol-signature/v1"
	VisiblePrefix     = "fysym1-sha256-128-"
	DigestPrefix      = "fysym1-sha256-256-"
)

const identityDomain = "fairyc.symbol-identity"

const (
	ReasonEmptyComponent   = "empty-component"
	ReasonInvalidUTF8      = "invalid-utf8"
	ReasonControlCharacter = "control-character"
	ReasonInvalidPath      = "invalid-qualified-path"
	ReasonInvalidKind      = "invalid-declaration-kind"
	ReasonEmptySignature   = "empty-signature"
)

// Descriptor is the exact tuple hashed by the v1 algorithm. Signature contains bytes produced by
// SignatureEncoding, not source text or a display spelling.
type Descriptor struct {
	PackageUID      string
	Module          string
	Kind            string
	DeclarationPath string
	GenericArity    uint64
	Signature       []byte
}

// ValidationError carries fixture-stable rejection reasons without freezing diagnostic prose.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid symbol identity %s: %s", e.Field, e.Reason)
}

// VisibleID is the compact 128-bit prefix used for ordinary map joins and display.
type VisibleID [sha256.Size / 2]byte

func (id VisibleID) String() string {
	return VisiblePrefix + hex.EncodeToString(id[:])
}

func (id VisibleID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *VisibleID) UnmarshalText(text []byte) error {
	parsed, err := ParseVisibleID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// FullDigest is stored beside VisibleID so readers can reject a compact-ID collision.
type FullDigest [sha256.Size]byte

func (digest FullDigest) String() string {
	return DigestPrefix + hex.EncodeToString(digest[:])
}

func (digest FullDigest) MarshalText() ([]byte, error) {
	return []byte(digest.String()), nil
}

func (digest *FullDigest) UnmarshalText(text []byte) error {
	parsed, err := ParseFullDigest(string(text))
	if err != nil {
		return err
	}
	*digest = parsed
	return nil
}

// ID retains both forms required by the collision-detecting map contract.
type ID struct {
	Visible VisibleID
	Full    FullDigest
}

func ParseVisibleID(value string) (VisibleID, error) {
	var result VisibleID
	if err := parseCanonicalHex(value, VisiblePrefix, result[:], "symbol visible id"); err != nil {
		return VisibleID{}, err
	}
	return result, nil
}

func ParseFullDigest(value string) (FullDigest, error) {
	var result FullDigest
	if err := parseCanonicalHex(value, DigestPrefix, result[:], "symbol full digest"); err != nil {
		return FullDigest{}, err
	}
	return result, nil
}

func parseCanonicalHex(value, prefix string, destination []byte, label string) error {
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("%s has invalid prefix", label)
	}
	digestText := strings.TrimPrefix(value, prefix)
	if len(digestText) != len(destination)*2 || digestText != strings.ToLower(digestText) {
		return fmt.Errorf("%s must contain %d lowercase hexadecimal digits", label, len(destination)*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	copy(destination, decoded)
	return nil
}

// Compute validates and hashes one canonical identity tuple.
func Compute(descriptor Descriptor) (ID, error) {
	encoded, err := Encode(descriptor)
	if err != nil {
		return ID{}, err
	}
	full := FullDigest(sha256.Sum256(encoded))
	var visible VisibleID
	copy(visible[:], full[:len(visible)])
	return ID{Visible: visible, Full: full}, nil
}

// Encode returns the platform-independent v1 bytes. Every variable component is tagged and
// length-prefixed, so concatenation boundaries cannot collide.
func Encode(descriptor Descriptor) ([]byte, error) {
	if err := ValidateDescriptor(descriptor); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	encoded.Grow(len(identityDomain) + len(descriptor.PackageUID) + len(descriptor.Module) +
		len(descriptor.Kind) + len(descriptor.DeclarationPath) + len(descriptor.Signature) + 96)
	encoded.WriteString(identityDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeString(&encoded, 'u', descriptor.PackageUID)
	writeString(&encoded, 'm', descriptor.Module)
	writeString(&encoded, 'k', descriptor.Kind)
	writeString(&encoded, 'p', descriptor.DeclarationPath)
	encoded.WriteByte('g')
	writeUint64(&encoded, descriptor.GenericArity)
	encoded.WriteByte('s')
	writeUint64(&encoded, uint64(len(descriptor.Signature)))
	encoded.Write(descriptor.Signature)
	return encoded.Bytes(), nil
}

func ValidateDescriptor(descriptor Descriptor) error {
	if err := validateText("packageUid", descriptor.PackageUID); err != nil {
		return err
	}
	if err := validateQualifiedPath("module", descriptor.Module); err != nil {
		return err
	}
	if !validKind(descriptor.Kind) {
		return &ValidationError{Field: "kind", Reason: ReasonInvalidKind}
	}
	if err := validateQualifiedPath("declarationPath", descriptor.DeclarationPath); err != nil {
		return err
	}
	if len(descriptor.Signature) == 0 {
		return &ValidationError{Field: "signature", Reason: ReasonEmptySignature}
	}
	return nil
}

func validateText(field, value string) error {
	if value == "" {
		return &ValidationError{Field: field, Reason: ReasonEmptyComponent}
	}
	if !utf8.ValidString(value) {
		return &ValidationError{Field: field, Reason: ReasonInvalidUTF8}
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return &ValidationError{Field: field, Reason: ReasonControlCharacter}
		}
	}
	return nil
}

func validateQualifiedPath(field, value string) error {
	if err := validateText(field, value); err != nil {
		return err
	}
	for _, segment := range strings.Split(value, "::") {
		if !source.ValidIdentifier(segment) {
			return &ValidationError{Field: field, Reason: ReasonInvalidPath}
		}
	}
	return nil
}

func validKind(value string) bool {
	switch value {
	case "struct", "enum", "resource", "function", "foreign-function":
		return true
	default:
		return false
	}
}

func writeString(buffer *bytes.Buffer, tag byte, value string) {
	buffer.WriteByte(tag)
	writeUint64(buffer, uint64(len(value)))
	buffer.WriteString(value)
}

func writeUint64(buffer *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buffer.Write(raw[:])
}
