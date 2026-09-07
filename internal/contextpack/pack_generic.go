package contextpack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

const (
	PackVersionV2       = "fairyc.context-pack/v2"
	PackEncodingV2      = "fairyc.context-pack-json/v2"
	ChecksumAlgorithmV2 = "fairyc.context-pack-checksum/v2-sha256"
	ChecksumPrefixV2    = "fyctx2-sha256-"
)

type ChecksumV2 [sha256.Size]byte

func (checksum ChecksumV2) String() string { return ChecksumPrefixV2 + hex.EncodeToString(checksum[:]) }

func ParseChecksumV2(value string) (ChecksumV2, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV2) {
		return ChecksumV2{}, fmt.Errorf("Context Pack v2 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV2))
	return ChecksumV2(digest), err
}

// EncodeOverviewV2 preserves generic-aware package counts and compatibility in a v2 pack.
func EncodeOverviewV2(index *atlasmap.QueryIndex, budget atlasmap.QueryBudget, maxBytes uint64) ([]byte, error) {
	return encodeOverview(index, budget, maxBytes, true)
}

// EncodeShowV2 keeps source declarations separate from concrete implementation observations.
func EncodeShowV2(index *atlasmap.QueryIndex, symbolID string, request atlasmap.RelationRequest, maxBytes uint64) ([]byte, error) {
	return encodeShow(index, symbolID, request, maxBytes, true)
}

// EncodeAroundV2 describes a declaration neighborhood, not a concrete execution path.
func EncodeAroundV2(index *atlasmap.QueryIndex, symbolID string, request atlasmap.RelationRequest, maxBytes uint64) ([]byte, error) {
	return encodeAround(index, symbolID, request, maxBytes, true)
}
