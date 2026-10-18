// Package workspacecompiler coordinates package-aware semantic analysis and process-local MIR reuse.
package workspacecompiler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
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
	StorageMemory      = "memory"
	StorageDisk        = "disk"

	compilerProcessEncoding = "fairyc.compiler-process/0:"
	compilerStableEncoding  = "fairyc.compiler-stable-package-cache/2"
	backendIdentity         = "verified-mir"
	toolchainIdentity       = "not-applicable:verified-mir"
)

const maxCompilerBuildIdentityBytes = 512

// PersistentConfiguration opts one Session into cross-process stable package-compilation reuse.
// CompilerBuild must identify the exact compiler build; using a release family or moving label is
// unsafe.
type PersistentConfiguration struct {
	Root          string
	RootBoundary  *os.Root
	Limits        packagecache.DiskLimits
	CompilerBuild string
}

// Configuration supplies the explicit target, artifact bound, and exactly one cache policy.
// Persistent nil selects the bounded process-local LRU. Persistent nonnil selects the bounded disk
// store and requires Cache to remain zero so callers cannot assume an unimplemented L1/L2 policy.
type Configuration struct {
	Target           target.ID
	Cache            packagecache.Limits
	MaxArtifactBytes uint64
	Persistent       *PersistentConfiguration
}

// Report describes one cache decision without treating raw byte-store hits as executable reuse.
type Report struct {
	Outcome            string
	Storage            string
	Packages           int
	Hits               int
	Misses             int
	Rejected           int
	Stored             int
	StoreSkipped       int
	StoreFailed        int
	Lowered            bool
	ReassemblyFallback bool
}

// Result contains the unified compiler result plus the exact process-cache
// decision that produced it.
type Result struct {
	Compiler compiler.Result
	Cache    Report
}

// Session owns exactly one bounded artifact cache: either process-local MIR or stable disk package
// compilation bundles.
type Session struct {
	cache           *packageartifact.Cache
	stableCache     *packageartifact.StableCache
	compilerBuild   string
	storage         string
	target          target.Spec
	standardLibrary packagegraph.InstanceID
	lower           func(context.Context, compiler.Analysis) (compiler.Result, error)
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
	var processCache *packageartifact.Cache
	var stableCache *packageartifact.StableCache
	var err error
	compilerBuild := ""
	storage := StorageMemory
	if configuration.Persistent == nil {
		processCache, err = packageartifact.NewCache(configuration.Cache, configuration.MaxArtifactBytes)
		if err != nil {
			return nil, err
		}
		compilerBuild = compilerProcessEncoding + processCache.Identity()
	} else {
		if configuration.Cache != (packagecache.Limits{}) {
			return nil, fmt.Errorf("workspace compiler cannot combine memory and persistent cache policies")
		}
		if configuration.Persistent.CompilerBuild == "" || !utf8.ValidString(configuration.Persistent.CompilerBuild) || len(configuration.Persistent.CompilerBuild) > maxCompilerBuildIdentityBytes {
			return nil, fmt.Errorf("persistent cache compiler build identity must be 1..%d bytes of valid UTF-8", maxCompilerBuildIdentityBytes)
		}
		if configuration.Persistent.RootBoundary == nil {
			stableCache, err = packageartifact.NewStableCache(configuration.Persistent.Root, configuration.Persistent.Limits, configuration.MaxArtifactBytes)
		} else {
			stableCache, err = packageartifact.NewStableCacheInRoot(configuration.Persistent.RootBoundary, configuration.Persistent.Root, configuration.Persistent.Limits, configuration.MaxArtifactBytes)
		}
		if err != nil {
			return nil, err
		}
		compilerBuild = strings.Join([]string{
			compilerStableEncoding,
			configuration.Persistent.CompilerBuild,
			packageartifact.StableCompilationVersion,
			packageartifact.StableArtifactVersion,
			contractstatemir.StableStateBodyVersion,
			packagecache.DiskRecordVersion,
		}, ";")
		storage = StorageDisk
	}
	standardLibrary, err := standardLibraryInstance()
	if err != nil {
		if stableCache != nil {
			_ = stableCache.Close()
		}
		return nil, err
	}
	return &Session{
		cache: processCache, stableCache: stableCache, compilerBuild: compilerBuild, storage: storage,
		target: targetSpec, standardLibrary: standardLibrary,
		lower: compiler.LowerAnalyzedContext,
	}, nil
}

// Close releases persistent filesystem resources. Process-local sessions have nothing to close.
func (session *Session) Close() error {
	if session == nil || session.stableCache == nil {
		return nil
	}
	return session.stableCache.Close()
}

// Check performs semantic analysis on every request. It skips MIR lowering only when every
// function-owning package is a verified cache hit and the complete artifact set reassembles.
func (session *Session) Check(workspace workspaceloader.Workspace) (Result, error) {
	return session.CheckContext(context.Background(), workspace)
}

func (session *Session) CheckContext(ctx context.Context, workspace workspaceloader.Workspace) (result Result, err error) {
	if err := ctx.Err(); err != nil {
		return Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: err}
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = Result{}, &compiler.Error{Stage: compiler.StageSemantic, Cause: canceled}
		}
	}()
	if session == nil || session.cache == nil && session.stableCache == nil || session.lower == nil {
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
	coordinates, err := workspace.PackageCoordinates()
	if err != nil {
		return Result{}, err
	}
	analysis, err := compiler.AnalyzePackagesContext(ctx, input)
	if err != nil {
		return Result{}, err
	}
	if !semanticReady(analysis.Semantic) {
		compiled, lowerErr := session.lower(ctx, analysis)
		return Result{Compiler: compiled, Cache: Report{Outcome: OutcomeNotEligible, Storage: session.storage}}, lowerErr
	}

	interfaces, err := packageInterfaces(analysis.Semantic.HIR, metadata, coordinates)
	if err != nil {
		return Result{}, err
	}
	owners := functionOwners(analysis.Semantic.HIR)
	report := Report{Outcome: OutcomeFallback, Storage: session.storage, Packages: len(owners)}
	if len(owners) == 0 {
		compiled, lowerErr := session.lower(ctx, analysis)
		report.Lowered = compiled.Verified
		return Result{Compiler: compiled, Cache: report}, lowerErr
	}
	bodies, err := contractstatemir.ComputeHIRStateBodyHashes(analysis.Semantic.HIR, owners, coordinates)
	if err != nil {
		return Result{}, fmt.Errorf("compute package contract-state body identities: %w", err)
	}
	keys, err := session.packageKeys(metadata, interfaces, bodies, owners)
	if err != nil {
		return Result{}, err
	}
	if session.stableCache != nil {
		return session.checkPersistent(ctx, analysis, identities, coordinates, interfaces, bodies, owners, keys, report)
	}

	artifacts := make([]packageartifact.Artifact, 0, len(owners))
	allHit := true
	for _, packageID := range owners {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
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
			return resultWithCurrentState(analysis, compiler.Result{Semantic: analysis.Semantic, MIR: program, Verified: true}, report)
		}
		report.ReassemblyFallback = true
	}

	compiled, err := session.lower(ctx, analysis)
	if err != nil {
		return Result{}, err
	}
	if !compiled.Verified {
		return Result{Compiler: compiled, Cache: report}, nil
	}
	report.Lowered = true
	extracted, err := packageartifact.ExtractAll(compiled.MIR, identities, coordinates)
	if err != nil {
		return Result{}, fmt.Errorf("extract package MIR artifacts: %w", err)
	}
	byPackage := make(map[hir.PackageID]packageartifact.Artifact, len(extracted))
	for _, artifact := range extracted {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		if _, duplicate := byPackage[artifact.PackageID]; duplicate {
			return Result{}, fmt.Errorf("lowered MIR produced duplicate package artifact %d", artifact.PackageID)
		}
		byPackage[artifact.PackageID] = artifact
	}
	for _, packageID := range owners {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
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

// PersistentStats returns stable-disk observations when the Session selected persistent storage.
func (session *Session) PersistentStats() (packageartifact.StableCacheStats, error) {
	if session == nil || session.stableCache == nil {
		return packageartifact.StableCacheStats{}, fmt.Errorf("workspace compiler session does not use persistent cache storage")
	}
	return session.stableCache.Stats()
}

func (session *Session) checkPersistent(ctx context.Context, analysis compiler.Analysis, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, interfaces map[hir.PackageID]moduleinterface.Hash, bodies map[hir.PackageID]contractstatemir.StateBodyHash, owners []hir.PackageID, keys map[hir.PackageID]packagecache.Key, report Report) (Result, error) {
	compilations := make([]packageartifact.StableCompilationArtifact, 0, len(owners))
	allHit := true
	for _, packageID := range owners {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		instance, found := identities[packageID]
		if !found {
			return Result{}, fmt.Errorf("function-owning package %d has no stable package identity", packageID)
		}
		expectedBody, found := bodies[packageID]
		if !found || expectedBody == (contractstatemir.StateBodyHash{}) {
			return Result{}, fmt.Errorf("function-owning package %d has no contract-state body identity", packageID)
		}
		lookup := session.stableCache.Get(keys[packageID], interfaces[packageID], instance, expectedBody)
		switch lookup.Status {
		case packageartifact.LookupHit:
			report.Hits++
			compilations = append(compilations, lookup.Compilation)
		case packageartifact.LookupMiss:
			report.Misses++
			allHit = false
		case packageartifact.LookupRejected:
			report.Rejected++
			allHit = false
		default:
			return Result{}, fmt.Errorf("workspace compiler persistent cache returned unknown status %q", lookup.Status)
		}
	}
	if allHit {
		program, stateTransactions, reassemblyErr := rebindPersistentCompilations(
			compilations, analysis.Semantic.HIR, coordinates, bodies, owners,
		)
		if reassemblyErr == nil {
			report.Outcome = OutcomeAllHit
			return Result{
				Compiler: compiler.Result{
					Semantic: analysis.Semantic, MIR: program,
					StateTransactions: stateTransactions, Verified: true,
				},
				Cache: report,
			}, nil
		}
		report.ReassemblyFallback = true
	}

	compiled, err := session.lower(ctx, analysis)
	if err != nil {
		return Result{}, err
	}
	if !compiled.Verified {
		return Result{Compiler: compiled, Cache: report}, nil
	}
	report.Lowered = true
	extracted, err := packageartifact.ExtractAll(compiled.MIR, identities, coordinates)
	if err != nil {
		return Result{}, fmt.Errorf("extract stable package MIR artifacts: %w", err)
	}
	byPackage := make(map[hir.PackageID]packageartifact.Artifact, len(extracted))
	for _, artifact := range extracted {
		if _, duplicate := byPackage[artifact.PackageID]; duplicate {
			return Result{}, fmt.Errorf("lowered MIR produced duplicate package artifact %d", artifact.PackageID)
		}
		byPackage[artifact.PackageID] = artifact
	}
	stateBodies, err := contractstatemir.DescribeStableHIRStateBodies(analysis.Semantic.HIR, owners, coordinates)
	if err != nil {
		return Result{}, fmt.Errorf("describe stable package contract-state bodies: %w", err)
	}
	for _, packageID := range owners {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		artifact, found := byPackage[packageID]
		if !found {
			return Result{}, fmt.Errorf("lowered MIR has no artifact for function-owning package %d", packageID)
		}
		stateBody, found := stateBodies[packageID]
		if !found || stateBody.Hash != bodies[packageID] {
			return Result{}, fmt.Errorf("stable contract-state body for package %d does not match the cache-key identity", packageID)
		}
		if err := session.stableCache.Put(keys[packageID], interfaces[packageID], artifact, stateBody); err != nil {
			if errors.Is(err, packageartifact.ErrStableCompilationTooLarge) || errors.Is(err, packageartifact.ErrStableArtifactTooLarge) || errors.Is(err, packageartifact.ErrBodyArtifactTooLarge) || errors.Is(err, contractstatemir.ErrStableStateBodyTooLarge) || errors.Is(err, packagecache.ErrRecordTooLarge) {
				report.StoreSkipped++
				continue
			}
			if errors.Is(err, packageartifact.ErrStableCacheStore) {
				report.StoreFailed++
				continue
			}
			return Result{}, fmt.Errorf("store package %d stable compilation artifact: %w", packageID, err)
		}
		report.Stored++
	}
	return Result{Compiler: compiled, Cache: report}, nil
}

func rebindPersistentCompilations(compilations []packageartifact.StableCompilationArtifact, input hir.Package, coordinates map[hir.PackageID]packagegraph.CoordinateID, bodies map[hir.PackageID]contractstatemir.StateBodyHash, owners []hir.PackageID) (mir.Program, []contractstatemir.LoweredTransaction, error) {
	artifacts := make([]packageartifact.StableArtifact, 0, len(compilations))
	stateByCoordinate := make(map[packagegraph.CoordinateID]contractstatemir.StableStateBodyArtifact, len(compilations))
	for _, compilation := range compilations {
		artifacts = append(artifacts, compilation.MIR)
		if _, duplicate := stateByCoordinate[compilation.MIR.Coordinate]; duplicate {
			return mir.Program{}, nil, fmt.Errorf("stable package compilation coordinate %s is duplicated", compilation.MIR.Coordinate)
		}
		stateByCoordinate[compilation.MIR.Coordinate] = compilation.StateBody
	}
	functionCount := 0
	for _, artifact := range artifacts {
		if len(artifact.Functions) > int(^uint(0)>>1)-functionCount {
			return mir.Program{}, nil, fmt.Errorf("stable package function namespace overflows")
		}
		functionCount += len(artifact.Functions)
	}
	firstFunction := hir.SymbolID(1)
	firstForeign := firstFunction + hir.SymbolID(functionCount)
	if firstForeign < firstFunction {
		return mir.Program{}, nil, fmt.Errorf("stable package symbol namespace overflows")
	}
	reassembled, err := packageartifact.RebindStableArtifactsWithBindings(artifacts, coordinates, firstFunction, firstForeign, mir.LocalID(1))
	if err != nil {
		return mir.Program{}, nil, err
	}
	symbolsByCoordinate := make(map[packagegraph.CoordinateID]map[string]hir.SymbolID, len(compilations))
	for _, binding := range reassembled.Functions {
		symbols := symbolsByCoordinate[binding.Package]
		if symbols == nil {
			symbols = make(map[string]hir.SymbolID)
			symbolsByCoordinate[binding.Package] = symbols
		}
		if _, duplicate := symbols[binding.Name]; duplicate {
			return mir.Program{}, nil, fmt.Errorf("stable package %s has multiple MIR functions named %q", binding.Package, binding.Name)
		}
		symbols[binding.Name] = binding.Symbol
	}
	transactions := make([]contractstatemir.LoweredTransaction, 0)
	for _, packageID := range owners {
		coordinate, found := coordinates[packageID]
		if !found || coordinate == (packagegraph.CoordinateID{}) {
			return mir.Program{}, nil, fmt.Errorf("function-owning package %d has no stable package coordinate", packageID)
		}
		stateBody, found := stateByCoordinate[coordinate]
		if !found {
			return mir.Program{}, nil, fmt.Errorf("function-owning package %d has no cached contract-state body", packageID)
		}
		expected, found := bodies[packageID]
		if !found || expected == (contractstatemir.StateBodyHash{}) {
			return mir.Program{}, nil, fmt.Errorf("function-owning package %d has no expected contract-state body identity", packageID)
		}
		rebound, err := contractstatemir.RebindStableStateBodyWithSymbols(
			stateBody, expected, input, packageID, coordinates, symbolsByCoordinate[coordinate],
		)
		if err != nil {
			return mir.Program{}, nil, fmt.Errorf("rebind package %d cached contract-state body: %w", packageID, err)
		}
		transactions = append(transactions, rebound.Transactions...)
	}
	return reassembled.MIR, transactions, nil
}

func resultWithCurrentState(analysis compiler.Analysis, compiled compiler.Result, report Report) (Result, error) {
	transactions, err := contractstatemir.LowerHIRTransactions(analysis.Semantic.HIR)
	if err != nil {
		return Result{}, fmt.Errorf("lower current HIR contract-state transactions: %w", err)
	}
	compiled.StateTransactions = transactions
	return Result{Compiler: compiled, Cache: report}, nil
}

func (session *Session) packageKeys(metadata []workspaceloader.SemanticPackage, interfaces map[hir.PackageID]moduleinterface.Hash, bodies map[hir.PackageID]contractstatemir.StateBodyHash, owners []hir.PackageID) (map[hir.PackageID]packagecache.Key, error) {
	inputs, err := session.packageInputs(metadata, interfaces, bodies, owners)
	if err != nil {
		return nil, err
	}
	result := make(map[hir.PackageID]packagecache.Key, len(owners))
	for _, packageID := range owners {
		key, err := packagecache.Compute(inputs[packageID])
		if err != nil {
			return nil, fmt.Errorf("compute cache key for package %d: %w", packageID, err)
		}
		result[packageID] = key
	}
	return result, nil
}

func (session *Session) packageInputs(metadata []workspaceloader.SemanticPackage, interfaces map[hir.PackageID]moduleinterface.Hash, bodies map[hir.PackageID]contractstatemir.StateBodyHash, owners []hir.PackageID) (map[hir.PackageID]packagecache.Inputs, error) {
	byID := make(map[hir.PackageID]workspaceloader.SemanticPackage, len(metadata))
	for _, loaded := range metadata {
		byID[loaded.ID] = loaded
	}
	result := make(map[hir.PackageID]packagecache.Inputs, len(owners))
	for _, packageID := range owners {
		loaded, found := byID[packageID]
		if !found {
			return nil, fmt.Errorf("function-owning package %d has no workspace metadata", packageID)
		}
		currentInterface, found := interfaces[packageID]
		if !found {
			return nil, fmt.Errorf("function-owning package %d has no public interface", packageID)
		}
		currentBody, found := bodies[packageID]
		if !found || currentBody == (contractstatemir.StateBodyHash{}) {
			return nil, fmt.Errorf("function-owning package %d has no contract-state body identity", packageID)
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
			dependencies = append(dependencies, packagecache.Dependency{Coordinate: dependency.Coordinate, Interface: interfaceHash})
		}
		dependencies, err := packagecache.CanonicalizeDependencies(dependencies)
		if err != nil {
			return nil, err
		}
		result[packageID] = packagecache.Inputs{
			CompilerBuild:    session.compilerBuild,
			LanguageRevision: sdkstdlib.LanguageRevision, MIRVersion: mircodec.MIRVersion,
			TargetSchema: session.target.SchemaVersion, ABIProfile: abiProfile(session.target),
			ToolchainFingerprint: toolchainIdentity, Backend: backendIdentity,
			RuntimeABI: sdkstdlib.RuntimeABI, StandardLibrary: session.standardLibrary,
			Package: loaded.Instance, Source: loaded.Source, CurrentInterface: currentInterface, CurrentBody: currentBody, Dependencies: dependencies,
			Features: slices.Clone(loaded.Features),
		}
	}
	return result, nil
}

func packageInterfaces(value hir.Package, metadata []workspaceloader.SemanticPackage, coordinates map[hir.PackageID]packagegraph.CoordinateID) (map[hir.PackageID]moduleinterface.Hash, error) {
	packageIDs := make([]hir.PackageID, 0, len(metadata))
	for _, loaded := range metadata {
		packageIDs = append(packageIDs, loaded.ID)
	}
	return moduleinterface.ComputePackages(value, packageIDs, coordinates)
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
