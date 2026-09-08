package contextpack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const (
	PackVersionV3       = "fairyc.context-pack/v3"
	PackEncodingV3      = "fairyc.context-pack-json/v3"
	ChecksumAlgorithmV3 = "fairyc.context-pack-checksum/v3-sha256"
	ChecksumPrefixV3    = "fyctx3-sha256-"
)

type ChecksumV3 [sha256.Size]byte

func (checksum ChecksumV3) String() string { return ChecksumPrefixV3 + hex.EncodeToString(checksum[:]) }

func ParseChecksumV3(value string) (ChecksumV3, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV3) {
		return ChecksumV3{}, fmt.Errorf("Context Pack v3 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV3))
	return ChecksumV3(digest), err
}

// EncodeOverviewV3 preserves original aggregate counts from an exact-current v3 map.
func EncodeOverviewV3(index *atlasmap.QueryIndex, budget atlasmap.QueryBudget, maxBytes uint64) ([]byte, error) {
	return encodeOverview(index, budget, maxBytes, 3)
}

// EncodeShowV3 keeps aggregate fields and cleanup observations separate from functions.
func EncodeShowV3(index *atlasmap.QueryIndex, symbolID string, request atlasmap.RelationRequest, maxBytes uint64) ([]byte, error) {
	return encodeShow(index, symbolID, request, maxBytes, 3)
}

// EncodeAroundV3 describes an aggregate declaration neighborhood, not drop execution order.
func EncodeAroundV3(index *atlasmap.QueryIndex, symbolID string, request atlasmap.RelationRequest, maxBytes uint64) ([]byte, error) {
	return encodeAround(index, symbolID, request, maxBytes, 3)
}
