package semantic

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

const resourceAbilitiesAttribute = "abilities"

type resourceAbility uint8

const (
	resourceAbilityDrop resourceAbility = 1 << iota
	resourceAbilityStore
	resourceAbilityKey
)

type resourceAbilityDeclaration struct {
	abilities resourceAbility
	span      diagnostic.Span
	explicit  bool
	valid     bool
}

func (d resourceAbilityDeclaration) has(ability resourceAbility) bool {
	return d.abilities&ability != 0
}

func (d resourceAbilityDeclaration) canonicalHIR() []hir.ResourceAbility {
	result := make([]hir.ResourceAbility, 0, 3)
	if d.has(resourceAbilityDrop) {
		result = append(result, hir.ResourceAbilityDrop)
	}
	if d.has(resourceAbilityStore) {
		result = append(result, hir.ResourceAbilityStore)
	}
	if d.has(resourceAbilityKey) {
		result = append(result, hir.ResourceAbilityKey)
	}
	return result
}

// cloneResourceAbilityInputs preserves the semantic input contract as immutable while allowing the
// analyzer to consume recognized resource-owned attributes on its private resource declarations.
func cloneResourceAbilityInputs(files []File) []File {
	cloned := slices.Clone(files)
	for index := range cloned {
		module := cloned[index].Module
		if module == nil || len(module.Resources) == 0 {
			continue
		}
		moduleClone := *module
		moduleClone.Resources = slices.Clone(module.Resources)
		for resourceIndex := range moduleClone.Resources {
			moduleClone.Resources[resourceIndex].Attributes = slices.Clone(moduleClone.Resources[resourceIndex].Attributes)
		}
		cloned[index].Module = &moduleClone
	}
	return cloned
}

// prepareResourceAbilities consumes the resource-owned @abilities attribute before the generic
// item-attribute fallback runs. Parsed declarations remain attached to the canonical resource type
// so Flow can reject compiler-scheduled destruction without leaking syntax details into later passes.
func (a *analyzer) prepareResourceAbilities() {
	if !a.resourceAbilities {
		return
	}
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			value := a.resources[resource]
			if value == nil {
				continue
			}
			info := a.resourceTypes[value.typ]
			if info == nil {
				continue
			}
			declaration, retained := a.parseResourceAbilityDeclaration(module.file.Path, resource)
			info.abilities = declaration
			resource.Attributes = retained
		}
	}
}

func (a *analyzer) parseResourceAbilityDeclaration(source string, resource *ast.Resource) (resourceAbilityDeclaration, []ast.Attribute) {
	declaration := resourceAbilityDeclaration{
		abilities: resourceAbilityDrop,
		valid:     true,
	}
	retained := make([]ast.Attribute, 0, len(resource.Attributes))
	seenAbilities := make(map[resourceAbility]diagnostic.Span)

	for _, attribute := range resource.Attributes {
		if attribute.Name != resourceAbilitiesAttribute {
			retained = append(retained, attribute)
			continue
		}
		if declaration.explicit {
			declaration.valid = false
			a.addDiagnostic("E1107", span(source, attribute.Span), RelatedSpan{
				Role: "ability-declaration", Span: span(source, declaration.span),
			})
			continue
		}

		declaration.explicit = true
		declaration.span = attribute.Span
		declaration.abilities = 0
		if len(attribute.Arguments) == 0 {
			declaration.valid = false
			a.addDiagnostic("E1107", span(source, attribute.Span), RelatedSpan{
				Role: "resource-declaration", Span: span(source, resource.NameSpan),
			})
			continue
		}

		for _, argument := range attribute.Arguments {
			ability, valid := parseResourceAbility(argument.Raw)
			if !valid {
				declaration.valid = false
				a.addDiagnostic("E1107", span(source, argument.Span), RelatedSpan{
					Role: "ability-declaration", Span: span(source, attribute.Span),
				})
				continue
			}
			if previous, duplicate := seenAbilities[ability]; duplicate {
				declaration.valid = false
				a.addDiagnostic("E1107", span(source, argument.Span), RelatedSpan{
					Role: "previous-ability", Span: span(source, previous),
				})
				continue
			}
			seenAbilities[ability] = argument.Span
			declaration.abilities |= ability
		}

		if keySpan, hasKey := seenAbilities[resourceAbilityKey]; hasKey && !declaration.has(resourceAbilityStore) {
			declaration.valid = false
			a.addDiagnostic("E1107", span(source, keySpan), RelatedSpan{
				Role: "ability-declaration", Span: span(source, attribute.Span),
			})
		}
	}

	return declaration, retained
}

func parseResourceAbility(raw string) (resourceAbility, bool) {
	switch raw {
	case "drop":
		return resourceAbilityDrop, true
	case "store":
		return resourceAbilityStore, true
	case "key":
		return resourceAbilityKey, true
	default:
		return 0, false
	}
}

func (a *analyzer) resourceAllowsImplicitDrop(owner *symbol) (bool, diagnostic.Span) {
	if owner == nil || owner.resource == nil {
		return true, diagnostic.Span{}
	}
	info := a.resourceTypes[owner.typ]
	if info == nil || !info.abilities.valid {
		return true, owner.resource.NameSpan
	}
	if !info.abilities.explicit {
		return true, owner.resource.NameSpan
	}
	return info.abilities.has(resourceAbilityDrop), info.abilities.span
}

// nonDroppableResourceOwners follows compiler-owned aggregate cleanup recursively. This prevents a
// droppable wrapper resource from becoming a laundering path that implicitly destroys an inner
// linear asset whose own declaration omits drop.
func (a *analyzer) nonDroppableResourceOwners(value hir.Type, visiting map[hir.Type]struct{}) []*symbol {
	if info := a.resourceTypes[value]; info != nil {
		if allowed, _ := a.resourceAllowsImplicitDrop(info.symbol); !allowed {
			return []*symbol{info.symbol}
		}
		if _, cycle := visiting[value]; cycle {
			return nil
		}
		visiting[value] = struct{}{}
		var owners []*symbol
		for _, field := range info.orderedFields {
			owners = append(owners, a.nonDroppableResourceOwners(field.typ, visiting)...)
		}
		delete(visiting, value)
		return owners
	}
	if element, option := hir.OptionElement(value); option {
		return a.nonDroppableResourceOwners(element, visiting)
	}
	if success, failure, result := hir.ResultElements(value); result {
		owners := a.nonDroppableResourceOwners(success, visiting)
		return append(owners, a.nonDroppableResourceOwners(failure, visiting)...)
	}
	return nil
}

// rejectDiscardedLinearResource covers resource-bearing temporaries that never receive a tracked
// local. Ordinary Resource Flow owns locals and parameters; this companion check prevents direct
// expression statements and `discard make_asset()` from scheduling an untracked implicit drop.
func (a *analyzer) rejectDiscardedLinearResource(module *moduleScope, expression *ast.Expression) {
	if expression == nil || resourcePlaceExpression(expression) {
		return
	}
	value := a.expressionTypes[expression]
	if !a.isResource(value) {
		return
	}
	owners := a.nonDroppableResourceOwners(value, make(map[hir.Type]struct{}))
	if len(owners) == 0 {
		return
	}
	at := span(module.file.Path, expression.Span)
	seen := make(map[*symbol]struct{}, len(owners))
	for _, owner := range owners {
		if owner == nil {
			continue
		}
		if _, duplicate := seen[owner]; duplicate {
			continue
		}
		seen[owner] = struct{}{}
		_, declaration := a.resourceAllowsImplicitDrop(owner)
		a.addDiagnostic("E1108", at, RelatedSpan{
			Role: "ability-declaration", Span: span(owner.source, declaration),
		})
	}
}

func resourcePlaceExpression(expression *ast.Expression) bool {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	return expression != nil && expression.Kind == syntax.KindPath
}
