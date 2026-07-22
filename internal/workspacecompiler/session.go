// Package workspacecompiler coordinates package-aware semantic analysis and process-local MIR reuse.
package workspacecompiler

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const (
	OutcomeNotEligible = "not-eligible"
	OutcomeFallback    = "fallback"
	OutcomeAllHit      = "all-hit"

	compilerProcessEncoding = "fairyc.compiler-process/0:"
	backendIdentity         = "verified-mir"
	toolchainIdentity       = "not-applicable:process-local-mir-v0"
)

// Configuration supplies the explicit target and mandatory cache bounds. The first cache reuses
// verified MIR only, so native toolchain and optimization identities are deliberately inapplicable.
type Configuration struct {
	Target           target.ID
	Cache            packagecache.Limits
	MaxArtifactBytes uint64
}

// Report describes one cache decision without treating raw byte-store hits as executable reuse.
type Report struct {
	Outcome            string
	Packages           int
	Hits               int
	Misses             int
	Rejected           int
	Stored             int
	StoreSkipped       int
	Lowered            bool
	ReassemblyFallback bool
}

// Result contains the ordinary compiler result plus the exact process-cache decision that led to it.
type Result struct {
	Compiler compiler.Result
	Cache    Report
}

// Session owns one bounded, process-private cache. A Session can never read artifacts written by
// another process or by another Session instance.
type Session struct {
	cache           *packageartifact.Cache
	target          target.Spec
	standardLibrary packagegraph.InstanceID
	lower           func(compiler.Analysis) (compiler.Result, error)
}

// NewSession validates every compatibility input before creating the bounded cache.
func NewSession(configuration Configuration) (*Session, error) {
	targetSpec, found := target.Lookup(configuration.Target)
	if !found {
		return nil, fmt.Errorf("workspace compiler target %q is unknown", configuration.Target)
	}
	if err := target.Validate(targetSpec); err != nil {
		return nil, fmt.Errorf("workspace compiler target %q: %w", configuration.Target, err)
	}
	cache, err := packageartifact.NewCache(configuration.Cache, configuration.MaxArtifactBytes)
	if err != nil {
		return nil, err
	}
	standardLibrary, err := standardLibraryInstance()
	if err != nil {
		return nil, err
	}
	return &Session{
		cache: cache, target: targetSpec, standardLibrary: standardLibrary,
		lower: compiler.LowerAnalyzed,
	}, nil
}

// Check performs semantic analysis on every request. It skips MIR lowering only when every
// function-owning package is a verified cache hit and the complete artifact set reassembles.
func (session *Session) Check(workspace workspaceloader.Workspace) (Result, error) {
	if session == nil || session.cache == nil || session.lower == nil {
		return Result{}, fmt.Errorf("workspace compiler session is not initialized")
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return Result{}, err
	}
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		return Result{}, err
	}
	identities, err := workspace.PackageIdentities()
	if err != nil {
		return Result{}, err
	}
	analysis, err := compiler.AnalyzePackages(input)
	if err != nil {
		return Result{}, err
	}
	if !semanticReady(analysis.Semantic) {
		compiled, lowerErr := session.lower(analysis)
		return Result{Compiler: compiled, Cache: Report{Outcome: OutcomeNotEligible}}, lowerErr
	}

	interfaces, err := packageInterfaces(analysis.Semantic.HIR, metadata, identities)
	if err != nil {
		return Result{}, err
	}
	owners := functionOwners(analysis.Semantic.HIR)
	report := Report{Outcome: OutcomeFallback, Packages: len(owners)}
	if len(owners) == 0 {
		compiled, lowerErr := session.lower(analysis)
		report.Lowered = compiled.Verified
		return Result{Compiler: compiled, Cache: report}, lowerErr
	}
	keys, err := session.packageKeys(metadata, interfaces, owners)
	if err != nil {
		return Result{}, err
	}

	artifacts := make([]packageartifact.Artifact, 0, len(owners))
	allHit := true
	for _, packageID := range owners {
		lookup := session.cache.Get(keys[packageID], interfaces[packageID])
		switch lookup.Status {
		case packageartifact.LookupHit:
			report.Hits++
			artifacts = append(artifacts, lookup.Artifact)
		case packageartifact.LookupMiss:
			report.Misses++
			allHit = false
		case packageartifact.LookupRejected:
			report.Rejected++
			allHit = false
		default:
			return Result{}, fmt.Errorf("workspace compiler cache returned unknown status %q", lookup.Status)
		}
	}
	if allHit {
		program, reassemblyErr := packageartifact.Reassemble(artifacts)
		if reassemblyErr == nil {
			report.Outcome = OutcomeAllHit
			return Result{Compiler: compiler.Result{Semantic: analysis.Semantic, MIR: program, Verified: true}, Cache: report}, nil
		}
		report.ReassemblyFallback = true
	}

	compiled, err := session.lower(analysis)
	if err != nil {
		return Result{}, err
	}
	if !compiled.Verified {
		return Result{Compiler: compiled, Cache: report}, nil
	}
	report.Lowered = true
	extracted, err := packageartifact.ExtractAll(compiled.MIR, identities)
	if err != nil {
		return Result{}, fmt.Errorf("extract package MIR artifacts: %w", err)
	}
	byPackage := make(map[hir.PackageID]packageartifact.Artifact, len(extracted))
	for _, artifact := range extracted {
		if _, duplicate := byPackage[artifact.PackageID]; duplicate {
			return Result{}, fmt.Errorf("lowered MIR produced duplicate package artifact %d", artifact.PackageID)
		}
		byPackage[artifact.PackageID] = artifact
	}
	for _, packageID := range owners {
		artifact, found := byPackage[packageID]
		if !found {
			return Result{}, fmt.Errorf("lowered MIR has no artifact for function-owning package %d", packageID)
		}
		if err := session.cache.Put(keys[packageID], interfaces[packageID], artifact); err != nil {
			if errors.Is(err, packageartifact.ErrProcessArtifactTooLarge) || errors.Is(err, packagecache.ErrRecordTooLarge) {
				report.StoreSkipped++
				continue
			}
			return Result{}, fmt.Errorf("store package %d MIR artifact: %w", packageID, err)
		}
		report.Stored++
	}
	return Result{Compiler: compiled, Cache: report}, nil
}

// Stats returns the underlying verified-cache counters for measurement and diagnostics.
func (session *Session) Stats() packageartifact.CacheStats {
	if session == nil || session.cache == nil {
		return packageartifact.CacheStats{}
	}
	return session.cache.Stats()
}

func (session *Session) packageKeys(metadata []workspaceloader.SemanticPackage, interfaces map[hir.PackageID]moduleinterface.Hash, owners []hir.PackageID) (map[hir.PackageID]packagecache.Key, error) {
	byID := make(map[hir.PackageID]workspaceloader.SemanticPackage, len(metadata))
	for _, loaded := range metadata {
		byID[loaded.ID] = loaded
	}
	result := make(map[hir.PackageID]packagecache.Key, len(owners))
	for _, packageID := range owners {
		loaded, found := byID[packageID]
		if !found {
			return nil, fmt.Errorf("function-owning package %d has no workspace metadata", packageID)
		}
		dependencies := make([]packagecache.Dependency, 0, len(loaded.Dependencies))
		for _, dependencyID := range loaded.Dependencies {
			dependency, found := byID[dependencyID]
			if !found {
				return nil, fmt.Errorf("package %d dependency %d has no workspace metadata", packageID, dependencyID)
			}
			interfaceHash, found := interfaces[dependencyID]
			if !found {
				return nil, fmt.Errorf("package %d dependency %d has no public interface", packageID, dependencyID)
			}
			dependencies = append(dependencies, packagecache.Dependency{Instance: dependency.Instance, Interface: interfaceHash})
		}
		dependencies, err := packagecache.CanonicalizeDependencies(dependencies)
		if err != nil {
			return nil, err
		}
		key, err := packagecache.Compute(packagecache.Inputs{
			CompilerBuild:    compilerProcessEncoding + session.cache.Identity(),
			LanguageRevision: sdkstdlib.LanguageRevision, MIRVersion: mircodec.MIRVersion,
			TargetSchema: session.target.SchemaVersion, ABIProfile: abiProfile(session.target),
			ToolchainFingerprint: toolchainIdentity, Backend: backendIdentity,
			RuntimeABI: sdkstdlib.RuntimeABI, StandardLibrary: session.standardLibrary,
			Package: loaded.Instance, Source: loaded.Source, Dependencies: dependencies,
			Features: slices.Clone(loaded.Features),
		})
		if err != nil {
			return nil, fmt.Errorf("compute cache key for package %d: %w", packageID, err)
		}
		result[packageID] = key
	}
	return result, nil
}

func packageInterfaces(value hir.Package, metadata []workspaceloader.SemanticPackage, identities map[hir.PackageID]packagegraph.InstanceID) (map[hir.PackageID]moduleinterface.Hash, error) {
	result := make(map[hir.PackageID]moduleinterface.Hash, len(metadata))
	for _, loaded := range metadata {
		descriptor, err := moduleinterface.Extract(value, loaded.ID, identities)
		if err != nil {
			return nil, fmt.Errorf("extract public interface for package %d: %w", loaded.ID, err)
		}
		hash, err := moduleinterface.Compute(descriptor)
		if err != nil {
			return nil, fmt.Errorf("hash public interface for package %d: %w", loaded.ID, err)
		}
		result[loaded.ID] = hash
	}
	return result, nil
}

func functionOwners(value hir.Package) []hir.PackageID {
	set := make(map[hir.PackageID]struct{})
	for _, module := range value.Modules {
		if len(module.Functions) != 0 {
			set[module.Package] = struct{}{}
		}
	}
	result := make([]hir.PackageID, 0, len(set))
	for packageID := range set {
		result = append(result, packageID)
	}
	slices.Sort(result)
	return result
}

func semanticReady(result semantic.Result) bool {
	return result.CompletedThrough == semantic.PhaseFlow && result.FailedPhase == semantic.PhaseNone &&
		len(result.Diagnostics) == 0 && len(result.Unsupported) == 0
}

func standardLibraryInstance() (packagegraph.InstanceID, error) {
	definition, err := sdkstdlib.Load()
	if err != nil {
		return packagegraph.InstanceID{}, fmt.Errorf("load bundled standard library identity: %w", err)
	}
	identity, err := packagegraph.ComputeInstanceID(packagegraph.Identity{
		PackageUID: sdkstdlib.PackageUID, SourceIdentity: definition.SourceIdentity,
		VersionIdentity: sdkstdlib.Version, Features: []string{},
	})
	if err != nil {
		return packagegraph.InstanceID{}, fmt.Errorf("compute bundled standard library identity: %w", err)
	}
	return identity, nil
}

func abiProfile(spec target.Spec) string {
	return strings.Join([]string{
		"fairyc.target-abi-profile/0", string(spec.ID), spec.DataModel, spec.CallingConvention,
		spec.AggregateLayout, spec.ReturnClassifier, spec.ObjectFormat, spec.UnwindModel,
	}, ";")
}
