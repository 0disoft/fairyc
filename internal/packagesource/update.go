package packagesource

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type PackageChange struct {
	Before Pin `json:"before"`
	After  Pin `json:"after"`
}
type UpdateProposal struct {
	Schema            string             `json:"schema"`
	CurrentPlanSHA256 string             `json:"currentPlanSha256"`
	Resolution        RegistryResolution `json:"resolution"`
	Changes           []PackageChange    `json:"changes"`
}

// ProposeRegistryUpdate is read-only and preserves the complete package identity
// set. A same-version content substitution is not an update.
func ProposeRegistryUpdate(current Plan, verified VerifiedCatalog, requests []byte) (UpdateProposal, error) {
	if err := Validate(current); err != nil {
		return UpdateProposal{}, err
	}
	resolution, err := ResolveRegistry(verified, requests)
	if err != nil {
		return UpdateProposal{}, err
	}
	if len(current.Packages) != len(resolution.SourcePlan.Packages) {
		return UpdateProposal{}, fmt.Errorf("updates cannot add or remove package identities")
	}
	before := map[string]Pin{}
	for _, pin := range current.Packages {
		before[pin.Path] = pin
	}
	changes := []PackageChange{}
	for _, next := range resolution.SourcePlan.Packages {
		old, found := before[next.Path]
		if !found || old.PackageUID != next.PackageUID {
			return UpdateProposal{}, fmt.Errorf("update package identity changed")
		}
		oldVersion, err := parseStableVersion(old.Version)
		if err != nil {
			return UpdateProposal{}, err
		}
		nextVersion, _ := parseStableVersion(next.Version)
		if compareStable(nextVersion, oldVersion) < 0 {
			return UpdateProposal{}, fmt.Errorf("package downgrade refused")
		}
		if old.Version == next.Version && old.SHA256 != next.SHA256 {
			return UpdateProposal{}, fmt.Errorf("existing exact version content changed")
		}
		if old != next {
			changes = append(changes, PackageChange{Before: old, After: next})
		}
	}
	canonical := Plan{Schema: current.Schema, Packages: slices.Clone(current.Packages)}
	slices.SortFunc(canonical.Packages, func(a, b Pin) int { return strings.Compare(a.Path, b.Path) })
	data, err := json.Marshal(canonical)
	if err != nil || len(data) > MaxPlanBytes {
		return UpdateProposal{}, fmt.Errorf("current plan byte bound")
	}
	return UpdateProposal{Schema: "fairyc.package-update-proposal/v1", CurrentPlanSHA256: hash(data), Resolution: resolution, Changes: changes}, nil
}
