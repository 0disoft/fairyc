package genericplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func declarationOrderInput(count int) Declaration {
	value := Declaration{
		ID: "app::run", Owner: "app", Definition: DigestText("declaration", "ordering/v1"),
		Parameters: []TypeParameter{{Name: "T", Abilities: []Ability{AbilityScalar}}},
	}
	for index := range count {
		name := fmt.Sprintf("sdk::%s%02d", strings.Repeat("component_", 8), (index*13+5)%count)
		arguments := []TypeExpression{ParameterType("T"), ConcreteType("sdk::Address"), ParameterType("T")}
		value.Constraints = append(value.Constraints, CapabilityConstraint{Capability: CapabilityID(name), Arguments: slices.Clone(arguments)})
		value.Dependencies = append(value.Dependencies, Dependency{Declaration: DeclarationID(name), Arguments: slices.Clone(arguments)})
	}
	return value
}

func TestAddDeclarationPreservesNestedInputsAndCanonicalOrder(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 1, 2, DefaultCatalogLimits().MaxConstraints} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			input := declarationOrderInput(count)
			before, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			builder, err := NewBuilder(DefaultCatalogLimits())
			if err != nil {
				t.Fatal(err)
			}
			if err := builder.AddDeclaration(input); err != nil {
				t.Fatal(err)
			}
			after, err := json.Marshal(input)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("AddDeclaration changed caller input: %s, %v", after, err)
			}
			stored := builder.declarations[input.ID]
			var constraintKeys, dependencyKeys []string
			for _, constraint := range stored.Constraints {
				constraintKeys = append(constraintKeys, expressionListKey(string(constraint.Capability), constraint.Arguments))
			}
			for _, dependency := range stored.Dependencies {
				dependencyKeys = append(dependencyKeys, expressionListKey(string(dependency.Declaration), dependency.Arguments))
			}
			if len(stored.Constraints) != count || len(stored.Dependencies) != count || !slices.IsSorted(constraintKeys) || !slices.IsSorted(dependencyKeys) {
				t.Fatalf("declaration order changed: %v, %v", constraintKeys, dependencyKeys)
			}
			retained, err := json.Marshal(stored)
			if err != nil {
				t.Fatal(err)
			}
			input.Parameters[0].Abilities[0] = AbilityResource
			if count > 0 {
				input.Constraints[0].Arguments[0] = ConcreteType("sdk::Changed")
				input.Dependencies[0].Arguments[0] = ConcreteType("sdk::Changed")
			}
			current, err := json.Marshal(builder.declarations[input.ID])
			if err != nil || !bytes.Equal(retained, current) {
				t.Fatalf("builder retained caller-owned nested slices: %s, %v", current, err)
			}
		})
	}
}

func TestAddDeclarationValidationOrderAndAtomicity(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		edit func(*Builder, *Declaration)
		want string
	}{
		{"duplicate before limits and definition", func(b *Builder, d *Declaration) {
			b.declarations[d.ID] = Declaration{}
			b.limits.MaxDeclarations = 1
			d.Definition = Digest{}
		}, `declaration "app::run" is declared more than once`},
		{"definition before parameters", func(_ *Builder, d *Declaration) {
			d.Definition = Digest{}
			d.Parameters = nil
		}, `declaration "app::run" has no definition digest`},
		{"constraints before parameter contents", func(b *Builder, d *Declaration) {
			b.limits.MaxConstraints = 1
			d.Parameters[0].Name = "invalid-name"
		}, `declaration "app::run" constraint count exceeds 1`},
		{"dependencies before parameter contents", func(b *Builder, d *Declaration) {
			b.limits.MaxDependencies = 1
			d.Parameters[0].Name = "invalid-name"
		}, `declaration "app::run" dependency count exceeds 1`},
		{"parameter validation failure", func(_ *Builder, d *Declaration) {
			d.Parameters = append(d.Parameters, d.Parameters[0])
		}, `declaration "app::run" parameter "T" is duplicated`},
	} {
		t.Run(test.name, func(t *testing.T) {
			builder, err := NewBuilder(DefaultCatalogLimits())
			if err != nil {
				t.Fatal(err)
			}
			input := declarationOrderInput(2)
			test.edit(builder, &input)
			before := len(builder.declarations)
			err = builder.AddDeclaration(input)
			if err == nil || !strings.HasSuffix(err.Error(), test.want) || len(builder.declarations) != before {
				t.Fatalf("AddDeclaration = %v, entries = %d; want %s, entries = %d", err, len(builder.declarations), test.want, before)
			}
		})
	}
}

func TestDeclarationSortBuildsEachKeyOnce(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 1, 2, DefaultCatalogLimits().MaxConstraints} {
		values := make([]int, count)
		for index := range values {
			values[index] = (index*13 + 5) % count
		}
		calls := 0
		sortDeclarationEntries(values, func(value int) string {
			calls++
			return fmt.Sprintf("%02d", value)
		})
		want := count
		if count < 2 {
			want = 0
		}
		if calls != want || !slices.IsSorted(values) {
			t.Fatalf("sort %d entries: %d key builds, order %v; want %d", count, calls, values, want)
		}
	}
}

func BenchmarkAddDeclarationCanonicalization(b *testing.B) {
	input := declarationOrderInput(DefaultCatalogLimits().MaxConstraints)
	b.ReportAllocs()
	for b.Loop() {
		builder, err := NewBuilder(DefaultCatalogLimits())
		if err != nil {
			b.Fatal(err)
		}
		if err := builder.AddDeclaration(input); err != nil {
			b.Fatal(err)
		}
	}
}
