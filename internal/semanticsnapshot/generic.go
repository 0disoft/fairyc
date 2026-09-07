package semanticsnapshot

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
)

// GenericDeclarations detaches original generic headers, independently of observed
// specializations. It publishes source-declared signatures only: unused bodies,
// inferred effects and executable summaries are not proven here. Nominal names
// use compiler-published visible scope facts, never a guessed specialization.
func GenericDeclarations(provenance compiler.GenericProvenance, packages []PackageInput) ([]Declaration, error) {
	metadata, _, _, err := normalizePackages(packages)
	if err != nil {
		return nil, err
	}
	type scopeKey struct {
		packageID hir.PackageID
		module    string
	}
	nominals := map[scopeKey]map[string]TypeRef{}
	for _, fact := range provenance.NominalTypes() {
		key := scopeKey{fact.Package, fact.Module}
		if _, ok := metadata[fact.Package]; !ok {
			return nil, fmt.Errorf("generic nominal scope has no package metadata")
		}
		resolved, err := normalizeType(fact.CanonicalType, metadata)
		if err != nil || resolved.Kind != TypeNominal {
			return nil, fmt.Errorf("generic nominal scope has no canonical nominal type")
		}
		if nominals[key] == nil {
			nominals[key] = map[string]TypeRef{}
		}
		if _, duplicate := nominals[key][fact.Name]; duplicate {
			return nil, fmt.Errorf("generic nominal scope repeats a written name")
		}
		nominals[key][fact.Name] = resolved
	}
	var declarations []Declaration
	seen := map[DeclarationKey]bool{}
	for _, original := range provenance.Declarations() {
		info, ok := metadata[original.Package]
		if !ok || original.Span.Package != original.Package {
			return nil, fmt.Errorf("generic declaration has no matching package metadata")
		}
		if !slices.ContainsFunc(info.input.Files, func(file FileInput) bool { return file.Path == original.Span.Source }) {
			return nil, fmt.Errorf("generic declaration has no exact source file identity")
		}
		module := hir.Module{Package: original.Package, Path: original.Module, SourcePath: original.Span.Source}
		if err := validateModule(module.Path, module.SourcePath); err != nil {
			return nil, err
		}
		declaration, err := declarationBase(module, DeclarationFunction, original.Name, original.Span, metadata)
		if err != nil {
			return nil, err
		}
		if seen[declaration.Key] {
			return nil, fmt.Errorf("generic declaration is duplicated")
		}
		seen[declaration.Key] = true
		binders := map[string]int{}
		for position, name := range original.Parameters {
			if _, duplicate := binders[name]; duplicate || !source.ValidIdentifier(name) {
				return nil, fmt.Errorf("generic declaration has invalid type parameters")
			}
			binders[name] = position
		}
		if len(binders) == 0 {
			return nil, fmt.Errorf("generic declaration has no type parameters")
		}
		declaration.TypeParameters = slices.Clone(original.Parameters)
		declaration.Public = original.Public
		declaration.Unsafe = original.Signature.Unsafe
		declaration.Effects, err = genericEffects(original.Signature.Attributes)
		if err != nil {
			return nil, fmt.Errorf("generic %s effects: %w", original.Name, err)
		}
		visible := nominals[scopeKey{original.Package, original.Module}]
		declaration.Type, err = genericType(original.Signature.Result, binders, visible, 0)
		if err != nil {
			return nil, fmt.Errorf("generic %s result: %w", original.Name, err)
		}
		for _, argument := range original.Signature.Arguments {
			typeRef, err := genericType(argument.Type, binders, visible, 0)
			if err != nil {
				return nil, fmt.Errorf("generic %s argument %s: %w", original.Name, argument.Name, err)
			}
			span := original.Span
			span.StartByte, span.EndByte = argument.NameSpan.StartByte, argument.NameSpan.EndByte
			if span.StartByte < original.Span.StartByte || span.EndByte > original.Span.EndByte {
				return nil, fmt.Errorf("generic argument span escapes its declaration")
			}
			stableSpan, err := normalizeSpan(span, metadata)
			if err != nil {
				return nil, err
			}
			declaration.Parameters = append(declaration.Parameters, Parameter{Name: argument.Name, Mutable: argument.Mutable, Type: typeRef, Span: stableSpan})
		}
		declarations = append(declarations, declaration)
	}
	slices.SortFunc(declarations, func(a, b Declaration) int { return compareDeclarationKey(a.Key, b.Key) })
	return declarations, nil
}

func genericEffects(attributes []ast.Attribute) ([]string, error) {
	var result []string
	seen := false
	for _, attribute := range attributes {
		if attribute.Name != "effects" {
			continue
		}
		if seen || len(attribute.Arguments) == 0 {
			return nil, fmt.Errorf("invalid or repeated effects attribute")
		}
		seen = true
		for _, argument := range attribute.Arguments {
			effect, ok := hir.ParseEffect(argument.Raw)
			if !ok || slices.Contains(result, string(effect)) {
				return nil, fmt.Errorf("invalid or repeated declared effect")
			}
			result = append(result, string(effect))
		}
	}
	slices.Sort(result)
	return result, nil
}

// This conversion describes written structure, not generic-body type checking.
// Text is diagnostic spelling only; all type decisions use structural AST fields.
func genericType(value ast.TypeReference, binders map[string]int, nominals map[string]TypeRef, depth int) (TypeRef, error) {
	if depth > 64 {
		return TypeRef{}, fmt.Errorf("generic signature type nesting exceeds the snapshot bound")
	}
	child := func(value ast.TypeReference) (TypeRef, error) { return genericType(value, binders, nominals, depth+1) }
	switch value.Kind {
	case ast.TypeKindGrouped, ast.TypeKindReference, ast.TypeKindRawPointer, ast.TypeKindArray, ast.TypeKindSlice:
		if value.Element == nil {
			return TypeRef{}, fmt.Errorf("generic signature type is missing its element")
		}
		element, err := child(*value.Element)
		if err != nil {
			return TypeRef{}, err
		}
		if value.Kind == ast.TypeKindGrouped {
			return element, nil
		}
		if value.Kind == ast.TypeKindSlice {
			if element.Kind != TypeBuiltin || element.Name != string(hir.TypeU8) {
				return TypeRef{}, fmt.Errorf("generic slice element has no resolved byte-slice identity")
			}
			return TypeRef{Kind: TypeBuiltin, Name: string(hir.TypeByteSlice)}, nil
		}
		result := TypeRef{Kind: TypeBorrow, Mutable: value.Mutable, Arguments: []TypeRef{element}}
		if value.Kind == ast.TypeKindRawPointer {
			result.Kind = TypePointer
		}
		if value.Kind == ast.TypeKindArray {
			if value.Length == nil {
				return TypeRef{}, fmt.Errorf("generic array has no length")
			}
			text := strings.ReplaceAll(value.Length.Text, "_", "")
			base := 10
			if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
				base, text = 16, text[2:]
			} else if strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
				base, text = 2, text[2:]
			}
			length, err := strconv.ParseUint(text, base, 64)
			if err != nil || length == 0 {
				return TypeRef{}, fmt.Errorf("generic array requires a resolved positive literal length")
			}
			result.Kind, result.Length, result.Mutable = TypeFixedArray, length, false
		}
		return result, nil
	case ast.TypeKindNamed:
		if position, ok := binders[value.Path]; ok && len(value.Arguments) == 0 {
			return TypeRef{Kind: TypeParameter, Name: strconv.Itoa(position)}, nil
		}
		if len(value.Arguments) == 0 && builtinType(hir.Type(value.Path)) {
			return TypeRef{Kind: TypeBuiltin, Name: value.Path}, nil
		}
		if resolved, ok := nominals[value.Path]; ok && len(value.Arguments) == 0 {
			return cloneType(resolved), nil
		}
		if value.Path == "Option" && len(value.Arguments) == 1 || value.Path == "Result" && len(value.Arguments) == 2 {
			kind := TypeOption
			if value.Path == "Result" {
				kind = TypeResult
			}
			result := TypeRef{Kind: kind}
			for _, argument := range value.Arguments {
				converted, err := child(argument)
				if err != nil {
					return TypeRef{}, err
				}
				result.Arguments = append(result.Arguments, converted)
			}
			return result, nil
		}
	}
	return TypeRef{}, fmt.Errorf("generic signature type %q requires compiler-owned resolution", value.Path)
}
