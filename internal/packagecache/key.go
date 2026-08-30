// Package packagecache defines deterministic whole-package cache keys and bounded storage.
package packagecache

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

const (
	KeyAlgorithm = "sha256"
	KeyEncoding  = "fairyc.package-cache-key/v3"
	KeyPrefix    = "fycache3-sha256-"
)

const keyDomain = "fairyc.package-cache-key"

// Dependency binds one source-independent dependency coordinate to the public interface consumed from it.
type Dependency struct {
	Coordinate packagegraph.CoordinateID
	Interface  moduleinterface.Hash
}

// Inputs contains every compatibility and invalidation member needed for whole-package reuse.
// Set-like slices must already be sorted and unique so producers cannot accidentally disagree.
type Inputs struct {
	CompilerBuild        string
	LanguageRevision     string
	MIRVersion           string
	TargetSchema         string
	ABIProfile           string
	ToolchainFingerprint string
	Backend              string
	ProfileFlags         []string
	RuntimeABI           string
	StandardLibrary      packagegraph.InstanceID
	Package              packagegraph.InstanceID
	Source               packagecontent.Digest
	CurrentInterface     moduleinterface.Hash
	Dependencies         []Dependency
	Features             []string
}

// Key is the complete SHA-256 digest of canonical cache inputs.
type Key [sha256.Size]byte

func (key Key) String() string {
	return KeyPrefix + hex.EncodeToString(key[:])
}

func (key Key) MarshalText() ([]byte, error) {
	return []byte(key.String()), nil
}

func (key *Key) UnmarshalText(text []byte) error {
	parsed, err := ParseKey(string(text))
	if err != nil {
		return err
	}
	*key = parsed
	return nil
}

// ParseKey accepts only the complete lowercase v2 representation.
func ParseKey(value string) (Key, error) {
	var result Key
	if !strings.HasPrefix(value, KeyPrefix) {
		return result, fmt.Errorf("package cache key has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, KeyPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("package cache key must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode package cache key: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// Compute validates and hashes one canonical whole-package input tuple.
func Compute(inputs Inputs) (Key, error) {
	encoded, err := Encode(inputs)
	if err != nil {
		return Key{}, err
	}
	return sha256.Sum256(encoded), nil
}

// Encode returns the versioned, platform-independent cache-key bytes.
func Encode(inputs Inputs) ([]byte, error) {
	if err := validateInputs(inputs); err != nil {
		return nil, err
	}

	var encoded bytes.Buffer
	encoded.WriteString(keyDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(3)
	writeString(&encoded, packagegraph.InstanceIDEncoding)
	writeString(&encoded, packagegraph.InstanceIDAlgorithm)
	writeString(&encoded, packagegraph.CoordinateIDEncoding)
	writeString(&encoded, packagegraph.CoordinateIDAlgorithm)
	writeString(&encoded, packagecontent.DigestEncoding)
	writeString(&encoded, packagecontent.DigestAlgorithm)
	writeString(&encoded, moduleinterface.HashEncoding)
	writeString(&encoded, moduleinterface.HashAlgorithm)
	writeString(&encoded, inputs.CompilerBuild)
	writeString(&encoded, inputs.LanguageRevision)
	writeString(&encoded, inputs.MIRVersion)
	writeString(&encoded, inputs.TargetSchema)
	writeString(&encoded, inputs.ABIProfile)
	writeString(&encoded, inputs.ToolchainFingerprint)
	writeString(&encoded, inputs.Backend)
	writeStrings(&encoded, inputs.ProfileFlags)
	writeString(&encoded, inputs.RuntimeABI)
	encoded.Write(inputs.StandardLibrary[:])
	encoded.Write(inputs.Package[:])
	encoded.Write(inputs.Source[:])
	encoded.Write(inputs.CurrentInterface[:])
	writeUint64(&encoded, uint64(len(inputs.Dependencies)))
	for _, dependency := range inputs.Dependencies {
		encoded.Write(dependency.Coordinate[:])
		encoded.Write(dependency.Interface[:])
	}
	writeStrings(&encoded, inputs.Features)
	return encoded.Bytes(), nil
}

func validateInputs(inputs Inputs) error {
	required := []struct {
		name  string
		value string
	}{
		{name: "compilerBuild", value: inputs.CompilerBuild},
		{name: "languageRevision", value: inputs.LanguageRevision},
		{name: "mirVersion", value: inputs.MIRVersion},
		{name: "targetSchema", value: inputs.TargetSchema},
		{name: "abiProfile", value: inputs.ABIProfile},
		{name: "toolchainFingerprint", value: inputs.ToolchainFingerprint},
		{name: "backend", value: inputs.Backend},
		{name: "runtimeAbi", value: inputs.RuntimeABI},
	}
	for _, field := range required {
		if field.value == "" || !utf8.ValidString(field.value) {
			return fmt.Errorf("package cache %s must be nonempty valid UTF-8", field.name)
		}
	}
	if isZero(inputs.StandardLibrary[:]) {
		return fmt.Errorf("package cache standardLibrary must be set")
	}
	if isZero(inputs.Package[:]) {
		return fmt.Errorf("package cache package must be set")
	}
	if isZero(inputs.Source[:]) {
		return fmt.Errorf("package cache source must be set")
	}
	if isZero(inputs.CurrentInterface[:]) {
		return fmt.Errorf("package cache currentInterface must be set")
	}
	if err := validateSortedStrings("profileFlags", inputs.ProfileFlags); err != nil {
		return err
	}
	if err := validateSortedStrings("features", inputs.Features); err != nil {
		return err
	}
	for index, dependency := range inputs.Dependencies {
		if isZero(dependency.Coordinate[:]) || isZero(dependency.Interface[:]) {
			return fmt.Errorf("package cache dependency %d must contain a coordinate and interface hash", index)
		}
		if index > 0 && bytes.Compare(inputs.Dependencies[index-1].Coordinate[:], dependency.Coordinate[:]) >= 0 {
			return fmt.Errorf("package cache dependencies must be sorted and unique by PackageCoordinateId")
		}
	}
	return nil
}

func isZero(value []byte) bool {
	for _, element := range value {
		if element != 0 {
			return false
		}
	}
	return true
}

func validateSortedStrings(name string, values []string) error {
	for index, value := range values {
		if value == "" || !utf8.ValidString(value) {
			return fmt.Errorf("package cache %s must contain nonempty valid UTF-8", name)
		}
		if index > 0 && values[index-1] >= value {
			return fmt.Errorf("package cache %s must be sorted and unique", name)
		}
	}
	return nil
}

func writeStrings(buffer *bytes.Buffer, values []string) {
	writeUint64(buffer, uint64(len(values)))
	for _, value := range values {
		writeString(buffer, value)
	}
}

func writeString(buffer *bytes.Buffer, value string) {
	writeUint64(buffer, uint64(len(value)))
	buffer.WriteString(value)
}

func writeUint64(buffer *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buffer.Write(raw[:])
}

// CanonicalizeDependencies returns a sorted copy for callers assembling a key from graph edges.
// Duplicate package coordinates reject because one coordinate cannot expose two interface hashes.
func CanonicalizeDependencies(values []Dependency) ([]Dependency, error) {
	result := slices.Clone(values)
	slices.SortFunc(result, func(left, right Dependency) int {
		return bytes.Compare(left.Coordinate[:], right.Coordinate[:])
	})
	for index := 1; index < len(result); index++ {
		if result[index-1].Coordinate == result[index].Coordinate {
			return nil, fmt.Errorf("duplicate package cache dependency %s", result[index].Coordinate)
		}
	}
	return result, nil
}
