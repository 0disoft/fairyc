package interfacereview

import (
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/moduleinterface"
)

type Change struct {
	State          State                        `json:"state"`
	Module         string                       `json:"module,omitempty"`
	Kind           string                       `json:"kind"`
	Name           string                       `json:"name"`
	Action         string                       `json:"action"`
	Fields         []string                     `json:"fields,omitempty"`
	EffectsAdded   []string                     `json:"effectsAdded,omitempty"`
	EffectsRemoved []string                     `json:"effectsRemoved,omitempty"`
	Before         *moduleinterface.Item        `json:"before,omitempty"`
	After          *moduleinterface.Item        `json:"after,omitempty"`
	BeforeLayout   *moduleinterface.StateLayout `json:"beforeLayout,omitempty"`
	AfterLayout    *moduleinterface.StateLayout `json:"afterLayout,omitempty"`
}

type Report struct {
	Schema                string   `json:"schema"`
	State                 State    `json:"state"`
	Changed               *bool    `json:"changed"`
	BehaviorCompatibility State    `json:"behaviorCompatibility"`
	Before                Snapshot `json:"before"`
	After                 Snapshot `json:"after"`
	Changes               []Change `json:"changes"`
	Issues                []Issue  `json:"issues"`
}

func Compare(before, after Snapshot) Report {
	result := Report{Schema: "fairyc.interface-review/v1", State: Unknown, BehaviorCompatibility: Unknown, Before: before, After: after, Changes: []Change{}, Issues: []Issue{}}
	if before.Descriptor == nil || after.Descriptor == nil || before.State == Unknown || after.State == Unknown {
		return result
	}
	if before.Package.UID == "" || before.Package.UID != after.Package.UID || before.Package.Language != after.Package.Language || !slices.Equal(before.Package.Features, after.Package.Features) {
		result.Issues = append(result.Issues, Issue{Reason: "different-package-language-or-feature-context"})
		return result
	}
	for _, snapshot := range []Snapshot{before, after} {
		hash, err := moduleinterface.Compute(*snapshot.Descriptor)
		if err != nil || snapshot.ProjectionHash != hash.String() {
			result.Issues = append(result.Issues, Issue{Reason: "invalid-interface-projection"})
			return result
		}
	}
	blocked := map[string]bool{}
	for _, snapshot := range []Snapshot{before, after} {
		for _, issue := range snapshot.Issues {
			if issue.Module != "" {
				blocked[issue.Module] = true
			}
		}
	}
	left, right := map[string]moduleinterface.Module{}, map[string]moduleinterface.Module{}
	for _, module := range before.Descriptor.Modules {
		if !blocked[module.Path] {
			left[module.Path] = module
		}
	}
	for _, module := range after.Descriptor.Modules {
		if !blocked[module.Path] {
			right[module.Path] = module
		}
	}
	if len(left) == 0 && len(right) == 0 {
		return result
	}
	for _, path := range unionKeys(left, right) {
		a, aok := left[path]
		b, bok := right[path]
		if !aok || !bok {
			action := "added"
			if !bok {
				action = "removed"
			}
			result.Changes = append(result.Changes, Change{State: Exact, Module: path, Kind: "module", Name: path, Action: action})
		}
		itemsA, itemsB := map[string]moduleinterface.Item{}, map[string]moduleinterface.Item{}
		for _, item := range a.Items {
			itemsA[item.Kind+"\x00"+item.Name] = item
		}
		for _, item := range b.Items {
			itemsB[item.Kind+"\x00"+item.Name] = item
		}
		for _, key := range unionKeys(itemsA, itemsB) {
			a, aok := itemsA[key]
			b, bok := itemsB[key]
			change := Change{State: Exact, Module: path, Kind: b.Kind, Name: b.Name, Action: "modified"}
			if aok {
				change.Before = &a
			}
			if bok {
				change.After = &b
			}
			switch {
			case !aok:
				change.Action = "added"
			case !bok:
				change.Action = "removed"
				change.Kind, change.Name = a.Kind, a.Name
			default:
				change.Fields = changedFields(a, b)
				if len(change.Fields) == 0 {
					continue
				}
			}
			change.EffectsAdded = difference(b.Effects, a.Effects)
			change.EffectsRemoved = difference(a.Effects, b.Effects)
			result.Changes = append(result.Changes, change)
		}
	}
	layoutsA, layoutsB := map[string]moduleinterface.StateLayout{}, map[string]moduleinterface.StateLayout{}
	for _, layout := range before.Descriptor.StateLayouts {
		layoutsA[layout.ContractUID] = layout
	}
	for _, layout := range after.Descriptor.StateLayouts {
		layoutsB[layout.ContractUID] = layout
	}
	for _, uid := range unionKeys(layoutsA, layoutsB) {
		a, aok := layoutsA[uid]
		b, bok := layoutsB[uid]
		if aok && bok && a.Hash == b.Hash {
			continue
		}
		change := Change{State: Exact, Kind: "state-layout", Name: uid, Action: "modified", Fields: []string{"hash"}}
		if aok {
			change.BeforeLayout = &a
		}
		if bok {
			change.AfterLayout = &b
		}
		if !aok {
			change.Action = "added"
		}
		if !bok {
			change.Action = "removed"
		}
		result.Changes = append(result.Changes, change)
	}
	slices.SortFunc(result.Changes, func(a, b Change) int {
		return strings.Compare(a.Module+"\x00"+a.Kind+"\x00"+a.Name+"\x00"+a.Action, b.Module+"\x00"+b.Kind+"\x00"+b.Name+"\x00"+b.Action)
	})
	result.State = Bounded
	if before.State == Partial || after.State == Partial {
		result.State = Partial
	}
	changed := len(result.Changes) != 0
	if changed || result.State == Bounded {
		result.Changed = &changed
	}
	return result
}

func changedFields(a, b moduleinterface.Item) []string {
	var fields []string
	for _, item := range []struct {
		name      string
		different bool
	}{
		{"c-enum-base", a.CBase != b.CBase}, {"repr-c", a.ReprC != b.ReprC}, {"unsafe", a.Unsafe != b.Unsafe},
		{"must-use", a.MustUse != b.MustUse}, {"abi", a.ABI != b.ABI}, {"link-name", a.LinkName != b.LinkName},
		{"owner-type", a.OwnerType != b.OwnerType}, {"effects", !slices.Equal(a.Effects, b.Effects)},
		{"abilities", !slices.Equal(a.Abilities, b.Abilities)}, {"parameters", !slices.Equal(a.Parameters, b.Parameters)},
		{"result-type", a.ResultType != b.ResultType}, {"fields", !slices.Equal(a.Fields, b.Fields)},
		{"variants", !slices.EqualFunc(a.Variants, b.Variants, func(a, b moduleinterface.Variant) bool { return a.Name == b.Name && slices.Equal(a.Fields, b.Fields) })},
	} {
		if item.different {
			fields = append(fields, item.name)
		}
	}
	return fields
}

func unionKeys[T any](a, b map[string]T) []string {
	keys := make([]string, 0, len(a)+len(b))
	for key := range a {
		keys = append(keys, key)
	}
	for key := range b {
		if _, ok := a[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func difference(a, b []string) []string {
	var result []string
	for _, value := range a {
		if !slices.Contains(b, value) {
			result = append(result, value)
		}
	}
	return result
}
