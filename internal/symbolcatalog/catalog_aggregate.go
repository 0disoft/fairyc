package symbolcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const (
	CatalogVersionV3    = "fairyc.symbol-catalog/v3"
	CatalogEncodingV3   = "fairyc.symbol-catalog-json/v3"
	ChecksumAlgorithmV3 = "fairyc.symbol-catalog-checksum/v3-sha256"
	ChecksumPrefixV3    = "fyscat3-sha256-"
)

type ChecksumV3 [sha256.Size]byte

func (checksum ChecksumV3) String() string {
	return ChecksumPrefixV3 + hex.EncodeToString(checksum[:])
}

func ParseChecksumV3(value string) (ChecksumV3, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV3) {
		return ChecksumV3{}, fmt.Errorf("symbol catalog v3 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV3))
	return ChecksumV3(digest), err
}

// EncodeV3 lists original declarations, including aggregates, from an exact-current v3 map.
// Concrete implementations and cleanup bodies do not become additional source symbols.
func EncodeV3(index *atlasmap.QueryIndex, request atlasmap.SymbolCatalogRequest, maxBytes uint64) ([]byte, error) {
	return encode(index, request, maxBytes, 3)
}
