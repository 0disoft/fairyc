package symbolcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const (
	CatalogVersionV2    = "fairyc.symbol-catalog/v2"
	CatalogEncodingV2   = "fairyc.symbol-catalog-json/v2"
	ChecksumAlgorithmV2 = "fairyc.symbol-catalog-checksum/v2-sha256"
	ChecksumPrefixV2    = "fyscat2-sha256-"
)

type ChecksumV2 [sha256.Size]byte

func (checksum ChecksumV2) String() string {
	return ChecksumPrefixV2 + hex.EncodeToString(checksum[:])
}

func ParseChecksumV2(value string) (ChecksumV2, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV2) {
		return ChecksumV2{}, fmt.Errorf("symbol catalog v2 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV2))
	return ChecksumV2(digest), err
}

// EncodeV2 catalogs original declaration identities from a generic-aware v2 map.
// Specializations do not become source declarations or additional catalog entries.
func EncodeV2(index *atlasmap.QueryIndex, request atlasmap.SymbolCatalogRequest, maxBytes uint64) ([]byte, error) {
	return encode(index, request, maxBytes, 2)
}
