package atlasmap

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolidentity"
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
	SourceSpecialization string
	TargetSpecialization string
	Source               WorkspaceSymbol
	Target               WorkspaceSymbol
	Span                 semanticsnapshot.Span
	Certainty            semanticsnapshot.Certainty
	Provenance           semanticsnapshot.Provenance
}

// WorkspaceCallIndex is an immutable, bounded reverse-call ledger over a caller-supplied complete
// exact-current workspace partition set. It does not replace package-local QueryIndex traversal.
type WorkspaceCallIndex struct {
	budget   WorkspaceCallBudget
	usage    WorkspaceCallUsage
	calls    []JoinedCall
	outgoing map[WorkspaceSymbol][]int
	incoming map[WorkspaceSymbol][]int
	packages map[string]workspaceCallPackage
}

// workspaceCallPackage is the compact package state retained after construction. Full QueryIndex
// values are needed while validating and joining, but keeping their files, declarations, and
// relation indexes afterward would duplicate facts that the workspace call ledger cannot query.
type workspaceCallPackage struct {
	context   QueryContext
	byVisible map[string]struct{}
	byFull    map[string]string
}

// NewWorkspaceCallIndex validates every package through NewQueryIndex, requires the complete direct
// dependency closure, resolves every non-foreign call target by DeclarationKey, and builds exact
// outgoing and incoming ledgers. Caller order cannot affect the result.
func NewWorkspaceCallIndex(
	partitions []Partition,
	expectations map[string]LoadExpectation,
	budget WorkspaceCallBudget,
) (*WorkspaceCallIndex, error) {
	return newWorkspaceCallIndex(partitions, expectations, budget, LogicalSchema)
}

// NewWorkspaceCallIndexV2 preserves concrete implementation endpoints while joining v2 maps.
func NewWorkspaceCallIndexV2(partitions []Partition, expectations map[string]LoadExpectation, budget WorkspaceCallBudget) (*WorkspaceCallIndex, error) {
	return newWorkspaceCallIndex(partitions, expectations, budget, LogicalSchemaV2)
}

func newWorkspaceCallIndex(partitions []Partition, expectations map[string]LoadExpectation, budget WorkspaceCallBudget, schema string) (*WorkspaceCallIndex, error) {
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
		packages: make(map[string]workspaceCallPackage, len(ordered)),
	}
	packageIndexes := make(map[string]*QueryIndex, len(ordered))
	for position, partition := range ordered {
		header := partition.data.header
		if position > 0 && ordered[position-1].data.header.PackageInstance == header.PackageInstance {
			return nil, fmt.Errorf("%w: duplicate package partition %q", ErrWorkspaceJoinIncomplete, header.PackageInstance)
		}
		expectation, found := expectations[header.PackageInstance]
		if !found {
			return nil, fmt.Errorf("%w: package %q has no load expectation", ErrWorkspaceJoinIncomplete, header.PackageInstance)
		}
		packageIndex, err := newQueryIndex(partition, expectation, schema)
		if err != nil {
			return nil, fmt.Errorf("build Atlas workspace call join package %q: %w", header.PackageInstance, err)
		}
		packageIndexes[header.PackageInstance] = packageIndex
		index.packages[header.PackageInstance] = compactWorkspaceCallPackage(packageIndex)
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
		packageIndex := packageIndexes[header.PackageInstance]
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
			if relation.Target.PackageInstance != header.PackageInstance {
				_, directDependency := slices.BinarySearch(header.Dependencies, relation.Target.PackageInstance)
				if !directDependency {
					return nil, fmt.Errorf("%w: package %q call target %q is not self or a direct dependency", ErrWorkspaceJoinIncomplete, header.PackageInstance, relation.Target.PackageInstance)
				}
			}
			targetIndex, found := packageIndexes[relation.Target.PackageInstance]
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
			if relation.Target.Specialization != "" {
				found := false
				for _, implementation := range targetNode.specializations {
					if implementation.ID == relation.Target.Specialization {
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("%w: call target implementation is absent from its exact declaration", ErrWorkspaceJoinIncomplete)
				}
			} else if len(targetNode.declaration.TypeParameters) != 0 {
				return nil, fmt.Errorf("%w: generic call target has no concrete implementation", ErrWorkspaceJoinIncomplete)
			}
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
				SourceSpecialization: relation.SourceSpecialization,
				TargetSpecialization: relation.Target.Specialization,
				Source:               WorkspaceSymbol{PackageInstance: header.PackageInstance, Symbol: packageIndex.nodes[sourcePosition].symbol.Visible},
				Target:               WorkspaceSymbol{PackageInstance: relation.Target.PackageInstance, Symbol: targetNode.symbol.Visible},
				Span:                 *relation.Span, Certainty: relation.Certainty, Provenance: relation.Provenance,
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
	packageState, found := index.packages[packageInstance]
	if !found {
		return QueryContext{}, fmt.Errorf("Atlas workspace call package %q is unknown", packageInstance)
	}
	return cloneQueryContext(packageState.context), nil
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
	packageState, found := index.packages[packageInstance]
	if !found {
		return WorkspaceSymbol{}, fmt.Errorf("Atlas workspace call package %q is unknown", packageInstance)
	}
	visible, err := packageState.resolveSymbol(symbolID)
	if err != nil {
		return WorkspaceSymbol{}, err
	}
	return WorkspaceSymbol{PackageInstance: packageInstance, Symbol: visible}, nil
}

func compactWorkspaceCallPackage(index *QueryIndex) workspaceCallPackage {
	result := workspaceCallPackage{
		context:   index.contextFor([]RelationKind{RelationCalls}),
		byVisible: make(map[string]struct{}, len(index.nodes)),
		byFull:    make(map[string]string, len(index.nodes)),
	}
	for _, node := range index.nodes {
		result.byVisible[node.symbol.Visible] = struct{}{}
		result.byFull[node.symbol.Full] = node.symbol.Visible
	}
	return result
}

func (state workspaceCallPackage) resolveSymbol(value string) (string, error) {
	if strings.HasPrefix(value, symbolidentity.VisiblePrefix) {
		if _, err := symbolidentity.ParseVisibleID(value); err != nil {
			return "", fmt.Errorf("parse Atlas visible symbol: %w", err)
		}
		if _, found := state.byVisible[value]; found {
			return value, nil
		}
		return "", fmt.Errorf("%w: %s", ErrSymbolNotFound, value)
	}
	if strings.HasPrefix(value, symbolidentity.DigestPrefix) {
		if _, err := symbolidentity.ParseFullDigest(value); err != nil {
			return "", fmt.Errorf("parse Atlas full symbol: %w", err)
		}
		if visible, found := state.byFull[value]; found {
			return visible, nil
		}
		return "", fmt.Errorf("%w: %s", ErrSymbolNotFound, value)
	}
	return "", fmt.Errorf("Atlas symbol selector must be a canonical visible or full SymbolId")
}

func (index *WorkspaceCallIndex) selectCalls(positions []int) []JoinedCall {
	result := make([]JoinedCall, 0, len(positions))
	for _, position := range positions {
		result = append(result, index.calls[position])
	}
	return result
}
