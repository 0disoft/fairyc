package evidencebundle

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/attributedtext"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	BundleVersionV2     = "fairyc.atlas-evidence/v2"
	BundleEncodingV2    = "fairyc.atlas-evidence-json/v2"
	ChecksumAlgorithmV2 = "fairyc.atlas-evidence-checksum/v2-sha256"
	ChecksumPrefixV2    = "fyevidence2-sha256-"
)

type ChecksumV2 Checksum

func (checksum ChecksumV2) String() string { return ChecksumPrefixV2 + hex.EncodeToString(checksum[:]) }

func ParseChecksumV2(value string) (ChecksumV2, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV2) {
		return ChecksumV2{}, fmt.Errorf("Atlas Evidence Bundle v2 checksum has invalid prefix")
	}
	digest, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV2))
	return ChecksumV2(digest), err
}

// EncodeShowV2 composes Context Pack v3 with two independently attributed text v0 documents.
// Aggregate fields and cleanup observations never acquire authority from source prose.
func EncodeShowV2(workspace workspaceloader.Workspace, index *atlasmap.QueryIndex, snapshot semanticsnapshot.Snapshot, symbolID string, request atlasmap.RelationRequest, textBudget attributedtext.Budget, limits Limits) ([]byte, error) {
	return encodeShow(workspace, index, snapshot, symbolID, request, textBudget, limits, 2)
}
