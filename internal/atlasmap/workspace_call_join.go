package atlasmap

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

var (
	ErrWorkspaceJoinBudget     = errors.New("Atlas workspace call join exceeds its explicit budget")
	ErrWorkspaceJoinIncomplete = errors.New("Atlas workspace call join is incomplete")
)

// WorkspaceCallBudget bounds the already-loaded exact-current package and resolved-call records
// admitted by one join. Construction performs no filesystem discovery or source loading.
type WorkspaceCallBudget struct {
	MaxPackages int
	MaxCalls    int
}

// WorkspaceCallUsage reports exact admitted counts. CrossPackageCalls is a subset of Calls.
type WorkspaceCallUsage struct {
	Packages          int
	Calls             int
	CrossPackageCalls int
}

// WorkspaceSymbol qualifies a canonical visible SymbolId with the exact package instance whose
// current map owns it. The package qualifier also scopes declaration source paths downstream.
type WorkspaceSymbol struct {
	PackageInstance string
	Symbol          string
}

// JoinedCall is one compiler-resolved call whose source and target both have exact-current maps.
// It is a detached compiler-fact record, not an impact or safe-change claim.
type JoinedCall struct {
	Source     WorkspaceSymbol
	Target     WorkspaceSymbol
	Span       semanticsnapshot.Span
	Certainty  semanticsnapshot.Certainty
	Provenance semanticsnapshot.Provenance
}

// WorkspaceCallIndex is an immutable, bounded reverse-call ledger over a caller-supplied complete
// exact-current workspace partition set. It does not replace package-local QueryIndex traversal.
type WorkspaceCallIndex struct {
	budget   WorkspaceCallBudget
	usage    WorkspaceCallUsage
	calls    []JoinedCall
	outgoing map[WorkspaceSymbol][]int
	incoming map[WorkspaceSymbol][]int
	packages map[string]*QueryIndex
}

// NewWorkspaceCallIndex validates every package through NewQueryIndex, requires the complete direct
// dependency closure, resolves every non-foreign call target by DeclarationKey, and builds exact
// outgoing and incoming ledgers. Caller order cannot affect the result.
func NewWorkspaceCallIndex(
	partitions []Partition,
	expectations map[string]LoadExpectation,
	budget WorkspaceCallBudget,
) (*WorkspaceCallIndex, error) {
	if budget.MaxPackages <= 0 || budget.MaxCalls < 0 {
		return nil, fmt.Errorf("Atlas workspace call join budget is invalid")
	}
	if len(partitions) == 0 {
		return nil, fmt.Errorf("%w: no package partitions were supplied", ErrWorkspaceJoinIncomplete)
	}
	if len(partitions) > budget.MaxPackages {
		return nil, fmt.Errorf("%w: packages %d exceed %d", ErrWorkspaceJoinBudget, len(partitions), budget.MaxPackages)
	}
	ordered := slices.Clone(partitions)
	slices.SortFunc(ordered, func(left, right Partition) int {
		return cmp.Compare(left.data.header.PackageInstance, right.data.header.PackageInstance)
	})
	index := &WorkspaceCallIndex{
		budget:   budget,
		usage:    WorkspaceCallUsage{Packages: len(ordered)},
		outgoing: make(map[WorkspaceSymbol][]int), incoming: make(map[WorkspaceSymbol][]int),
		packages: make(map[string]*QueryIndex, len(ordered)),
	}
	for position, partition := range ordered {
		header := partition.data.header
		if position > 0 && ordered[position-1].data.header.PackageInstance == header.PackageInstance {
			return nil, fmt.Errorf("%w: duplicate package partition %q", ErrWorkspaceJoinIncomplete, header.PackageInstance)
		}
		expectation, found := expectations[header.PackageInstance]
		if !found {
			return nil, fmt.Errorf("%w: package %q has no load expectation", ErrWorkspaceJoinIncomplete, header.PackageInstance)
		}
		packageIndex, err := NewQueryIndex(partition, expectation)
		if err != nil {
			return nil, fmt.Errorf("build Atlas workspace call join package %q: %w", header.PackageInstance, err)
		}
		index.packages[header.PackageInstance] = packageIndex
	}
	if len(expectations) != len(index.packages) {
		return nil, fmt.Errorf("%w: expectation and partition package sets differ", ErrWorkspaceJoinIncomplete)
	}
	for packageInstance := range expectations {
		if _, found := index.packages[packageInstance]; !found {
			return nil, fmt.Errorf("%w: expectation names unknown package %q", ErrWorkspaceJoinIncomplete, packageInstance)
		}
	}
	for _, partition := range ordered {
		header := partition.data.header
		packageIndex := index.packages[header.PackageInstance]
		for _, dependency := range header.Dependencies {
			if _, found := index.packages[dependency]; !found {
				return nil, fmt.Errorf("%w: package %q dependency %q has no exact-current partition", ErrWorkspaceJoinIncomplete, header.PackageInstance, dependency)
			}
		}
		for _, relation := range packageIndex.relations {
			if relation.Kind != RelationCalls {
				continue
			}
			if relation.Target == nil {
				return nil, fmt.Errorf("%w: resolved call from package %q has no target", ErrWorkspaceJoinIncomplete, header.PackageInstance)
			}
			if relation.Span == nil || relation.Certainty != semanticsnapshot.CertaintyExact ||
				relation.Provenance != semanticsnapshot.ProvenanceCompilerObserved {
				return nil, fmt.Errorf("%w: resolved call from package %q lacks exact observed provenance", ErrWorkspaceJoinIncomplete, header.PackageInstance)
			}
			if relation.Target.PackageInstance != header.PackageInstance &&
				!slices.Contains(header.Dependencies, relation.Target.PackageInstance) {
				return nil, fmt.Errorf("%w: package %q call target %q is not self or a direct dependency", ErrWorkspaceJoinIncomplete, header.PackageInstance, relation.Target.PackageInstance)
			}
			targetIndex, found := index.packages[relation.Target.PackageInstance]
			if !found {
				return nil, fmt.Errorf("%w: call target package %q has no exact-current partition", ErrWorkspaceJoinIncomplete, relation.Target.PackageInstance)
			}
			targetKey := semanticsnapshot.DeclarationKey{
				PackageInstance: relation.Target.PackageInstance,
				Module:          relation.Target.Module,
				Kind:            relation.Target.Kind,
				Name:            relation.Target.Name,
			}
			targetPosition, found := targetIndex.byKey[targetKey]
			if !found {
				return nil, fmt.Errorf("%w: call target %q %s::%s has no exact declaration", ErrWorkspaceJoinIncomplete, relation.Target.PackageInstance, relation.Target.Module, relation.Target.Name)
			}
			targetNode := targetIndex.nodes[targetPosition]
			if relation.Target.Symbol != "" && relation.Target.Symbol != targetNode.symbol.Visible {
				return nil, fmt.Errorf("%w: package-local and workspace target symbols disagree", ErrWorkspaceJoinIncomplete)
			}
			if relation.Target.PackageInstance != header.PackageInstance && !targetNode.declaration.Public {
				return nil, fmt.Errorf("%w: cross-package call target is not public", ErrWorkspaceJoinIncomplete)
			}
			sourcePosition, found := packageIndex.byVisible[relation.Source]
			if !found {
				return nil, fmt.Errorf("%w: call source %q is absent from its package", ErrWorkspaceJoinIncomplete, relation.Source)
			}
			if len(index.calls) >= budget.MaxCalls {
				return nil, fmt.Errorf("%w: calls exceed %d", ErrWorkspaceJoinBudget, budget.MaxCalls)
			}
			call := JoinedCall{
				Source: WorkspaceSymbol{PackageInstance: header.PackageInstance, Symbol: packageIndex.nodes[sourcePosition].symbol.Visible},
				Target: WorkspaceSymbol{PackageInstance: relation.Target.PackageInstance, Symbol: targetNode.symbol.Visible},
				Span:   *relation.Span, Certainty: relation.Certainty, Provenance: relation.Provenance,
			}
			callPosition := len(index.calls)
			index.calls = append(index.calls, call)
			index.outgoing[call.Source] = append(index.outgoing[call.Source], callPosition)
			index.incoming[call.Target] = append(index.incoming[call.Target], callPosition)
			if call.Source.PackageInstance != call.Target.PackageInstance {
				index.usage.CrossPackageCalls++
			}
		}
	}
	index.usage.Calls = len(index.calls)
	return index, nil
}

func (index *WorkspaceCallIndex) Budget() WorkspaceCallBudget {
	if index == nil {
		return WorkspaceCallBudget{}
	}
	return index.budget
}

func (index *WorkspaceCallIndex) Usage() WorkspaceCallUsage {
	if index == nil {
		return WorkspaceCallUsage{}
	}
	return index.usage
}

// ResolveSymbol canonicalizes a visible or full SymbolId inside one exact package instance.
func (index *WorkspaceCallIndex) ResolveSymbol(packageInstance, symbolID string) (WorkspaceSymbol, error) {
	return index.resolveWorkspaceSymbol(packageInstance, symbolID)
}

// PackageContext returns detached exact-current identity and completeness facts for one joined
// package without exposing its mutable implementation index.
func (index *WorkspaceCallIndex) PackageContext(packageInstance string) (QueryContext, error) {
	if index == nil {
		return QueryContext{}, fmt.Errorf("Atlas workspace call index is not initialized")
	}
	packageIndex, found := index.packages[packageInstance]
	if !found {
		return QueryContext{}, fmt.Errorf("Atlas workspace call package %q is unknown", packageInstance)
	}
	return packageIndex.contextFor([]RelationKind{RelationCalls}), nil
}

func (index *WorkspaceCallIndex) Calls() []JoinedCall {
	if index == nil {
		return nil
	}
	return slices.Clone(index.calls)
}

func (index *WorkspaceCallIndex) OutgoingCalls(packageInstance, symbolID string) ([]JoinedCall, error) {
	key, err := index.resolveWorkspaceSymbol(packageInstance, symbolID)
	if err != nil {
		return nil, err
	}
	return index.selectCalls(index.outgoing[key]), nil
}

func (index *WorkspaceCallIndex) IncomingCalls(packageInstance, symbolID string) ([]JoinedCall, error) {
	key, err := index.resolveWorkspaceSymbol(packageInstance, symbolID)
	if err != nil {
		return nil, err
	}
	return index.selectCalls(index.incoming[key]), nil
}

func (index *WorkspaceCallIndex) resolveWorkspaceSymbol(packageInstance, symbolID string) (WorkspaceSymbol, error) {
	if index == nil {
		return WorkspaceSymbol{}, fmt.Errorf("Atlas workspace call index is not initialized")
	}
	packageIndex, found := index.packages[packageInstance]
	if !found {
		return WorkspaceSymbol{}, fmt.Errorf("Atlas workspace call package %q is unknown", packageInstance)
	}
	position, err := packageIndex.resolveSymbol(symbolID)
	if err != nil {
		return WorkspaceSymbol{}, err
	}
	return WorkspaceSymbol{PackageInstance: packageInstance, Symbol: packageIndex.nodes[position].symbol.Visible}, nil
}

func (index *WorkspaceCallIndex) selectCalls(positions []int) []JoinedCall {
	result := make([]JoinedCall, 0, len(positions))
	for _, position := range positions {
		result = append(result, index.calls[position])
	}
	return result
}
