package attributedtext

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"
)

const (
	ChecksumAlgorithm = "fairyc.attributed-text-fragment/v0-sha256"
	ChecksumPrefix    = "fytext0-sha256-"
	checksumDomain    = "fairyc.attributed-text-fragment"
)

type Checksum [sha256.Size]byte

func (checksum Checksum) String() string {
	return ChecksumPrefix + hex.EncodeToString(checksum[:])
}

func ParseChecksum(value string) (Checksum, error) {
	var result Checksum
	if !strings.HasPrefix(value, ChecksumPrefix) {
		return result, fmt.Errorf("Atlas attributed-text checksum has invalid prefix")
	}
	digestText := strings.TrimPrefix(value, ChecksumPrefix)
	if len(digestText) != sha256.Size*2 || digestText != strings.ToLower(digestText) {
		return result, fmt.Errorf("Atlas attributed-text checksum must contain %d lowercase hexadecimal digits", sha256.Size*2)
	}
	decoded, err := hex.DecodeString(digestText)
	if err != nil {
		return result, fmt.Errorf("decode Atlas attributed-text checksum: %w", err)
	}
	copy(result[:], decoded)
	return result, nil
}

func computeChecksum(value View) Checksum {
	digest := sha256.New()
	_, _ = digest.Write([]byte(checksumDomain))
	_, _ = digest.Write([]byte{0, 0})
	for _, text := range []string{
		value.Policy, string(value.Kind), string(value.Authority), string(value.Origin),
		value.SymbolVisible, value.SymbolFull, value.PackageUID, value.PackageInstance,
		value.PackageCoordinate, value.PackageSource, value.Path, string(value.FileKind),
		value.FileIdentity, value.LineMapIdentity, string(value.Format),
	} {
		writeChecksumBytes(digest, []byte(text))
	}
	writeChecksumUint64(digest, value.StartByte)
	writeChecksumUint64(digest, value.EndByte)
	writeChecksumBytes(digest, []byte(value.Text))
	var result Checksum
	copy(result[:], digest.Sum(nil))
	return result
}

func writeChecksumBytes(destination hash.Hash, value []byte) {
	writeChecksumUint64(destination, uint64(len(value)))
	_, _ = destination.Write(value)
}

func writeChecksumUint64(destination hash.Hash, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	_, _ = destination.Write(raw[:])
}
