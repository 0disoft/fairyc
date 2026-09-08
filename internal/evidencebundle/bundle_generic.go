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
	BundleVersionV1     = "fairyc.atlas-evidence/v1"
	BundleEncodingV1    = "fairyc.atlas-evidence-json/v1"
	ChecksumAlgorithmV1 = "fairyc.atlas-evidence-checksum/v1-sha256"
	ChecksumPrefixV1    = "fyevidence1-sha256-"
)

type ChecksumV1 Checksum

func (checksum ChecksumV1) String() string {
	return ChecksumPrefixV1 + hex.EncodeToString(checksum[:])
}

func ParseChecksumV1(value string) (ChecksumV1, error) {
	if !strings.HasPrefix(value, ChecksumPrefixV1) {
		return ChecksumV1{}, fmt.Errorf("Atlas Evidence Bundle v1 checksum has invalid prefix")
	}
	parsed, err := ParseChecksum(ChecksumPrefix + strings.TrimPrefix(value, ChecksumPrefixV1))
	return ChecksumV1(parsed), err
}

// EncodeShowV1 composes Context Pack v2 and unchanged attributed-text v0 documents.
// It accepts only exact-current v2 queries and retains each child's authority and limits.
func EncodeShowV1(
	workspace workspaceloader.Workspace,
	index *atlasmap.QueryIndex,
	snapshot semanticsnapshot.Snapshot,
	symbolID string,
	request atlasmap.RelationRequest,
	textBudget attributedtext.Budget,
	limits Limits,
) ([]byte, error) {
	return encodeShow(workspace, index, snapshot, symbolID, request, textBudget, limits, 1)
}
