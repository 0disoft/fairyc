// Package packagecontent defines the exact package-source digest used by cache and Atlas records.
package packagecontent

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	pathpkg "path"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	DigestAlgorithm = "sha256"
	DigestEncoding  = "fairyc.package-source/v1"
	DigestPrefix    = "fysrc1-sha256-"
)

const sourceDomain = "fairyc.package-source"

// File is one source-root-relative canonical path and its exact bytes.
type File struct {
	Path  string
	Bytes []byte
}

// Digest is the full SHA-256 of one canonical package source graph.
type Digest [sha256.Size]byte

func (digest Digest) String() string {
	return DigestPrefix + hex.EncodeToString(digest[:])
}

func (digest Digest) MarshalText() ([]byte, error) {
	return []byte(digest.String()), nil
}

func (digest *Digest) UnmarshalText(text []byte) error {
	parsed, err := ParseDigest(string(text))
	if err != nil {
		return err
	}
	*digest = parsed
	return nil
}

// ParseDigest accepts only the lowercase full-digest text form.
func ParseDigest(value string) (Digest, error) {
	var result Digest
	if !strings.HasPrefix(value, DigestPrefix) {
		return result, fmt.Errorf("package source digest has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, DigestPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("package source digest must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode package source digest: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

// Encode returns the canonical bytes for the exact package source graph. Input order is ignored;
// path and file bytes are not normalized.
func Encode(files []File) ([]byte, error) {
	ordered, err := canonicalFiles(files)
	if err != nil {
		return nil, err
	}

	var encoded bytes.Buffer
	encoded.WriteString(sourceDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeUint64(&encoded, uint64(len(ordered)))
	for _, file := range ordered {
		writeBytes(&encoded, []byte(file.Path))
		writeBytes(&encoded, file.Bytes)
	}
	return encoded.Bytes(), nil
}

// Compute validates and hashes one exact package source graph.
func Compute(files []File) (Digest, error) {
	ordered, err := canonicalFiles(files)
	if err != nil {
		return Digest{}, err
	}
	digest := sha256.New()
	writeIdentityHeader(digest, sourceDomain)
	writeHashUint64(digest, uint64(len(ordered)))
	for _, file := range ordered {
		writeHashUint64(digest, uint64(len(file.Path)))
		_, _ = digest.Write([]byte(file.Path))
		writeHashUint64(digest, uint64(len(file.Bytes)))
		_, _ = digest.Write(file.Bytes)
	}
	return Digest(digestValue(digest)), nil
}

func canonicalFiles(files []File) ([]File, error) {
	ordered := slices.Clone(files)
	if len(ordered) == 0 {
		return nil, fmt.Errorf("package source graph has no files")
	}
	slices.SortFunc(ordered, func(left, right File) int {
		return strings.Compare(left.Path, right.Path)
	})
	for index, file := range ordered {
		if err := validatePath(file.Path); err != nil {
			return nil, err
		}
		if index > 0 && ordered[index-1].Path == file.Path {
			return nil, fmt.Errorf("package source graph contains duplicate path %q", file.Path)
		}
	}
	return ordered, nil
}

func validatePath(value string) error {
	if value == "" || !utf8.ValidString(value) || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return fmt.Errorf("invalid canonical package source path %q", value)
	}
	if clean := pathpkg.Clean(value); clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("invalid canonical package source path %q", value)
	}
	if pathpkg.Ext(value) != ".fy" {
		return fmt.Errorf("package source path %q does not end in .fy", value)
	}
	return nil
}

func writeBytes(buffer *bytes.Buffer, value []byte) {
	writeUint64(buffer, uint64(len(value)))
	buffer.Write(value)
}

func writeUint64(buffer *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buffer.Write(raw[:])
}
