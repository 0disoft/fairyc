package packagegraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	CoordinateIDAlgorithm = "sha256"
	CoordinateIDEncoding  = "fairyc.package-coordinate/v1"
	CoordinateIDPrefix    = "fypc1-sha256-"
)

const coordinateIDDomain = "fairyc.package-coordinate"

// Coordinate is the source-content-independent package namespace used by public linking. Exact
// source identity remains owned by Identity and InstanceID.
type Coordinate struct {
	PackageUID      string
	VersionIdentity string
	Features        []string
}

// CoordinateID is the full digest of one canonical package coordinate.
type CoordinateID [sha256.Size]byte

func (id CoordinateID) String() string {
	return CoordinateIDPrefix + hex.EncodeToString(id[:])
}

func (id CoordinateID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *CoordinateID) UnmarshalText(text []byte) error {
	parsed, err := ParseCoordinateID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// ParseCoordinateID accepts only the canonical lowercase full-digest spelling.
func ParseCoordinateID(value string) (CoordinateID, error) {
	var result CoordinateID
	if !strings.HasPrefix(value, CoordinateIDPrefix) {
		return result, fmt.Errorf("package coordinate id has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, CoordinateIDPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("package coordinate id must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode package coordinate id: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

func CoordinateOf(identity Identity) Coordinate {
	return Coordinate{
		PackageUID: identity.PackageUID, VersionIdentity: identity.VersionIdentity,
		Features: append([]string(nil), identity.Features...),
	}
}

func ComputeCoordinateID(coordinate Coordinate) (CoordinateID, error) {
	encoded, err := EncodeCoordinate(coordinate)
	if err != nil {
		return CoordinateID{}, err
	}
	return sha256.Sum256(encoded), nil
}

// EncodeCoordinate returns the exact v1 byte stream. Source identity is absent by construction.
func EncodeCoordinate(coordinate Coordinate) ([]byte, error) {
	if err := ValidateCoordinate(coordinate); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	encoded.Grow(len(coordinateIDDomain) + len(coordinate.PackageUID) + len(coordinate.VersionIdentity) + 48)
	encoded.WriteString(coordinateIDDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeIdentityString(&encoded, 'u', coordinate.PackageUID)
	writeIdentityString(&encoded, 'v', coordinate.VersionIdentity)
	encoded.WriteByte('f')
	writeIdentityUint64(&encoded, uint64(len(coordinate.Features)))
	for _, feature := range coordinate.Features {
		writeIdentityUint64(&encoded, uint64(len(feature)))
		encoded.WriteString(feature)
	}
	return encoded.Bytes(), nil
}

func ValidateCoordinate(coordinate Coordinate) error {
	components := []struct {
		name  string
		value string
	}{
		{name: "packageUid", value: coordinate.PackageUID},
		{name: "versionIdentity", value: coordinate.VersionIdentity},
	}
	for _, component := range components {
		if component.value == "" {
			return &IdentityValidationError{Field: component.name, Reason: IdentityReasonEmptyComponent}
		}
		if !utf8.ValidString(component.value) {
			return &IdentityValidationError{Field: component.name, Reason: IdentityReasonInvalidUTF8}
		}
	}
	if !sortedUnique(coordinate.Features, true) {
		return &IdentityValidationError{Field: "features", Reason: IdentityReasonFeaturesNotSortedOrUnique}
	}
	for _, feature := range coordinate.Features {
		if feature == "" {
			return &IdentityValidationError{Field: "features", Reason: IdentityReasonEmptyFeature}
		}
		if !utf8.ValidString(feature) {
			return &IdentityValidationError{Field: "features", Reason: IdentityReasonInvalidUTF8}
		}
	}
	return nil
}
