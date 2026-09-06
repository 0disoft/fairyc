package workspacecompiler

import (
	"errors"
	"fmt"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func (session *GenericSession) checkGenericPersistent(analysis compiler.GenericAnalysis, identities map[hir.PackageID]packagegraph.InstanceID, coordinates map[hir.PackageID]packagegraph.CoordinateID, owners []hir.PackageID, keys map[hir.PackageID]packagecache.GenericKey, report Report) (Result, error) {
	var artifacts []packageartifact.StableArtifact
	allHit := true
	for _, id := range owners {
		lookup := session.stableCache.Get(keys[id], analysis.Interfaces[id], identities[id], coordinates[id])
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
			return Result{}, fmt.Errorf("unknown generic persistent lookup status")
		}
	}
	if allHit {
		functionCount := uint64(0)
		maxSymbols := uint64(^uint(0) >> 1)
		for _, artifact := range artifacts {
			if uint64(len(artifact.Functions)) >= maxSymbols-functionCount {
				return Result{}, fmt.Errorf("generic persistent function namespace overflows")
			}
			functionCount += uint64(len(artifact.Functions))
		}
		program, err := packageartifact.RebindStableArtifacts(artifacts, coordinates, 1, hir.SymbolID(functionCount)+1, 1)
		if err == nil {
			report.Outcome = OutcomeAllHit
			return Result{Compiler: compiler.Result{Semantic: analysis.Analysis.Semantic, MIR: program, Verified: true, GenericProvenance: analysis.Analysis.GenericProvenance}, Cache: report}, nil
		}
		report.ReassemblyFallback = true
	}
	compiled, err := session.lower(analysis.Analysis)
	if err != nil {
		return Result{}, err
	}
	if !compiled.Verified {
		return Result{Compiler: compiled, Cache: report}, nil
	}
	report.Lowered = true
	extracted, err := packageartifact.ExtractAll(compiled.MIR, identities, coordinates)
	if err != nil {
		return Result{}, err
	}
	for _, artifact := range extracted {
		key, found := keys[artifact.PackageID]
		if !found {
			return Result{}, fmt.Errorf("generic persistent artifact has no current key")
		}
		if err := session.stableCache.Put(key, analysis.Interfaces[artifact.PackageID], artifact); err != nil {
			if errors.Is(err, packageartifact.ErrStableArtifactTooLarge) || errors.Is(err, packageartifact.ErrContextArtifactTooLarge) || errors.Is(err, packageartifact.ErrBodyArtifactTooLarge) || errors.Is(err, packagecache.ErrRecordTooLarge) {
				report.StoreSkipped++
				continue
			}
			if errors.Is(err, packageartifact.ErrStableCacheStore) {
				report.StoreFailed++
				continue
			}
			return Result{}, err
		}
		report.Stored++
	}
	return Result{Compiler: compiled, Cache: report}, nil
}
