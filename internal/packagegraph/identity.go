package packagegraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// InstanceIDAlgorithm names the digest fixed by the v1 package-instance contract.
	InstanceIDAlgorithm = "sha256"
	// InstanceIDEncoding names the canonical byte encoding hashed by the v1 contract.
	InstanceIDEncoding = "fairyc.package-instance/v1"
	// InstanceIDPrefix keeps serialized IDs self-identifying without making them paths.
	InstanceIDPrefix = "fypi1-sha256-"

	IdentityReasonEmptyComponent            = "empty-component"
	IdentityReasonInvalidUTF8               = "invalid-utf8"
	IdentityReasonEmptyFeature              = "empty-feature"
	IdentityReasonFeaturesNotSortedOrUnique = "features-not-sorted-unique"
)

const instanceIDDomain = "fairyc.package-instance"

// IdentityValidationError gives fixture-stable reasons without freezing diagnostic wording.
type IdentityValidationError struct {
	Field  string
	Reason string
}

func (e *IdentityValidationError) Error() string {
	return fmt.Sprintf("invalid package identity %s: %s", e.Field, e.Reason)
}

// InstanceID is the full 256-bit digest of one canonical resolved package identity.
// It is independent from build-local HIR package numbering and native ABI names.
type InstanceID [sha256.Size]byte

// String returns the single canonical text representation accepted by ParseInstanceID.
func (id InstanceID) String() string {
	return InstanceIDPrefix + hex.EncodeToString(id[:])
}

// MarshalText implements encoding.TextMarshaler for future lockfile and map records.
func (id InstanceID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText accepts only the lowercase, full-width canonical representation.
func (id *InstanceID) UnmarshalText(text []byte) error {
	parsed, err := ParseInstanceID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// ParseInstanceID parses the canonical full-digest text representation.
func ParseInstanceID(value string) (InstanceID, error) {
	var result InstanceID
	if !strings.HasPrefix(value, InstanceIDPrefix) {
		return result, fmt.Errorf("package instance id has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, InstanceIDPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("package instance id must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode package instance id: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// ComputeInstanceID validates and hashes the canonical encoding of identity.
func ComputeInstanceID(identity Identity) (InstanceID, error) {
	encoded, err := EncodeIdentity(identity)
	if err != nil {
		return InstanceID{}, err
	}
	return sha256.Sum256(encoded), nil
}

// EncodeIdentity returns the platform-independent bytes hashed by PackageInstanceId v1.
// Components remain exact UTF-8 byte strings; the contract performs no path, case, or Unicode
// normalization because those policies belong to the source resolver that produced the tuple.
func EncodeIdentity(identity Identity) ([]byte, error) {
	if err := ValidateIdentity(identity); err != nil {
		return nil, err
	}

	var encoded bytes.Buffer
	encoded.Grow(len(instanceIDDomain) + len(identity.PackageUID) + len(identity.SourceIdentity) + len(identity.VersionIdentity) + 64)
	encoded.WriteString(instanceIDDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeIdentityString(&encoded, 'u', identity.PackageUID)
	writeIdentityString(&encoded, 's', identity.SourceIdentity)
	writeIdentityString(&encoded, 'v', identity.VersionIdentity)
	encoded.WriteByte('f')
	writeIdentityUint64(&encoded, uint64(len(identity.Features)))
	for _, feature := range identity.Features {
		writeIdentityUint64(&encoded, uint64(len(feature)))
		encoded.WriteString(feature)
	}
	return encoded.Bytes(), nil
}

// ValidateIdentity enforces the exact preconditions of the canonical v1 encoding.
func ValidateIdentity(identity Identity) error {
	components := []struct {
		name  string
		value string
	}{
		{name: "packageUid", value: identity.PackageUID},
		{name: "sourceIdentity", value: identity.SourceIdentity},
		{name: "versionIdentity", value: identity.VersionIdentity},
	}
	for _, component := range components {
		if component.value == "" {
			return &IdentityValidationError{Field: component.name, Reason: IdentityReasonEmptyComponent}
		}
		if !utf8.ValidString(component.value) {
			return &IdentityValidationError{Field: component.name, Reason: IdentityReasonInvalidUTF8}
		}
	}
	if !sortedUnique(identity.Features, true) {
		return &IdentityValidationError{Field: "features", Reason: IdentityReasonFeaturesNotSortedOrUnique}
	}
	for _, feature := range identity.Features {
		if feature == "" {
			return &IdentityValidationError{Field: "features", Reason: IdentityReasonEmptyFeature}
		}
		if !utf8.ValidString(feature) {
			return &IdentityValidationError{Field: "features", Reason: IdentityReasonInvalidUTF8}
		}
	}
	return nil
}

func writeIdentityString(buffer *bytes.Buffer, tag byte, value string) {
	buffer.WriteByte(tag)
	writeIdentityUint64(buffer, uint64(len(value)))
	buffer.WriteString(value)
}

func writeIdentityUint64(buffer *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buffer.Write(raw[:])
}
