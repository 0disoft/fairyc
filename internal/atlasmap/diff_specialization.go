package atlasmap

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

// DiffSpecializationAliases joins the verified build's specialization IDs to
// source-independent comparison names. The v3 compiler admits scalar arguments.
func DiffSpecializationAliases(partitions []Partition) map[string]string {
	aliases := make(map[string]string)
	for _, partition := range partitions {
		coordinate := partition.data.header.PackageCoordinate
		for _, item := range partition.data.specializations {
			key := specializationKey(item)
			identity := strings.Join([]string{coordinate, key.module, string(key.kind), key.name, key.arguments}, "\x00")
			aliases[item.ID] = fmt.Sprintf("_fy_generic_diff_%x", sha256.Sum256([]byte(identity)))
		}
	}
	return aliases
}

type diffSpecializationKey struct {
	diffKey
	arguments string
}

func specializationKey(item semanticsnapshot.SpecializationSummary) diffSpecializationKey {
	key := item.Declaration
	names := make([]string, len(item.TypeArguments))
	for i, argument := range item.TypeArguments {
		names[i] = argument.Name
	}
	return diffSpecializationKey{diffKey{key.Module, key.Kind, key.Name}, strings.Join(names, ",")}
}

type specializationDiffFacts struct {
	bodyName  string
	body      bodyDiffFacts
	aggregate *semanticsnapshot.Declaration
}

func specializeDiffFacts(partition Partition, aliases map[string]string, keys map[diffSpecializationKey]struct{}) map[diffSpecializationKey]specializationDiffFacts {
	coordinates := diffCoordinates(partition)
	result := make(map[diffSpecializationKey]specializationDiffFacts, len(partition.data.specializations))
	for _, item := range partition.Specializations() {
		key := specializationKey(item)
		keys[key] = struct{}{}
		facts := specializationDiffFacts{aggregate: item.Aggregate}
		if key.kind == semanticsnapshot.DeclarationFunction {
			facts.body = normalizeDiffBody(item.Function, coordinates, aliases)
			facts.bodyName = key.module + "::_fy_generic_" + item.ID
		} else {
			normalizeDiffDeclaration(facts.aggregate)
			if item.Cleanup != nil {
				facts.body = normalizeDiffBody(*item.Cleanup, coordinates, aliases)
				facts.bodyName = key.module + "::$drop$_fy_generic_" + item.ID
			}
		}
		result[key] = facts
	}
	return result
}

func appendSpecializationDiff(changes []DiffChange, before, after Partition, oldAliases, newAliases map[string]string,
	compareCode func(change *DiffChange, oldName, newName string),
) []DiffChange {
	keys := make(map[diffSpecializationKey]struct{})
	left := specializeDiffFacts(before, oldAliases, keys)
	right := specializeDiffFacts(after, newAliases, keys)
	for key := range keys {
		a, aok := left[key]
		b, bok := right[key]
		change := DiffChange{Module: key.module, Kind: key.kind, Name: key.name, Action: "modified", TypeArguments: strings.Split(key.arguments, ",")}
		switch {
		case !aok:
			change.Action = "added"
		case !bok:
			change.Action = "removed"
		default:
			if !reflect.DeepEqual(a.aggregate, b.aggregate) {
				change.Facts = append(change.Facts, "declaration")
			}
			compareBodyFacts(&change, a.body, b.body)
			if a.bodyName != "" && b.bodyName != "" {
				compareCode(&change, a.bodyName, b.bodyName)
			}
			if len(change.Facts) == 0 {
				continue
			}
		}
		changes = append(changes, change)
	}
	return changes
}
