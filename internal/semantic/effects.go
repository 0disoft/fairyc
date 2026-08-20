package semantic

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

const effectsAttribute = "effects"

func (a *analyzer) prepareCallableEffects() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.ForeignFunctions {
			function := &module.file.Module.ForeignFunctions[index]
			value := a.foreignFunctions[function]
			if value == nil {
				continue
			}
			value.effects, value.effectSpan, value.effectsValid = a.parseEffects(
				module.file.Path,
				function.NameSpan,
				function.Attributes,
			)
			a.effectFacts = append(a.effectFacts, EffectFact{
				Declaration: span(module.file.Path, function.NameSpan),
				Effects:     slices.Clone(value.effects),
			})
		}
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			value := a.functions[function]
			if value == nil {
				continue
			}
			value.effects, value.effectSpan, value.effectsValid = a.parseEffects(
				module.file.Path,
				function.NameSpan,
				function.Attributes,
			)
			a.effectFacts = append(a.effectFacts, EffectFact{
				Declaration: span(module.file.Path, function.NameSpan),
				Effects:     slices.Clone(value.effects),
			})
		}
		for index := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[index]
			resourceValue := a.resources[resource]
			if resourceValue == nil {
				continue
			}
			effectAttributes := resource.Attributes
			effectDeclaration := resource.NameSpan
			if resource.Cleanup != nil && len(resource.Cleanup.Attributes) > 0 {
				effectAttributes = resource.Cleanup.Attributes
				effectDeclaration = resource.Cleanup.Span
			}
			resourceValue.effects, resourceValue.effectSpan, resourceValue.effectsValid = a.parseEffects(
				module.file.Path,
				effectDeclaration,
				effectAttributes,
			)
			a.effectFacts = append(a.effectFacts, EffectFact{
				Declaration: span(module.file.Path, resource.NameSpan),
				Effects:     slices.Clone(resourceValue.effects),
			})
			function := a.dropFunctions[resource]
			value := a.functions[function]
			if function == nil || value == nil {
				continue
			}
			value.effects = slices.Clone(resourceValue.effects)
			value.effectSpan = resourceValue.effectSpan
			value.effectsValid = resourceValue.effectsValid
		}
	}
	a.prepareResourceAbilities()
}

func (a *analyzer) parseEffects(source string, declaration diagnostic.Span, attributes []ast.Attribute) ([]hir.Effect, diagnostic.Span, bool) {
	var effects []hir.Effect
	var effectSpan diagnostic.Span
	seenAttribute := false
	validDeclaration := true
	seenEffects := make(map[hir.Effect]struct{})
	for _, attribute := range attributes {
		if attribute.Name != effectsAttribute {
			continue
		}
		if seenAttribute {
			validDeclaration = false
			a.addDiagnostic("E7101", span(source, attribute.Span), RelatedSpan{
				Role: "effect-declaration", Span: span(source, effectSpan),
			})
			continue
		}
		seenAttribute = true
		effectSpan = attribute.Span
		if len(attribute.Arguments) == 0 {
			validDeclaration = false
			a.addDiagnostic("E7101", span(source, attribute.Span), RelatedSpan{
				Role: "declaration", Span: span(source, declaration),
			})
			continue
		}
		for _, argument := range attribute.Arguments {
			effect, valid := hir.ParseEffect(argument.Raw)
			if !valid {
				validDeclaration = false
				a.addDiagnostic("E7101", span(source, argument.Span), RelatedSpan{
					Role: "declaration", Span: span(source, declaration),
				})
				continue
			}
			if _, duplicate := seenEffects[effect]; duplicate {
				validDeclaration = false
				a.addDiagnostic("E7101", span(source, argument.Span), RelatedSpan{
					Role: "declaration", Span: span(source, declaration),
				})
				continue
			}
			seenEffects[effect] = struct{}{}
			effects = append(effects, effect)
		}
	}
	return hir.CanonicalEffects(effects), effectSpan, validDeclaration
}

func (c *typeChecker) checkCallEffects(expression *ast.Expression, result *hir.Expression, callee *symbol) {
	result.Effects = slices.Clone(callee.effects)
	caller := c.analyzer.functions[c.function]
	if caller == nil ||
		!caller.effectsValid ||
		!callee.effectsValid ||
		len(hir.MissingEffects(caller.effects, callee.effects)) == 0 {
		return
	}
	relatedSpan := callee.declaration
	if callee.effectSpan.EndByte > callee.effectSpan.StartByte {
		relatedSpan = callee.effectSpan
	}
	c.analyzer.addDiagnostic("E7102", span(c.module.file.Path, expression.Span), RelatedSpan{
		Role: "effect-declaration", Span: span(callee.source, relatedSpan),
	})
}
