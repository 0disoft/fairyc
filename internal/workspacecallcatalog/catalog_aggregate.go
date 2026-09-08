package workspacecallcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const (
	CatalogVersionV2    = "fairyc.workspace-call-catalog/v2"
	CatalogEncodingV2   = "fairyc.workspace-call-catalog-json/v2"
	ChecksumAlgorithmV2 = "fairyc.workspace-call-catalog-checksum/v2-sha256"
	ChecksumPrefixV2    = "fywcall2-sha256-"
)

type ChecksumV2 [sha256.Size]byte

func (checksum ChecksumV2) String() string {
	return ChecksumPrefixV2 + hex.EncodeToString(checksum[:])
}

func ParseChecksumV2(value string) (ChecksumV2, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV2) {
		return ChecksumV2{}, fmt.Errorf("workspace call catalog v2 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV2))
	return ChecksumV2(digest), err
}

// EncodeV2 emits cleanup-owned calls from an exact-current v3 workspace join.
// Selection remains declaration-based and does not model implicit drop scheduling.
func EncodeV2(index *atlasmap.WorkspaceCallIndex, packageInstance, symbolID string, direction atlasmap.QueryDirection, maxBytes uint64) ([]byte, error) {
	return encode(index, packageInstance, symbolID, direction, maxBytes, 2)
}
