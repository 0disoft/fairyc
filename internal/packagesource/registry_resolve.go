package packagesource

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const RegistryRequestsSchema = "fairyc.registry-requests/v1"

type RegistryRequest struct {
	Path       string `json:"path"`
	PackageUID string `json:"packageUid"`
	Version    string `json:"version"`
}
type RegistryRequests struct {
	Schema   string            `json:"schema"`
	Packages []RegistryRequest `json:"packages"`
}
type RegistryResolution struct {
	Schema        string `json:"schema"`
	CatalogSHA256 string `json:"catalogSha256"`
	Sequence      uint64 `json:"sequence"`
	SourcePlan    Plan   `json:"sourcePlan"`
}

// ResolveRegistry emits exact pins only; it neither downloads nor changes a manifest.
func ResolveRegistry(verified VerifiedCatalog, data []byte) (RegistryResolution, error) {
	if verified.catalog.Schema != RegistryCatalogSchema || verified.digest == "" {
		return RegistryResolution{}, fmt.Errorf("verified catalog required")
	}
	fields, err := boundedObject(data, MaxPlanBytes, "schema", "packages")
	if err != nil {
		return RegistryResolution{}, err
	}
	var requests RegistryRequests
	if err = json.Unmarshal(data, &requests); err != nil {
		return RegistryResolution{}, err
	}
	if requests.Schema != RegistryRequestsSchema || len(requests.Packages) == 0 || len(requests.Packages) > MaxPackages {
		return RegistryResolution{}, fmt.Errorf("invalid registry request header or bound")
	}
	var entries []json.RawMessage
	if err = json.Unmarshal(fields["packages"], &entries); err != nil {
		return RegistryResolution{}, err
	}
	plan := Plan{Schema: Schema}
	for i, entry := range entries {
		if _, err = object(entry, "path", "packageUid", "version"); err != nil {
			return RegistryResolution{}, err
		}
		request := requests.Packages[i]
		constraint, err := parseVersionRange(request.Version)
		if err != nil {
			return RegistryResolution{}, err
		}
		var best *RegistryRelease
		var bestVersion stableVersion
		for _, release := range verified.catalog.Packages {
			if release.PackageUID != request.PackageUID {
				continue
			}
			version, _ := parseStableVersion(release.Version)
			if constraint.contains(version) && (best == nil || compareStable(version, bestVersion) > 0) {
				copy := release
				best = &copy
				bestVersion = version
			}
		}
		if best == nil {
			return RegistryResolution{}, fmt.Errorf("no catalog version satisfies %s %s", request.PackageUID, request.Version)
		}
		plan.Packages = append(plan.Packages, Pin{Path: request.Path, PackageUID: request.PackageUID, Version: best.Version, URL: best.URL, SHA256: best.SHA256})
	}
	slices.SortFunc(plan.Packages, func(a, b Pin) int { return strings.Compare(a.Path, b.Path) })
	if err = Validate(plan); err != nil {
		return RegistryResolution{}, err
	}
	encoded, err := json.Marshal(plan)
	if err != nil || len(encoded) > MaxPlanBytes {
		return RegistryResolution{}, fmt.Errorf("resolved source plan byte bound")
	}
	return RegistryResolution{Schema: "fairyc.registry-resolution/v1", CatalogSHA256: verified.digest, Sequence: verified.catalog.Sequence, SourcePlan: plan}, nil
}
