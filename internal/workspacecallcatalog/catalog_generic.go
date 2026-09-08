package workspacecallcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const (
	CatalogVersionV1    = "fairyc.workspace-call-catalog/v1"
	CatalogEncodingV1   = "fairyc.workspace-call-catalog-json/v1"
	ChecksumAlgorithmV1 = "fairyc.workspace-call-catalog-checksum/v1-sha256"
	ChecksumPrefixV1    = "fywcall1-sha256-"
)

type ChecksumV1 [sha256.Size]byte

func (checksum ChecksumV1) String() string {
	return ChecksumPrefixV1 + hex.EncodeToString(checksum[:])
}

func ParseChecksumV1(value string) (ChecksumV1, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV1) {
		return ChecksumV1{}, fmt.Errorf("workspace call catalog v1 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV1))
	return ChecksumV1(digest), err
}

// EncodeV1 emits concrete implementation endpoints from an exact-current v2 workspace join.
// Selection remains declaration-based; neither the format nor its IDs are execution authority.
func EncodeV1(index *atlasmap.WorkspaceCallIndex, packageInstance, symbolID string, direction atlasmap.QueryDirection, maxBytes uint64) ([]byte, error) {
	return encode(index, packageInstance, symbolID, direction, maxBytes, 1)
}
