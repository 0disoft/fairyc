package atlasmap

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

type DiffChange struct {
	Module string                           `json:"module"`
	Kind   semanticsnapshot.DeclarationKind `json:"kind"`
	Name   string                           `json:"name"`
	Action string                           `json:"action"`
	Facts  []string                         `json:"facts,omitempty"`
}

type DiffResult struct {
	Schema    string       `json:"schema"`
	State     string       `json:"state"`
	Before    QueryContext `json:"before"`
	After     QueryContext `json:"after"`
	Changes   []DiffChange `json:"changes"`
	Omissions []string     `json:"omissions"`
}

type diffKey struct {
	module string
	kind   semanticsnapshot.DeclarationKind
	name   string
}

// CompareV3 compares bounded, exact-current facts for the same package
// coordinate. Source offsets and source-dependent instance IDs are excluded
// from equality, and changed names remain separate additions and removals.
func CompareV3(before, after Partition, expectedBefore, expectedAfter LoadExpectation, maxDeclarations int) (DiffResult, error) {
	if maxDeclarations < 1 {
		return DiffResult{}, fmt.Errorf("invalid Atlas diff declaration limit")
	}
	old, new := before.data.header, after.data.header
	if old.LogicalSchema != LogicalSchemaV3 || new.LogicalSchema != LogicalSchemaV3 ||
		old.PackageCoordinate != new.PackageCoordinate || old.Language != new.Language ||
		old.CompilerBuild != new.CompilerBuild || old.Target != new.Target ||
		old.TargetScope != new.TargetScope || old.RuntimeABI != new.RuntimeABI {
		return DiffResult{}, fmt.Errorf("Atlas diff requires matching package, compiler and target identities")
	}
	if err := compatibilityError(before, expectedBefore); err != nil {
		return DiffResult{}, fmt.Errorf("Atlas diff baseline: %w", err)
	}
	if err := compatibilityError(after, expectedAfter); err != nil {
		return DiffResult{}, fmt.Errorf("Atlas diff current: %w", err)
	}
	if len(before.data.declarations) > maxDeclarations ||
		len(after.data.declarations) > maxDeclarations-len(before.data.declarations) {
		return DiffResult{}, fmt.Errorf("Atlas diff declarations exceed limit")
	}

	result := DiffResult{
		Schema: "fairyc.atlas-diff/v1", State: "bounded",
		Before: diffContext(before), After: diffContext(after),
		Changes:   []DiffChange{},
		Omissions: []string{"behavior-equivalence-not-analyzed", "renames-not-inferred"},
	}
	for _, context := range []QueryContext{result.Before, result.After} {
		if context.Generation != GenerationExact || len(context.MapOmissions) != 0 {
			result.State = "partial"
		}
		for _, fact := range context.Completeness {
			if fact.Status != semanticsnapshot.CompletenessExact || fact.External != 0 || fact.Unknown != 0 {
				result.State = "partial"
			}
		}
	}
	if len(before.data.specializations) != 0 || len(after.data.specializations) != 0 {
		result.State = "partial"
		result.Omissions = append(result.Omissions, "specialized-implementation-deltas-not-compared")
	}

	keys := make(map[diffKey]struct{})
	left, right := diffDeclarations(before, keys), diffDeclarations(after, keys)
	leftBodies, rightBodies := diffBodies(before), diffBodies(after)
	for key := range keys {
		a, aok := left[key]
		b, bok := right[key]
		change := DiffChange{Module: key.module, Kind: key.kind, Name: key.name, Action: "modified"}
		switch {
		case !aok:
			change.Action = "added"
		case !bok:
			change.Action = "removed"
		default:
			if !reflect.DeepEqual(a, b) {
				change.Facts = append(change.Facts, "declaration")
			}
			if !reflect.DeepEqual(leftBodies[key].calls, rightBodies[key].calls) {
				change.Facts = append(change.Facts, "calls")
			}
			if !reflect.DeepEqual(leftBodies[key].observations, rightBodies[key].observations) {
				change.Facts = append(change.Facts, "observations")
			}
			if len(change.Facts) == 0 {
				continue
			}
		}
		result.Changes = append(result.Changes, change)
	}
	slices.SortFunc(result.Changes, func(a, b DiffChange) int {
		if n := cmp.Compare(a.Module, b.Module); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Kind, b.Kind); n != 0 {
			return n
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return result, nil
}

func diffContext(partition Partition) QueryContext {
	h := partition.Header()
	return QueryContext{
		LogicalSchema: h.LogicalSchema, CompilerBuild: h.CompilerBuild, Language: h.Language,
		PackageUID: h.PackageUID, PackageInstance: h.PackageInstance, PackageCoordinate: h.PackageCoordinate,
		PackageSource: h.PackageSource, InterfaceHash: h.InterfaceHash, GenericInterfaceHash: h.GenericInterfaceHash,
		Generation: h.GenerationStatus, MapOmissions: h.Omissions, Completeness: partition.Completeness(),
	}
}

func diffDeclarations(partition Partition, keys map[diffKey]struct{}) map[diffKey]semanticsnapshot.Declaration {
	result := make(map[diffKey]semanticsnapshot.Declaration, len(partition.data.declarations))
	for _, declaration := range partition.Declarations() {
		key := diffKey{declaration.Key.Module, declaration.Key.Kind, declaration.Key.Name}
		keys[key] = struct{}{}
		declaration.Key.PackageInstance = ""
		declaration.Span = semanticsnapshot.Span{}
		for i := range declaration.Parameters {
			declaration.Parameters[i].Span = semanticsnapshot.Span{}
		}
		for i := range declaration.Fields {
			declaration.Fields[i].Span = semanticsnapshot.Span{}
		}
		for i := range declaration.Variants {
			variant := &declaration.Variants[i]
			variant.Span = semanticsnapshot.Span{}
			for j := range variant.Fields {
				variant.Fields[j].Span = semanticsnapshot.Span{}
			}
		}
		result[key] = declaration
	}
	return result
}

type bodyDiffFacts struct {
	calls        []semanticsnapshot.Call
	observations bodyObservations
}

type bodyObservations struct {
	traps                                     []string
	borrows                                   []semanticsnapshot.Borrow
	unsafe, moves, drops, allocations, errors int
}

func diffBodies(partition Partition) map[diffKey]bodyDiffFacts {
	coordinates := make(map[string]string)
	for _, ref := range partition.data.header.PackageReferences {
		coordinates[ref.Instance] = ref.Coordinate
	}
	result := make(map[diffKey]bodyDiffFacts, len(partition.data.functions))
	for _, body := range partition.Functions() {
		calls := body.Calls
		for i := range calls {
			calls[i].Span = semanticsnapshot.Span{}
			if coordinate, found := coordinates[calls[i].Target.PackageInstance]; found {
				calls[i].Target.PackageInstance = coordinate
			}
		}
		for i := range body.Borrows {
			body.Borrows[i].Span = semanticsnapshot.Span{}
		}
		key := diffKey{body.Function.Module, body.Function.Kind, body.Function.Name}
		result[key] = bodyDiffFacts{calls: calls, observations: bodyObservations{
			traps: body.TrapCategories, borrows: body.Borrows,
			unsafe: len(body.UnsafeSites), moves: len(body.MoveSites), drops: len(body.DropSites),
			allocations: len(body.AllocationSites), errors: len(body.ReturnErrorSites),
		}}
	}
	return result
}
