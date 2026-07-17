// Package source owns original Fairy C bytes, identities, and canonical byte spans.
package source

import (
	"errors"
	"path"
	"strings"
)

// File is one source-root-relative Fairy C input. Bytes are never normalized.
type File struct {
	RelativePath string
	Bytes        []byte
}

// NewFile copies bytes so a parse observes one immutable source version.
func NewFile(relativePath string, bytes []byte) File {
	return File{
		RelativePath: strings.ReplaceAll(relativePath, "\\", "/"),
		Bytes:        append([]byte(nil), bytes...),
	}
}

// ExpectedModule derives the Stage 0 module identity from a source-root-relative path.
func (f File) ExpectedModule() (string, error) {
	clean := path.Clean(f.RelativePath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return "", errors.New("source path escapes its root")
	}
	if path.Ext(clean) != ".fy" {
		return "", errors.New("source path must end in .fy")
	}
	withoutExtension := strings.TrimSuffix(clean, ".fy")
	segments := strings.Split(withoutExtension, "/")
	for _, segment := range segments {
		if !ValidIdentifier(segment) {
			return "", errors.New("source path segment is not a Stage 0 identifier")
		}
	}
	return strings.Join(segments, "::"), nil
}

// ValidIdentifier implements the ASCII-only Stage 0 identifier rule.
func ValidIdentifier(value string) bool {
	if value == "" || !isIdentifierStart(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !isIdentifierContinue(value[i]) {
			return false
		}
	}
	return true
}

func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isIdentifierContinue(value byte) bool {
	return isIdentifierStart(value) || value >= '0' && value <= '9'
}
