package atlasmap

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

var (
	ErrSymbolNotFound      = errors.New("Atlas symbol was not found in the current package map")
	ErrModuleNotFound      = errors.New("Atlas module was not found in the current package map")
	ErrQueryBudgetTooSmall = errors.New("Atlas query budget cannot contain the required subject")
)

type RelationKind string

const (
	RelationAllocates      RelationKind = "allocates"
	RelationBorrowsMutable RelationKind = "borrows-mutable"
	RelationBorrowsShared  RelationKind = "borrows-shared"
	RelationCalls          RelationKind = "calls"
	RelationCallsForeign   RelationKind = "calls-foreign"
	RelationDrops          RelationKind = "drops"
	RelationMayTrap        RelationKind = "may-trap"
	RelationMoves          RelationKind = "moves"
	RelationReturnsError   RelationKind = "returns-error"
	RelationUsesUnsafe     RelationKind = "uses-unsafe"
)

var supportedRelationKinds = []RelationKind{
	RelationAllocates,
	RelationBorrowsMutable,
	RelationBorrowsShared,
	RelationCalls,
	RelationCallsForeign,
	RelationDrops,
	RelationMayTrap,
	RelationMoves,
	RelationReturnsError,
	RelationUsesUnsafe,
}

type QueryDirection string

const (
	DirectionOutgoing QueryDirection = "outgoing"
	DirectionIncoming QueryDirection = "incoming"
	DirectionBoth     QueryDirection = "both"
)

const (
	OmissionBudgetBytes               = "selection-byte-budget"
	OmissionBudgetRecords             = "record-budget"
	OmissionBudgetRelations           = "relation-budget"
	OmissionBudgetSymbols             = "symbol-budget"
	OmissionDependencyMapNotJoined    = "dependency-map-not-joined"
	OmissionDepthLimit                = "depth-limit"
	OmissionIncomingPackagesNotJoined = "incoming-package-maps-not-joined"
	OmissionRelationFilter            = "relation-filter"
)

// QueryBudget is an explicit caller policy. SelectionBytes is the sum of compact JSON bytes for
// records admitted in deterministic selection order, plus one record separator byte per record. It
// is not a provider-token estimate or a promise about a later transport wrapper.
type QueryBudget struct {
	MaxRecords        int    `json:"max_records"`
	MaxRelations      int    `json:"max_relations"`
	MaxDepth          int    `json:"max_depth"`
	MaxSelectionBytes uint64 `json:"max_selection_bytes"`
}

type QueryUsage struct {
	Records        int    `json:"records"`
	Relations      int    `json:"relations"`
	Depth          int    `json:"depth"`
	SelectionBytes uint64 `json:"selection_bytes"`
}

type QueryOmission struct {
	Code       string         `json:"code"`
	Section    string         `json:"section,omitempty"`
	Relations  []RelationKind `json:"relations,omitempty"`
	Count      int            `json:"count,omitempty"`
	CountKnown bool           `json:"count_known"`
}

type RelationRequest struct {
	Direction QueryDirection `json:"direction"`
	Relations []RelationKind `json:"relations"`
	Budget    QueryBudget    `json:"budget"`
}

type RelationKindCount struct {
	Kind  RelationKind `json:"kind"`
	Count int          `json:"count"`
}

type DeclarationKindCount struct {
	Kind  semanticsnapshot.DeclarationKind `json:"kind"`
	Count int                              `json:"count"`
}

type SymbolCatalogBudget struct {
	MaxSymbols        int    `json:"max_symbols"`
	MaxSelectionBytes uint64 `json:"max_selection_bytes"`
}

type SymbolCatalogUsage struct {
	Symbols        int    `json:"symbols"`
	SelectionBytes uint64 `json:"selection_bytes"`
}

type SymbolCatalogRequest struct {
	Module string                             `json:"module,omitempty"`
	Name   string                             `json:"name,omitempty"`
	Kinds  []semanticsnapshot.DeclarationKind `json:"kinds"`
	After  string                             `json:"after,omitempty"`
	Budget SymbolCatalogBudget                `json:"budget"`
}

type SymbolCatalogEntry struct {
	Symbol  Symbol                           `json:"symbol"`
	Module  string                           `json:"module"`
	Kind    semanticsnapshot.DeclarationKind `json:"kind"`
	Name    string                           `json:"name"`
	Public  bool                             `json:"public"`
	Unsafe  bool                             `json:"unsafe"`
	Foreign bool                             `json:"foreign"`
}

type SymbolCatalogResult struct {
	Context   QueryContext                       `json:"context"`
	Module    string                             `json:"module,omitempty"`
	Name      string                             `json:"name,omitempty"`
	Kinds     []semanticsnapshot.DeclarationKind `json:"kinds"`
	After     string                             `json:"after,omitempty"`
	Total     int                                `json:"total"`
	Symbols   []SymbolCatalogEntry               `json:"symbols"`
	NextAfter string                             `json:"next_after,omitempty"`
	Complete  bool                               `json:"complete"`
	Budget    SymbolCatalogBudget                `json:"budget"`
	Usage     SymbolCatalogUsage                 `json:"usage"`
	Omissions []QueryOmission                    `json:"omissions"`
}

type QueryFunctionFacts struct {
	Provenance semanticsnapshot.Provenance `json:"provenance"`
	Relations  []RelationKindCount         `json:"relations"`
}

type QueryNode struct {
	TypeParameters  []string                     `json:"type_parameters,omitempty"`
	Specializations []QuerySpecializationFacts   `json:"specializations,omitempty"`
	Symbol          Symbol                       `json:"symbol"`
	Declaration     semanticsnapshot.Declaration `json:"declaration"`
	Function        *QueryFunctionFacts          `json:"function,omitempty"`
}

type QueryTarget struct {
	Specialization  string                           `json:"specialization,omitempty"`
	Symbol          string                           `json:"symbol,omitempty"`
	PackageInstance string                           `json:"package_instance"`
	Module          string                           `json:"module"`
	Kind            semanticsnapshot.DeclarationKind `json:"kind"`
	Name            string                           `json:"name"`
}

type QueryRelation struct {
	SourceSpecialization string                      `json:"source_specialization,omitempty"`
	Source               string                      `json:"source"`
	Kind                 RelationKind                `json:"kind"`
	Target               *QueryTarget                `json:"target,omitempty"`
	Span                 *semanticsnapshot.Span      `json:"span,omitempty"`
	Certainty            semanticsnapshot.Certainty  `json:"certainty,omitempty"`
	Provenance           semanticsnapshot.Provenance `json:"provenance"`
	Detail               string                      `json:"detail,omitempty"`
	ABI                  string                      `json:"abi,omitempty"`
	LinkName             string                      `json:"link_name,omitempty"`
	Escapes              bool                        `json:"escapes,omitempty"`
}

type QueryContext struct {
	GenericInterfaceHash string                          `json:"generic_interface_hash,omitempty"`
	LogicalSchema        string                          `json:"logical_schema"`
	CompilerBuild        string                          `json:"compiler_build"`
	Language             string                          `json:"language"`
	PackageUID           string                          `json:"package_uid"`
	PackageInstance      string                          `json:"package_instance"`
	PackageCoordinate    string                          `json:"package_coordinate"`
	PackageSource        string                          `json:"package_source"`
	InterfaceHash        string                          `json:"interface_hash"`
	Generation           string                          `json:"generation"`
	MapOmissions         []string                        `json:"map_omissions"`
	Completeness         []semanticsnapshot.Completeness `json:"completeness"`
}

type PackageOverview struct {
	Header           Header                 `json:"header"`
	DeclarationKinds []DeclarationKindCount `json:"declaration_kinds"`
}

type OverviewResult struct {
	Package      PackageOverview                 `json:"package"`
	Completeness []semanticsnapshot.Completeness `json:"completeness"`
	Modules      []semanticsnapshot.Module       `json:"modules"`
	Budget       QueryBudget                     `json:"budget"`
	Usage        QueryUsage                      `json:"usage"`
	Omissions    []QueryOmission                 `json:"omissions"`
}

type ShowResult struct {
	Context       QueryContext    `json:"context"`
	Subject       QueryNode       `json:"subject"`
	Direction     QueryDirection  `json:"direction"`
	RelationKinds []RelationKind  `json:"relation_kinds"`
	Relations     []QueryRelation `json:"relations"`
	Budget        QueryBudget     `json:"budget"`
	Usage         QueryUsage      `json:"usage"`
	Omissions     []QueryOmission `json:"omissions"`
}

type AroundNode struct {
	Depth int       `json:"depth"`
	Node  QueryNode `json:"node"`
}

type AroundResult struct {
	Context       QueryContext    `json:"context"`
	Subject       string          `json:"subject"`
	Direction     QueryDirection  `json:"direction"`
	RelationKinds []RelationKind  `json:"relation_kinds"`
	Nodes         []AroundNode    `json:"nodes"`
	Relations     []QueryRelation `json:"relations"`
	Budget        QueryBudget     `json:"budget"`
	Usage         QueryUsage      `json:"usage"`
	Omissions     []QueryOmission `json:"omissions"`
}

// AttributionSubject is a detached exact-current handoff for a downstream source-attribution
// boundary. It is not a standalone query result and contains no source text or filesystem authority.
type AttributionSubject struct {
	Context QueryContext
	Subject QueryNode
	File    semanticsnapshot.File
}

type indexedQueryNode struct {
	symbol             Symbol
	declaration        semanticsnapshot.Declaration
	functionProvenance *semanticsnapshot.Provenance
	specializations    []QuerySpecializationFacts
}

// QueryIndex is an immutable package-local index over one semantically validated, exact-current
// partition. It never discovers dependency maps, reads source text, or grants filesystem or process
// authority.
type QueryIndex struct {
	header       Header
	modules      []semanticsnapshot.Module
	completeness []semanticsnapshot.Completeness
	files        map[string]semanticsnapshot.File
	nodes        []indexedQueryNode
	relations    []QueryRelation
	byVisible    map[string]int
	byFull       map[string]int
	byKey        map[semanticsnapshot.DeclarationKey]int
	outgoing     map[string][]int
	incoming     map[string][]int
}

// AllRelationKinds returns the complete canonical relation filter supported by this query version.
func AllRelationKinds() []RelationKind {
	return slices.Clone(supportedRelationKinds)
}

// AllDeclarationKinds returns the complete canonical declaration-kind filter supported by the
// exact-current symbol catalog.
func AllDeclarationKinds() []semanticsnapshot.DeclarationKind {
	return []semanticsnapshot.DeclarationKind{
		semanticsnapshot.DeclarationEnum,
		semanticsnapshot.DeclarationForeignFunction,
		semanticsnapshot.DeclarationFunction,
		semanticsnapshot.DeclarationResource,
		semanticsnapshot.DeclarationStruct,
	}
}

func NewQueryIndex(partition Partition, expectation LoadExpectation) (*QueryIndex, error) {
	return newQueryIndex(partition, expectation, LogicalSchema)
}

func newQueryIndex(partition Partition, expectation LoadExpectation, schema string) (*QueryIndex, error) {
	if err := validatePartitionVersion(partition, schema); err != nil {
		return nil, fmt.Errorf("build Atlas query index: %w", err)
	}
	if err := validateLoadExpectation(expectation); err != nil {
		return nil, fmt.Errorf("build Atlas query index: %w", err)
	}
	if err := compatibilityError(partition, expectation); err != nil {
		return nil, fmt.Errorf("build Atlas query index: %w", err)
	}
	declarations := partition.Declarations()
	declarationsByKey := make(map[semanticsnapshot.DeclarationKey]semanticsnapshot.Declaration, len(declarations))
	for _, declaration := range declarations {
		declarationsByKey[declaration.Key] = declaration
	}
	functions := partition.Functions()
	functionProvenance := make(map[semanticsnapshot.DeclarationKey]semanticsnapshot.Provenance, len(functions))
	for _, function := range functions {
		functionProvenance[function.Function] = function.Provenance
	}
	index := &QueryIndex{
		header: partition.Header(), modules: partition.Modules(), completeness: partition.Completeness(),
		files:     make(map[string]semanticsnapshot.File, partition.Header().Counts.Files),
		byVisible: make(map[string]int, len(declarations)), byFull: make(map[string]int, len(declarations)),
		byKey:    make(map[semanticsnapshot.DeclarationKey]int, len(declarations)),
		outgoing: make(map[string][]int), incoming: make(map[string][]int),
	}
	for _, file := range partition.Files() {
		index.files[file.Path] = file
	}
	for _, symbol := range partition.Symbols() {
		declaration := declarationsByKey[symbol.Declaration]
		node := indexedQueryNode{symbol: symbol, declaration: declaration}
		if provenance, found := functionProvenance[symbol.Declaration]; found {
			value := provenance
			node.functionProvenance = &value
		}
		position := len(index.nodes)
		index.nodes = append(index.nodes, node)
		index.byVisible[symbol.Visible] = position
		index.byFull[symbol.Full] = position
		index.byKey[symbol.Declaration] = position
	}
	index.relations = index.buildRelations(functions)
	index.addSpecializations(partition.Specializations())
	slices.SortFunc(index.relations, compareQueryRelation)
	if len(index.relations) != index.header.Counts.Relations {
		return nil, fmt.Errorf("build Atlas query index: relation expansion count differs from map header")
	}
	for relationIndex, relation := range index.relations {
		index.outgoing[relation.Source] = append(index.outgoing[relation.Source], relationIndex)
		if relation.Target != nil && relation.Target.Symbol != "" {
			index.incoming[relation.Target.Symbol] = append(index.incoming[relation.Target.Symbol], relationIndex)
		}
	}
	return index, nil
}

func (index *QueryIndex) Overview(budget QueryBudget) (OverviewResult, error) {
	if err := validateQueryBudget(budget); err != nil {
		return OverviewResult{}, err
	}
	if index == nil {
		return OverviewResult{}, fmt.Errorf("Atlas query index is not initialized")
	}
	packageValue := PackageOverview{Header: cloneQueryHeader(index.header), DeclarationKinds: index.declarationKindCounts()}
	selection := newSelectionBudget(budget)
	if accepted, _, err := selection.reserveRecord(packageValue); err != nil {
		return OverviewResult{}, err
	} else if !accepted {
		return OverviewResult{}, ErrQueryBudgetTooSmall
	}
	result := OverviewResult{Package: packageValue, Budget: budget}
	for position, completeness := range index.completeness {
		accepted, code, err := selection.reserveRecord(completeness)
		if err != nil {
			return OverviewResult{}, err
		}
		if !accepted {
			result.Omissions = append(result.Omissions,
				knownOmission(code, "completeness", len(index.completeness)-position),
				knownOmission(code, "modules", len(index.modules)))
			result.Usage = selection.usage
			return result, nil
		}
		result.Completeness = append(result.Completeness, cloneCompletenessRecord(completeness))
	}
	for position, module := range index.modules {
		accepted, code, err := selection.reserveRecord(module)
		if err != nil {
			return OverviewResult{}, err
		}
		if !accepted {
			result.Omissions = append(result.Omissions, knownOmission(code, "modules", len(index.modules)-position))
			break
		}
		result.Modules = append(result.Modules, module)
	}
	result.Usage = selection.usage
	return result, nil
}

// CatalogSymbols returns an exact, full-SymbolId-ordered, statelessly pageable declaration catalog.
// An optional name is matched byte-for-byte as a Stage 0 identifier; the query performs no fuzzy
// matching and never treats a compact visible ID as a page cursor.
func (index *QueryIndex) CatalogSymbols(request SymbolCatalogRequest) (SymbolCatalogResult, error) {
	if index == nil {
		return SymbolCatalogResult{}, fmt.Errorf("Atlas query index is not initialized")
	}
	kinds, filter, err := validateSymbolCatalogRequest(index, request)
	if err != nil {
		return SymbolCatalogResult{}, err
	}
	candidates := index.catalogCandidates(request.Module, request.Name, filter)
	start := 0
	if request.After != "" {
		start = slices.IndexFunc(candidates, func(position int) bool { return index.nodes[position].symbol.Full == request.After })
		if start < 0 {
			return SymbolCatalogResult{}, fmt.Errorf("Atlas symbol catalog cursor is not present in the filtered current package")
		}
		start++
	}
	context := index.contextFor(nil)
	mandatory := struct {
		Context QueryContext                       `json:"context"`
		Module  string                             `json:"module,omitempty"`
		Name    string                             `json:"name,omitempty"`
		Kinds   []semanticsnapshot.DeclarationKind `json:"kinds"`
		After   string                             `json:"after,omitempty"`
		Total   int                                `json:"total"`
	}{
		Context: context, Module: request.Module, Name: request.Name,
		Kinds: kinds, After: request.After, Total: len(candidates),
	}
	mandatoryBytes, err := selectionRecordCost(mandatory)
	if err != nil {
		return SymbolCatalogResult{}, err
	}
	if mandatoryBytes > request.Budget.MaxSelectionBytes {
		return SymbolCatalogResult{}, ErrQueryBudgetTooSmall
	}
	result := SymbolCatalogResult{
		Context: context, Module: request.Module, Name: request.Name, Kinds: kinds, After: request.After,
		Total: len(candidates), Budget: request.Budget,
		Usage: SymbolCatalogUsage{SelectionBytes: mandatoryBytes},
	}
	for position := start; position < len(candidates); position++ {
		entry := index.catalogEntry(candidates[position])
		cost, costErr := selectionRecordCost(entry)
		if costErr != nil {
			return SymbolCatalogResult{}, costErr
		}
		if result.Usage.Symbols >= request.Budget.MaxSymbols ||
			cost > request.Budget.MaxSelectionBytes-result.Usage.SelectionBytes {
			if len(result.Symbols) == 0 {
				return SymbolCatalogResult{}, ErrQueryBudgetTooSmall
			}
			code := OmissionBudgetSymbols
			if result.Usage.Symbols < request.Budget.MaxSymbols {
				code = OmissionBudgetBytes
			}
			result.NextAfter = result.Symbols[len(result.Symbols)-1].Symbol.Full
			result.Omissions = append(result.Omissions, knownOmission(code, "symbols", len(candidates)-position))
			return result, nil
		}
		result.Symbols = append(result.Symbols, entry)
		result.Usage.Symbols++
		result.Usage.SelectionBytes += cost
	}
	result.Complete = true
	return result, nil
}

func (index *QueryIndex) Show(symbolID string, request RelationRequest) (ShowResult, error) {
	relations, filter, err := validateRelationRequest(request)
	if err != nil {
		return ShowResult{}, err
	}
	position, err := index.resolveSymbol(symbolID)
	if err != nil {
		return ShowResult{}, err
	}
	context := index.contextFor(relations)
	subject := index.node(position)
	selection := newSelectionBudget(request.Budget)
	mandatory := struct {
		Context QueryContext `json:"context"`
		Subject QueryNode    `json:"subject"`
	}{Context: context, Subject: subject}
	if accepted, _, reserveErr := selection.reserveRecord(mandatory); reserveErr != nil {
		return ShowResult{}, reserveErr
	} else if !accepted {
		return ShowResult{}, ErrQueryBudgetTooSmall
	}
	result := ShowResult{
		Context: context, Subject: subject, Direction: request.Direction, RelationKinds: relations,
		Budget: request.Budget, Omissions: filterOmissions(relations),
	}
	candidates := index.relationIDs(index.nodes[position].symbol.Visible, request.Direction, filter)
	for candidateIndex, relationIndex := range candidates {
		relation := cloneQueryRelation(index.relations[relationIndex])
		accepted, code, reserveErr := selection.reserveRelation(relation)
		if reserveErr != nil {
			return ShowResult{}, reserveErr
		}
		if !accepted {
			result.Omissions = append(result.Omissions, knownOmission(code, "relations", len(candidates)-candidateIndex))
			break
		}
		result.Relations = append(result.Relations, relation)
	}
	result.Omissions = appendJoinOmissions(result.Omissions, result.Relations, request.Direction, filter)
	result.Usage = selection.usage
	return result, nil
}

func (index *QueryIndex) Around(symbolID string, request RelationRequest) (AroundResult, error) {
	relations, filter, err := validateRelationRequest(request)
	if err != nil {
		return AroundResult{}, err
	}
	rootPosition, err := index.resolveSymbol(symbolID)
	if err != nil {
		return AroundResult{}, err
	}
	context := index.contextFor(relations)
	root := AroundNode{Depth: 0, Node: index.node(rootPosition)}
	selection := newSelectionBudget(request.Budget)
	mandatory := struct {
		Context QueryContext `json:"context"`
		Root    AroundNode   `json:"root"`
	}{Context: context, Root: root}
	if accepted, _, reserveErr := selection.reserveRecord(mandatory); reserveErr != nil {
		return AroundResult{}, reserveErr
	} else if !accepted {
		return AroundResult{}, ErrQueryBudgetTooSmall
	}
	result := AroundResult{
		Context: context, Subject: index.nodes[rootPosition].symbol.Visible, Direction: request.Direction,
		RelationKinds: relations, Nodes: []AroundNode{root}, Budget: request.Budget,
		Omissions: filterOmissions(relations),
	}
	seenNodes := map[int]struct{}{rootPosition: {}}
	seenRelations := make(map[int]struct{})
	frontier := []int{rootPosition}
	exhausted := false
	for depth := 0; depth < request.Budget.MaxDepth && len(frontier) != 0 && !exhausted; depth++ {
		candidateRelations := index.frontierRelations(frontier, request.Direction, filter, seenRelations)
		nextSet := make(map[int]struct{})
		for _, relationIndex := range candidateRelations {
			relation := cloneQueryRelation(index.relations[relationIndex])
			accepted, code, reserveErr := selection.reserveRelation(relation)
			if reserveErr != nil {
				return AroundResult{}, reserveErr
			}
			if !accepted {
				result.Omissions = append(result.Omissions, unknownOmission(code, "relations"))
				exhausted = true
				break
			}
			result.Relations = append(result.Relations, relation)
			seenRelations[relationIndex] = struct{}{}
			neighbors := index.adjacentNodes(frontier, relation, request.Direction)
			for _, neighbor := range neighbors {
				if _, seen := seenNodes[neighbor]; seen {
					continue
				}
				node := AroundNode{Depth: depth + 1, Node: index.node(neighbor)}
				accepted, code, reserveErr := selection.reserveRecord(node)
				if reserveErr != nil {
					return AroundResult{}, reserveErr
				}
				if !accepted {
					result.Omissions = append(result.Omissions, unknownOmission(code, "nodes"))
					exhausted = true
					break
				}
				seenNodes[neighbor] = struct{}{}
				nextSet[neighbor] = struct{}{}
				result.Nodes = append(result.Nodes, node)
				selection.usage.Depth = max(selection.usage.Depth, depth+1)
			}
			if exhausted {
				break
			}
		}
		frontier = sortedNodePositions(nextSet, index.nodes)
	}
	if !exhausted && len(frontier) != 0 && index.hasUnselectedExpansion(frontier, request.Direction, filter, seenRelations) {
		result.Omissions = append(result.Omissions, unknownOmission(OmissionDepthLimit, "relations"))
	}
	result.Omissions = appendJoinOmissions(result.Omissions, result.Relations, request.Direction, filter)
	result.Usage = selection.usage
	return result, nil
}

// ResolveAttributionSubject resolves either canonical SymbolId form without inventing a name search
// or requiring an unrelated relation query. The returned node and context are detached copies.
func (index *QueryIndex) ResolveAttributionSubject(symbolID string) (AttributionSubject, error) {
	position, err := index.resolveSymbol(symbolID)
	if err != nil {
		return AttributionSubject{}, err
	}
	subject := index.node(position)
	file, found := index.files[subject.Declaration.Span.Source]
	if !found {
		return AttributionSubject{}, fmt.Errorf("Atlas attribution subject references an unknown current source")
	}
	return AttributionSubject{Context: index.contextFor(nil), Subject: subject, File: file}, nil
}

func (index *QueryIndex) node(position int) QueryNode {
	value := index.nodes[position]
	result := QueryNode{Symbol: value.symbol, Declaration: cloneDeclarations([]semanticsnapshot.Declaration{value.declaration})[0]}
	result.TypeParameters = slices.Clone(value.declaration.TypeParameters)
	result.Specializations = cloneQuerySpecializations(value.specializations)
	if value.functionProvenance != nil {
		counts := make(map[RelationKind]int)
		for _, relationIndex := range index.outgoing[value.symbol.Visible] {
			counts[index.relations[relationIndex].Kind]++
		}
		facts := &QueryFunctionFacts{Provenance: *value.functionProvenance}
		for _, kind := range supportedRelationKinds {
			if count := counts[kind]; count != 0 {
				facts.Relations = append(facts.Relations, RelationKindCount{Kind: kind, Count: count})
			}
		}
		result.Function = facts
	}
	return result
}

func (index *QueryIndex) declarationKindCounts() []DeclarationKindCount {
	counts := make(map[semanticsnapshot.DeclarationKind]int)
	for _, node := range index.nodes {
		counts[node.declaration.Key.Kind]++
	}
	result := make([]DeclarationKindCount, 0, len(counts))
	for kind, count := range counts {
		result = append(result, DeclarationKindCount{Kind: kind, Count: count})
	}
	slices.SortFunc(result, func(left, right DeclarationKindCount) int { return cmp.Compare(left.Kind, right.Kind) })
	return result
}

func validateSymbolCatalogRequest(
	index *QueryIndex,
	request SymbolCatalogRequest,
) ([]semanticsnapshot.DeclarationKind, map[semanticsnapshot.DeclarationKind]struct{}, error) {
	if request.Budget.MaxSymbols <= 0 || request.Budget.MaxSelectionBytes == 0 {
		return nil, nil, fmt.Errorf("Atlas symbol catalog requires positive symbol and selection-byte limits")
	}
	if request.Module != "" {
		if !slices.ContainsFunc(index.modules, func(module semanticsnapshot.Module) bool { return module.Path == request.Module }) {
			return nil, nil, fmt.Errorf("%w: %s", ErrModuleNotFound, request.Module)
		}
	}
	if request.Name != "" && !source.ValidIdentifier(request.Name) {
		return nil, nil, fmt.Errorf("Atlas symbol catalog name must be an exact Stage 0 identifier")
	}
	kinds := slices.Clone(request.Kinds)
	if len(kinds) == 0 {
		return nil, nil, fmt.Errorf("Atlas symbol catalog requires an explicit nonempty declaration-kind filter")
	}
	slices.Sort(kinds)
	filter := make(map[semanticsnapshot.DeclarationKind]struct{}, len(kinds))
	supported := AllDeclarationKinds()
	for index, kind := range kinds {
		if index > 0 && kinds[index-1] == kind {
			return nil, nil, fmt.Errorf("Atlas symbol catalog declaration-kind filter repeats %q", kind)
		}
		if _, found := slices.BinarySearch(supported, kind); !found {
			return nil, nil, fmt.Errorf("Atlas symbol catalog declaration kind %q is unsupported", kind)
		}
		filter[kind] = struct{}{}
	}
	if request.After != "" {
		if _, err := symbolidentity.ParseFullDigest(request.After); err != nil {
			return nil, nil, fmt.Errorf("parse Atlas symbol catalog cursor: %w", err)
		}
	}
	return kinds, filter, nil
}

func (index *QueryIndex) catalogCandidates(
	module string,
	name string,
	kinds map[semanticsnapshot.DeclarationKind]struct{},
) []int {
	result := make([]int, 0, len(index.nodes))
	for position, node := range index.nodes {
		if module != "" && node.declaration.Key.Module != module {
			continue
		}
		if name != "" && node.declaration.Key.Name != name {
			continue
		}
		if _, found := kinds[node.declaration.Key.Kind]; !found {
			continue
		}
		result = append(result, position)
	}
	slices.SortFunc(result, func(left, right int) int {
		return cmp.Compare(index.nodes[left].symbol.Full, index.nodes[right].symbol.Full)
	})
	return result
}

func (index *QueryIndex) catalogEntry(position int) SymbolCatalogEntry {
	node := index.nodes[position]
	return SymbolCatalogEntry{
		Symbol: node.symbol, Module: node.declaration.Key.Module, Kind: node.declaration.Key.Kind,
		Name: node.declaration.Key.Name, Public: node.declaration.Public,
		Unsafe: node.declaration.Unsafe, Foreign: node.declaration.Foreign,
	}
}

func (index *QueryIndex) contextFor(relations []RelationKind) QueryContext {
	wanted := make(map[string]struct{})
	if index.header.LogicalSchema == LogicalSchemaV2 {
		wanted["generic-specializations"] = struct{}{}
		wanted["generic-template-bodies"] = struct{}{}
	}
	for _, relation := range relations {
		wanted[completenessRelation(relation)] = struct{}{}
	}
	context := QueryContext{
		GenericInterfaceHash: index.header.GenericInterfaceHash,
		LogicalSchema:        index.header.LogicalSchema, CompilerBuild: index.header.CompilerBuild,
		Language: index.header.Language, PackageUID: index.header.PackageUID,
		PackageInstance: index.header.PackageInstance, PackageCoordinate: index.header.PackageCoordinate,
		PackageSource: index.header.PackageSource, InterfaceHash: index.header.InterfaceHash,
		Generation: index.header.GenerationStatus, MapOmissions: slices.Clone(index.header.Omissions),
	}
	for _, completeness := range index.completeness {
		if _, found := wanted[completeness.Relation]; found {
			context.Completeness = append(context.Completeness, cloneCompletenessRecord(completeness))
		}
	}
	return context
}

func completenessRelation(kind RelationKind) string {
	switch kind {
	case RelationBorrowsMutable, RelationBorrowsShared:
		return "borrows"
	case RelationCalls, RelationCallsForeign:
		return "calls"
	default:
		return string(kind)
	}
}

func (index *QueryIndex) resolveSymbol(value string) (int, error) {
	if index == nil {
		return 0, fmt.Errorf("Atlas query index is not initialized")
	}
	if strings.HasPrefix(value, symbolidentity.VisiblePrefix) {
		if _, err := symbolidentity.ParseVisibleID(value); err != nil {
			return 0, fmt.Errorf("parse Atlas visible symbol: %w", err)
		}
		if position, found := index.byVisible[value]; found {
			return position, nil
		}
		return 0, fmt.Errorf("%w: %s", ErrSymbolNotFound, value)
	}
	if strings.HasPrefix(value, symbolidentity.DigestPrefix) {
		if _, err := symbolidentity.ParseFullDigest(value); err != nil {
			return 0, fmt.Errorf("parse Atlas full symbol: %w", err)
		}
		if position, found := index.byFull[value]; found {
			return position, nil
		}
		return 0, fmt.Errorf("%w: %s", ErrSymbolNotFound, value)
	}
	return 0, fmt.Errorf("Atlas symbol selector must be a canonical visible or full SymbolId")
}

func validateRelationRequest(request RelationRequest) ([]RelationKind, map[RelationKind]struct{}, error) {
	if err := validateQueryBudget(request.Budget); err != nil {
		return nil, nil, err
	}
	if request.Direction != DirectionOutgoing && request.Direction != DirectionIncoming && request.Direction != DirectionBoth {
		return nil, nil, fmt.Errorf("Atlas query direction %q is invalid", request.Direction)
	}
	if len(request.Relations) == 0 {
		return nil, nil, fmt.Errorf("Atlas query requires an explicit nonempty relation filter")
	}
	result := slices.Clone(request.Relations)
	slices.Sort(result)
	filter := make(map[RelationKind]struct{}, len(result))
	for index, relation := range result {
		if index > 0 && result[index-1] == relation {
			return nil, nil, fmt.Errorf("Atlas query relation filter repeats %q", relation)
		}
		if _, found := slices.BinarySearch(supportedRelationKinds, relation); !found {
			return nil, nil, fmt.Errorf("Atlas query relation %q is unsupported", relation)
		}
		filter[relation] = struct{}{}
	}
	return result, filter, nil
}

func validateQueryBudget(value QueryBudget) error {
	if value.MaxRecords <= 0 || value.MaxRelations < 0 || value.MaxDepth < 0 || value.MaxSelectionBytes == 0 {
		return fmt.Errorf("Atlas query budget requires positive records and bytes plus nonnegative relations and depth")
	}
	return nil
}

type selectionBudget struct {
	limit QueryBudget
	usage QueryUsage
}

func newSelectionBudget(limit QueryBudget) *selectionBudget {
	return &selectionBudget{limit: limit}
}

func (budget *selectionBudget) reserveRecord(value any) (bool, string, error) {
	if budget.usage.Records >= budget.limit.MaxRecords {
		return false, OmissionBudgetRecords, nil
	}
	cost, err := selectionRecordCost(value)
	if err != nil {
		return false, "", err
	}
	if cost > budget.limit.MaxSelectionBytes-budget.usage.SelectionBytes {
		return false, OmissionBudgetBytes, nil
	}
	budget.usage.Records++
	budget.usage.SelectionBytes += cost
	return true, "", nil
}

func (budget *selectionBudget) reserveRelation(value QueryRelation) (bool, string, error) {
	if budget.usage.Relations >= budget.limit.MaxRelations {
		return false, OmissionBudgetRelations, nil
	}
	accepted, code, err := budget.reserveRecord(value)
	if accepted {
		budget.usage.Relations++
	}
	return accepted, code, err
}

func selectionRecordCost(value any) (uint64, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0, fmt.Errorf("measure Atlas query record: %w", err)
	}
	return uint64(len(encoded)) + 1, nil
}

func (index *QueryIndex) relationIDs(symbol string, direction QueryDirection, filter map[RelationKind]struct{}) []int {
	result := make([]int, 0, len(index.outgoing[symbol])+len(index.incoming[symbol]))
	if direction == DirectionOutgoing || direction == DirectionBoth {
		result = append(result, index.outgoing[symbol]...)
	}
	if direction == DirectionIncoming || direction == DirectionBoth {
		result = append(result, index.incoming[symbol]...)
	}
	slices.Sort(result)
	result = slices.Compact(result)
	return slices.DeleteFunc(result, func(position int) bool {
		_, found := filter[index.relations[position].Kind]
		return !found
	})
}

func (index *QueryIndex) frontierRelations(
	frontier []int,
	direction QueryDirection,
	filter map[RelationKind]struct{},
	seen map[int]struct{},
) []int {
	set := make(map[int]struct{})
	for _, position := range frontier {
		for _, relation := range index.relationIDs(index.nodes[position].symbol.Visible, direction, filter) {
			if _, already := seen[relation]; !already {
				set[relation] = struct{}{}
			}
		}
	}
	result := make([]int, 0, len(set))
	for relation := range set {
		result = append(result, relation)
	}
	slices.Sort(result)
	return result
}

func (index *QueryIndex) adjacentNodes(frontier []int, relation QueryRelation, direction QueryDirection) []int {
	frontierSymbols := make(map[string]struct{}, len(frontier))
	for _, position := range frontier {
		frontierSymbols[index.nodes[position].symbol.Visible] = struct{}{}
	}
	set := make(map[int]struct{}, 2)
	if (direction == DirectionOutgoing || direction == DirectionBoth) && relation.Target != nil && relation.Target.Symbol != "" {
		if _, found := frontierSymbols[relation.Source]; found {
			set[index.byVisible[relation.Target.Symbol]] = struct{}{}
		}
	}
	if (direction == DirectionIncoming || direction == DirectionBoth) && relation.Target != nil && relation.Target.Symbol != "" {
		if _, found := frontierSymbols[relation.Target.Symbol]; found {
			set[index.byVisible[relation.Source]] = struct{}{}
		}
	}
	return sortedNodePositions(set, index.nodes)
}

func sortedNodePositions(values map[int]struct{}, nodes []indexedQueryNode) []int {
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.SortFunc(result, func(left, right int) int { return cmp.Compare(nodes[left].symbol.Full, nodes[right].symbol.Full) })
	return result
}

func (index *QueryIndex) hasUnselectedExpansion(
	frontier []int,
	direction QueryDirection,
	filter map[RelationKind]struct{},
	seen map[int]struct{},
) bool {
	return len(index.frontierRelations(frontier, direction, filter, seen)) != 0
}

func filterOmissions(included []RelationKind) []QueryOmission {
	excluded := make([]RelationKind, 0, len(supportedRelationKinds)-len(included))
	for _, relation := range supportedRelationKinds {
		if _, found := slices.BinarySearch(included, relation); !found {
			excluded = append(excluded, relation)
		}
	}
	if len(excluded) == 0 {
		return nil
	}
	return []QueryOmission{{Code: OmissionRelationFilter, Section: "relations", Relations: excluded, CountKnown: false}}
}

func appendJoinOmissions(
	omissions []QueryOmission,
	relations []QueryRelation,
	direction QueryDirection,
	filter map[RelationKind]struct{},
) []QueryOmission {
	unjoined := 0
	for _, relation := range relations {
		if relation.Target != nil && relation.Target.Symbol == "" {
			unjoined++
		}
	}
	if unjoined != 0 {
		omissions = append(omissions, QueryOmission{
			Code: OmissionDependencyMapNotJoined, Section: "nodes", Relations: []RelationKind{RelationCalls},
			Count: unjoined, CountKnown: true,
		})
	}
	if direction == DirectionIncoming || direction == DirectionBoth {
		if _, calls := filter[RelationCalls]; calls {
			omissions = append(omissions, QueryOmission{
				Code: OmissionIncomingPackagesNotJoined, Section: "relations", Relations: []RelationKind{RelationCalls},
				CountKnown: false,
			})
		}
	}
	return omissions
}

func knownOmission(code, section string, count int) QueryOmission {
	return QueryOmission{Code: code, Section: section, Count: count, CountKnown: true}
}

func unknownOmission(code, section string) QueryOmission {
	return QueryOmission{Code: code, Section: section, CountKnown: false}
}

func cloneCompletenessRecord(value semanticsnapshot.Completeness) semanticsnapshot.Completeness {
	value.Reasons = slices.Clone(value.Reasons)
	return value
}

func cloneQueryContext(value QueryContext) QueryContext {
	value.MapOmissions = slices.Clone(value.MapOmissions)
	value.Completeness = slices.Clone(value.Completeness)
	for position := range value.Completeness {
		value.Completeness[position] = cloneCompletenessRecord(value.Completeness[position])
	}
	return value
}

func cloneQueryHeader(value Header) Header {
	value.Features = slices.Clone(value.Features)
	value.Dependencies = slices.Clone(value.Dependencies)
	value.PackageReferences = slices.Clone(value.PackageReferences)
	value.Omissions = slices.Clone(value.Omissions)
	return value
}

func cloneQueryRelation(value QueryRelation) QueryRelation {
	if value.Target != nil {
		target := *value.Target
		value.Target = &target
	}
	if value.Span != nil {
		span := *value.Span
		value.Span = &span
	}
	return value
}

func compareQueryRelation(left, right QueryRelation) int {
	if value := cmp.Compare(left.Source, right.Source); value != 0 {
		return value
	}
	if value := cmp.Compare(left.SourceSpecialization, right.SourceSpecialization); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Kind, right.Kind); value != 0 {
		return value
	}
	if value := compareOptionalSpan(left.Span, right.Span); value != 0 {
		return value
	}
	if value := compareOptionalTarget(left.Target, right.Target); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Detail, right.Detail); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Certainty, right.Certainty); value != 0 {
		return value
	}
	if value := cmp.Compare(left.ABI, right.ABI); value != 0 {
		return value
	}
	if value := cmp.Compare(left.LinkName, right.LinkName); value != 0 {
		return value
	}
	if left.Escapes == right.Escapes {
		return 0
	}
	if !left.Escapes {
		return -1
	}
	return 1
}

func compareOptionalSpan(left, right *semanticsnapshot.Span) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}
	return compareSpan(*left, *right)
}

func compareOptionalTarget(left, right *QueryTarget) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}
	if value := cmp.Compare(left.PackageInstance, right.PackageInstance); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Module, right.Module); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Kind, right.Kind); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Name, right.Name); value != 0 {
		return value
	}
	if value := cmp.Compare(left.Symbol, right.Symbol); value != 0 {
		return value
	}
	return cmp.Compare(left.Specialization, right.Specialization)
}
