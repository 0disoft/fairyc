// Package packagesource owns explicitly pinned remote source preparation, not implicit build networking.
package packagesource

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const (
	FileName     = "fairy.sources.json"
	Schema       = "fairyc.sources/v1"
	MaxPlanBytes = 64 << 10
	MaxPackages  = 32
	CachePath    = "build/package-sources"
)

type Pin struct {
	Path       string `json:"path"`
	PackageUID string `json:"packageUid"`
	Version    string `json:"version"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
}

type Plan struct {
	Schema   string `json:"schema"`
	Packages []Pin  `json:"packages"`
}

// object rejects duplicate, omitted, unknown and noncanonical JSON field spellings.
func object(data []byte, fields ...string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("expected source object")
	}
	result := map[string]json.RawMessage{}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return nil, err
		}
		key, ok := t.(string)
		if !ok || !slices.Contains(fields, key) {
			return nil, fmt.Errorf("unknown source field")
		}
		if _, ok := result[key]; ok {
			return nil, fmt.Errorf("duplicate source field %s", key)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		result[key] = raw
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing source JSON")
	}
	if len(result) != len(fields) {
		return nil, fmt.Errorf("missing source field")
	}
	return result, nil
}

func Parse(data []byte) (Plan, error) {
	if len(data) > MaxPlanBytes || !utf8.Valid(data) {
		return Plan{}, fmt.Errorf("source plan exceeds byte limit or is not UTF-8")
	}
	fields, err := object(data, "schema", "packages")
	if err != nil {
		return Plan{}, err
	}
	var plan Plan
	if err := json.Unmarshal(fields["schema"], &plan.Schema); err != nil {
		return Plan{}, err
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(fields["packages"], &entries); err != nil {
		return Plan{}, err
	}
	if len(entries) == 0 || len(entries) > MaxPackages {
		return Plan{}, fmt.Errorf("source plan needs 1..%d packages", MaxPackages)
	}
	for _, entry := range entries {
		if _, err := object(entry, "path", "packageUid", "version", "url", "sha256"); err != nil {
			return Plan{}, err
		}
		var pin Pin
		if err := json.Unmarshal(entry, &pin); err != nil {
			return Plan{}, err
		}
		plan.Packages = append(plan.Packages, pin)
	}
	slices.SortFunc(plan.Packages, func(a, b Pin) int { return strings.Compare(a.Path, b.Path) })
	return plan, Validate(plan)
}

func Validate(plan Plan) error {
	if plan.Schema != Schema || len(plan.Packages) == 0 || len(plan.Packages) > MaxPackages {
		return fmt.Errorf("invalid source plan header")
	}
	paths, uids := map[string]bool{}, map[string]bool{}
	for _, pin := range plan.Packages {
		parts := strings.Split(pin.Path, "/")
		if len(parts) != 2 || parts[0] != "vendor" || !projectmanifest.ValidPackageName(parts[1]) || !projectmanifest.ValidPortablePathSegment(parts[1]) {
			return fmt.Errorf("source path must be vendor/<portable-package-name>")
		}
		if paths[pin.Path] || uids[pin.PackageUID] {
			return fmt.Errorf("duplicate source path or package UID")
		}
		paths[pin.Path], uids[pin.PackageUID] = true, true
		if !projectmanifest.ValidPackageUID(pin.PackageUID) || !projectmanifest.ValidExactVersion(pin.Version) {
			return fmt.Errorf("invalid source package UID or exact version")
		}
		if err := validateURL(pin.URL); err != nil {
			return err
		}
		if len(pin.SHA256) != 64 || pin.SHA256 != strings.ToLower(pin.SHA256) {
			return fmt.Errorf("source SHA-256 must be 64 lowercase hex digits")
		}
		if _, err := hex.DecodeString(pin.SHA256); err != nil {
			return fmt.Errorf("invalid source SHA-256")
		}
	}
	return nil
}

func validateURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || !strings.HasPrefix(u.Path, "/") || strings.ContainsAny(value, "\\\r\n\t #") {
		return fmt.Errorf("source URL must be absolute HTTPS without credentials, query or fragment")
	}
	if u.Port() != "" {
		n, err := strconv.Atoi(u.Port())
		if err != nil || n < 1 || n > 65535 {
			return fmt.Errorf("invalid source HTTPS port")
		}
	}
	return nil
}

// Bind admits only explicitly catalogued members and preserves exact dependency versions.
func Bind(plan Plan, root projectmanifest.Manifest) error {
	if err := Validate(plan); err != nil {
		return err
	}
	for _, pin := range plan.Packages {
		if !slices.Contains(root.Workspace.Members, pin.Path) || pin.PackageUID == root.Package.UID {
			return fmt.Errorf("source package must name an explicit non-root workspace member")
		}
		if overlaps(pin.Path, root.Package.Source) || overlaps(pin.Path, CachePath) {
			return fmt.Errorf("source destination overlaps root source or cache")
		}
		for _, member := range root.Workspace.Members {
			if member != pin.Path && overlaps(member, pin.Path) {
				return fmt.Errorf("source destination overlaps another member")
			}
		}
		for _, dep := range root.Dependencies {
			if dep.Path == pin.Path && dep.Version != pin.Version {
				return fmt.Errorf("source pin and dependency versions differ")
			}
		}
	}
	if overlaps(root.Package.Source, CachePath) {
		return fmt.Errorf("source cache overlaps root source")
	}
	for _, member := range root.Workspace.Members {
		if overlaps(member, CachePath) {
			return fmt.Errorf("source cache overlaps member")
		}
	}
	return nil
}

func overlaps(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
