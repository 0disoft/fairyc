package workspacecompiler

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// GenericSession owns exactly one bounded memory or persistent generic MIR cache.
// SourceSession selects this family when the loaded workspace contains generics.
type GenericSession struct {
	cache        *packageartifact.GenericCache
	stableCache  *packageartifact.GenericStableCache
	storage      string
	keys         Session
	limits       genericplan.PlanLimits
	lower        func(compiler.Analysis, admission.Capabilities) (compiler.Result, error)
	capabilities admission.Capabilities
	artifacts    genericArtifactPolicy
}

func NewGenericSession(configuration Configuration, limits genericplan.PlanLimits) (*GenericSession, error) {
	return NewGenericSessionWithCapabilities(configuration, limits, admission.Capabilities{})
}

// NewGenericSessionWithCapabilities selects one supported cache admission family.
// SDK source owners are authenticated by the workspace on each Check, not here.
func NewGenericSessionWithCapabilities(configuration Configuration, limits genericplan.PlanLimits, capabilities admission.Capabilities) (*GenericSession, error) {
	artifacts, err := genericArtifactsFor(capabilities)
	if err != nil {
		return nil, err
	}
	if limits.MaxSpecializations <= 0 || limits.MaxSpecializationsPerDeclaration <= 0 || limits.MaxExpansionDepth <= 0 {
		return nil, fmt.Errorf("generic workspace planning limits must be positive")
	}
	spec, found := target.Lookup(configuration.Target)
	if !found {
		return nil, fmt.Errorf("generic workspace target is unknown")
	}
	if err := target.Validate(spec); err != nil {
		return nil, err
	}
	stdlib, err := standardLibraryInstance()
	if err != nil {
		return nil, err
	}
	session := &GenericSession{keys: Session{target: spec, standardLibrary: stdlib}, limits: limits, lower: compiler.LowerAnalyzedWithCapabilities, storage: StorageMemory, capabilities: capabilities, artifacts: artifacts}
	if configuration.Persistent == nil {
		session.cache, err = artifacts.newMemory(configuration.Cache, configuration.MaxArtifactBytes)
		if err != nil {
			return nil, err
		}
		session.keys.compilerBuild = compilerProcessEncoding + session.cache.Identity() + artifacts.processSuffix
	} else {
		if configuration.Cache != (packagecache.Limits{}) {
			return nil, fmt.Errorf("generic workspace cannot combine memory and persistent cache policies")
		}
		persistent := configuration.Persistent
		if _, err := buildidentity.Parse(persistent.CompilerBuild); err != nil {
			return nil, fmt.Errorf("generic persistent cache requires an exact compiler executable identity: %w", err)
		}
		if persistent.RootBoundary == nil {
			session.stableCache, err = artifacts.newDisk(persistent.Root, persistent.Limits, configuration.MaxArtifactBytes)
		} else {
			session.stableCache, err = artifacts.newDiskInRoot(persistent.RootBoundary, persistent.Root, persistent.Limits, configuration.MaxArtifactBytes)
		}
		if err != nil {
			return nil, err
		}
		session.keys.compilerBuild = strings.Join([]string{artifacts.family, persistent.CompilerBuild, artifacts.version, packagecache.DiskRecordVersion}, ";")
		session.storage = StorageDisk
	}
	if capabilities.SDKTemplates() {
		session.keys.compilerBuild += ":sdk-owned-storage-v1"
	}
	return session, nil
}

func (session *GenericSession) Close() error {
	if session == nil || session.stableCache == nil {
		return nil
	}
	return session.stableCache.Close()
}

func (session *GenericSession) Check(workspace workspaceloader.Workspace) (Result, error) {
	if session == nil || session.cache == nil && session.stableCache == nil || session.lower == nil {
		return Result{}, fmt.Errorf("generic workspace session is not initialized")
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
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return Result{}, err
	}
	policy, err := workspaceAdmission(workspace, session.capabilities)
	if err != nil {
		return Result{}, err
	}
	analysis, err := compiler.AnalyzeWithPolicy(input, coordinates, session.limits, policy)
	if err != nil {
		return Result{}, err
	}
	if !semanticReady(analysis.Analysis.Semantic) || len(analysis.Analysis.Semantic.HIR.StateTransactions) != 0 {
		compiled, err := session.lower(analysis.Analysis, policy.Capabilities)
		return Result{Compiler: compiled, Cache: Report{Outcome: OutcomeNotEligible, Storage: session.storage, Lowered: compiled.Verified}}, err
	}
	owners := functionOwners(analysis.Analysis.Semantic.HIR)
	report := Report{Outcome: OutcomeFallback, Storage: session.storage, Packages: len(owners)}
	if len(owners) == 0 {
		compiled, err := session.lower(analysis.Analysis, policy.Capabilities)
		report.Lowered = compiled.Verified
		return Result{Compiler: compiled, Cache: report}, err
	}
	bodies, err := contractstatemir.ComputeHIRStateBodyHashes(analysis.Analysis.Semantic.HIR, owners, coordinates)
	if err != nil {
		return Result{}, err
	}
	base, err := session.keys.packageInputs(metadata, analysis.OrdinaryInterfaces, bodies, owners)
	if err != nil {
		return Result{}, err
	}
	interfaces := map[packagegraph.CoordinateID]moduleinterface.GenericHash{}
	for _, loaded := range metadata {
		interfaces[loaded.Coordinate] = analysis.Interfaces[loaded.ID]
	}
	keys := map[hir.PackageID]packagecache.GenericKey{}
	for _, id := range owners {
		var dependencies []packagecache.GenericDependency
		for _, dependency := range base[id].Dependencies {
			dependencies = append(dependencies, packagecache.GenericDependency{Coordinate: dependency.Coordinate, Interface: interfaces[dependency.Coordinate]})
		}
		key, err := packagecache.ComputeGeneric(packagecache.GenericInputs{Base: base[id], CurrentInterface: analysis.Interfaces[id], Dependencies: dependencies, Specializations: analysis.Specializations[id]})
		if err != nil {
			return Result{}, err
		}
		keys[id] = key
	}
	if session.stableCache != nil {
		return session.checkGenericPersistent(analysis, identities, coordinates, owners, keys, report, policy.Capabilities)
	}
	var artifacts []packageartifact.Artifact
	allHit := true
	for _, id := range owners {
		lookup := session.cache.Get(keys[id], analysis.Interfaces[id])
		switch lookup.Status {
		case packageartifact.LookupMiss:
			report.Misses++
			allHit = false
		case packageartifact.LookupRejected:
			report.Rejected++
			allHit = false
		case packageartifact.LookupHit:
			report.Hits++
			artifact := lookup.Artifact
			if artifact.PackageID != id || artifact.Package != identities[id] || !maps.Equal(artifact.PackageIdentities, identities) || !maps.Equal(artifact.PackageCoordinates, coordinates) {
				allHit = false
				report.ReassemblyFallback = true
			}
			artifacts = append(artifacts, artifact)
		default:
			return Result{}, fmt.Errorf("unknown generic artifact lookup status")
		}
	}
	if allHit {
		program, err := session.artifacts.reassemble(artifacts)
		if err == nil {
			report.Outcome = OutcomeAllHit
			return Result{Compiler: compiler.Result{Semantic: analysis.Analysis.Semantic, MIR: program, Verified: true, GenericProvenance: analysis.Analysis.GenericProvenance}, Cache: report}, nil
		}
		report.ReassemblyFallback = true
	}
	compiled, err := session.lower(analysis.Analysis, policy.Capabilities)
	if err != nil {
		return Result{}, err
	}
	if !compiled.Verified {
		return Result{Compiler: compiled, Cache: report}, nil
	}
	report.Lowered = true
	extracted, err := session.artifacts.extract(compiled.MIR, identities, coordinates)
	if err != nil {
		return Result{}, err
	}
	for _, artifact := range extracted {
		key, found := keys[artifact.PackageID]
		if !found {
			return Result{}, fmt.Errorf("generic artifact has no current cache key")
		}
		if err := session.cache.Put(key, analysis.Interfaces[artifact.PackageID], artifact); err != nil {
			if errors.Is(err, packagecache.ErrRecordTooLarge) || errors.Is(err, packageartifact.ErrProcessArtifactTooLarge) {
				report.StoreSkipped++
				continue
			}
			return Result{}, err
		}
		report.Stored++
	}
	return Result{Compiler: compiled, Cache: report}, nil
}

func (session *GenericSession) Stats() packageartifact.CacheStats {
	if session == nil || session.cache == nil {
		return packageartifact.CacheStats{}
	}
	return session.cache.Stats()
}

func (session *GenericSession) PersistentStats() (packageartifact.StableCacheStats, error) {
	if session == nil || session.stableCache == nil {
		return packageartifact.StableCacheStats{}, fmt.Errorf("generic workspace session does not use persistent storage")
	}
	return session.stableCache.Stats()
}
