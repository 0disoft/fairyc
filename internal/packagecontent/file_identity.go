package packagecontent

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

const (
	FileDigestEncoding = "fairyc.source-file/v1"
	FileDigestPrefix   = "fyfile1-sha256-"
	LineMapEncoding    = "fairyc.source-line-map/v1"
	LineMapPrefix      = "fylm1-sha256-"
)

const (
	fileDigestDomain = "fairyc.source-file"
	lineMapDomain    = "fairyc.source-line-map"
)

// FileDigest identifies exact source bytes without including a host or package-relative path.
type FileDigest [sha256.Size]byte

func (digest FileDigest) String() string {
	return FileDigestPrefix + hex.EncodeToString(digest[:])
}

func (digest FileDigest) MarshalText() ([]byte, error) {
	return []byte(digest.String()), nil
}

func (digest *FileDigest) UnmarshalText(text []byte) error {
	parsed, err := ParseFileDigest(string(text))
	if err != nil {
		return err
	}
	*digest = parsed
	return nil
}

// LineMapID identifies the byte length and every zero-based line-start offset. It intentionally
// ignores non-newline content so a consumer can distinguish content invalidation from line-map
// invalidation while requiring both identities before trusting a source span.
type LineMapID [sha256.Size]byte

func (identity LineMapID) String() string {
	return LineMapPrefix + hex.EncodeToString(identity[:])
}

func (identity LineMapID) MarshalText() ([]byte, error) {
	return []byte(identity.String()), nil
}

func (identity *LineMapID) UnmarshalText(text []byte) error {
	parsed, err := ParseLineMapID(string(text))
	if err != nil {
		return err
	}
	*identity = parsed
	return nil
}

func ParseFileDigest(value string) (FileDigest, error) {
	var result FileDigest
	if err := parseIdentityText(value, FileDigestPrefix, result[:], "source file digest"); err != nil {
		return FileDigest{}, err
	}
	return result, nil
}

func ParseLineMapID(value string) (LineMapID, error) {
	var result LineMapID
	if err := parseIdentityText(value, LineMapPrefix, result[:], "source line-map identity"); err != nil {
		return LineMapID{}, err
	}
	return result, nil
}

func parseIdentityText(value, prefix string, destination []byte, label string) error {
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

// EncodeFileDigestInput returns the exact v1 bytes hashed for FileDigest. It is intended for
// fixtures and compatibility tooling; ComputeFileDigest streams the same sequence without copying
// the source body.
func EncodeFileDigestInput(contents []byte) []byte {
	var encoded bytes.Buffer
	encoded.Grow(len(fileDigestDomain) + len(contents) + 16)
	encoded.WriteString(fileDigestDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeUint64(&encoded, uint64(len(contents)))
	encoded.Write(contents)
	return encoded.Bytes()
}

func ComputeFileDigest(contents []byte) FileDigest {
	digest := sha256.New()
	writeIdentityHeader(digest, fileDigestDomain)
	writeHashUint64(digest, uint64(len(contents)))
	_, _ = digest.Write(contents)
	return FileDigest(digestValue(digest))
}

// ComputeFileIdentities computes both map identities without retaining source or allocating a
// line-offset table. The line-map pass trades another bounded scan for constant auxiliary memory.
func ComputeFileIdentities(contents []byte) (FileDigest, LineMapID) {
	return ComputeFileDigest(contents), computeLineMapID(contents)
}

// EncodeLineMapInput returns the exact v1 byte sequence: file byte length, line-start count, then
// every zero-based line start. Offset zero is always present; a final LF adds an EOF line start.
func EncodeLineMapInput(contents []byte) []byte {
	lineCount := 1 + bytes.Count(contents, []byte{'\n'})
	var encoded bytes.Buffer
	encoded.Grow(len(lineMapDomain) + 24 + lineCount*8)
	encoded.WriteString(lineMapDomain)
	encoded.WriteByte(0)
	encoded.WriteByte(1)
	writeUint64(&encoded, uint64(len(contents)))
	writeUint64(&encoded, uint64(lineCount))
	writeUint64(&encoded, 0)
	for index, value := range contents {
		if value == '\n' {
			writeUint64(&encoded, uint64(index+1))
		}
	}
	return encoded.Bytes()
}

func ComputeLineMapID(contents []byte) LineMapID {
	return computeLineMapID(contents)
}

func computeLineMapID(contents []byte) LineMapID {
	digest := sha256.New()
	writeIdentityHeader(digest, lineMapDomain)
	writeHashUint64(digest, uint64(len(contents)))
	lineCount := uint64(1)
	for _, value := range contents {
		if value == '\n' {
			lineCount++
		}
	}
	writeHashUint64(digest, lineCount)
	writeHashUint64(digest, 0)
	for index, value := range contents {
		if value == '\n' {
			writeHashUint64(digest, uint64(index+1))
		}
	}
	return LineMapID(digestValue(digest))
}

func writeIdentityHeader(destination hash.Hash, domain string) {
	_, _ = destination.Write([]byte(domain))
	_, _ = destination.Write([]byte{0, 1})
}

func writeHashUint64(destination hash.Hash, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	_, _ = destination.Write(raw[:])
}

func digestValue(digest hash.Hash) [sha256.Size]byte {
	var result [sha256.Size]byte
	copy(result[:], digest.Sum(nil))
	return result
}
