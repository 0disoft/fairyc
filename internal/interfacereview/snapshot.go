// Package interfacereview provides opt-in developer review, not ABI compatibility admission.
package interfacereview

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

type State string

const (
	Exact   State = "exact"
	Bounded State = "bounded"
	Partial State = "partial"
	Unknown State = "unknown"
)

type Issue struct {
	Module string `json:"module,omitempty"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type Package struct {
	UID      string   `json:"uid"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Language string   `json:"language"`
	Features []string `json:"features"`
}

type Snapshot struct {
	State          State                       `json:"state"`
	Package        Package                     `json:"package"`
	SourceDigest   string                      `json:"sourceDigest,omitempty"`
	ProjectionHash string                      `json:"projectionHash,omitempty"`
	Descriptor     *moduleinterface.Descriptor `json:"-"`
	Issues         []Issue                     `json:"issues"`
}

// Capture checks the root-reachable workspace using the existing read-only compiler
// entry. Only the root package's ordinary v5 interface is compared, not its dependencies.
func Capture(ctx context.Context, root string) Snapshot {
	result := Snapshot{State: Unknown, Issues: []Issue{}}
	fail := func(reason string, err error) Snapshot {
		result.Issues = append(result.Issues, Issue{Reason: reason, Detail: err.Error()})
		return result
	}
	workspace, err := workspaceloader.LoadContext(ctx, root)
	if err != nil {
		return fail("workspace-unavailable", err)
	}
	id, err := workspace.RootPackageID()
	if err != nil {
		return fail("root-unavailable", err)
	}
	packages, err := workspace.SemanticPackages()
	if err != nil {
		return fail("package-metadata-unavailable", err)
	}
	var owner workspaceloader.SemanticPackage
	for _, candidate := range packages {
		if candidate.ID == id {
			owner = candidate
			break
		}
	}
	if owner.PackageUID == "" {
		return fail("root-unavailable", fmt.Errorf("root package metadata missing"))
	}
	result.Package = Package{owner.PackageUID, owner.Name, owner.Version, owner.Language, slices.Clone(owner.Features)}
	result.SourceDigest = owner.Source.String()
	loaded, ok := workspace.Package(owner.Path)
	if !ok {
		return fail("root-unavailable", fmt.Errorf("root catalog missing"))
	}
	blocked := map[string]bool{}
	for _, file := range loaded.Sources {
		module := file.Module
		if module == nil {
			return fail("source-unavailable", fmt.Errorf("missing parsed module"))
		}
		generic := false
		for _, item := range module.Functions {
			generic = generic || len(item.TypeParameters) != 0
		}
		for _, item := range module.Structs {
			generic = generic || len(item.TypeParameters) != 0
		}
		for _, item := range module.Resources {
			generic = generic || len(item.TypeParameters) != 0
		}
		if generic && !blocked[module.Path] {
			blocked[module.Path] = true
			result.Issues = append(result.Issues, Issue{Module: module.Path, Reason: "generic-module-outside-ordinary-v5-review"})
		}
	}
	checked, err := workspacecompiler.CheckContext(ctx, workspace)
	if err != nil {
		return fail("compiler-unavailable", err)
	}
	if !checked.Verified {
		return fail("source-not-verified", fmt.Errorf("diagnostics=%d unsupported=%d; no interface conclusion", len(checked.Semantic.Diagnostics), len(checked.Semantic.Unsupported)))
	}
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return fail("coordinates-unavailable", err)
	}
	// Omit entire blocked modules, including generated specializations. This keeps
	// an unavailable generic body from appearing as a removed ordinary declaration.
	typed := checked.Semantic.HIR
	modules := make([]hir.Module, 0, len(typed.Modules))
	for _, module := range typed.Modules {
		if module.Package == id && !blocked[module.Path] {
			modules = append(modules, module)
		}
	}
	if len(modules) == 0 {
		return fail("no-comparable-modules", fmt.Errorf("no verified ordinary root modules remain"))
	}
	typed.Modules = modules
	descriptor, err := moduleinterface.Extract(typed, id, coordinates)
	if err != nil {
		return fail("interface-unavailable", err)
	}
	hash, err := moduleinterface.Compute(descriptor)
	if err != nil {
		return fail("interface-unavailable", err)
	}
	result.Descriptor, result.ProjectionHash = &descriptor, hash.String()
	result.State = Bounded
	if len(result.Issues) != 0 {
		result.State = Partial
	}
	slices.SortFunc(result.Issues, func(a, b Issue) int { return strings.Compare(a.Module+"\x00"+a.Reason, b.Module+"\x00"+b.Reason) })
	return result
}
