package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

const contractUIDAttribute = "contract_uid"

type sourceStateOwner struct {
	module      *moduleScope
	declaration *ast.StateDeclaration
}

func (a *analyzer) checkStateLayouts() []hir.StateLayout {
	owners := make(map[hir.PackageID]sourceStateOwner)
	invalidPackages := make(map[hir.PackageID]struct{})
	var ordered []sourceStateOwner
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.States {
			declaration := &module.file.Module.States[index]
			if previous, found := owners[module.key.packageID]; found {
				a.addDiagnostic("E7201", span(module.file.Path, declaration.Span), RelatedSpan{
					Role: "previous-declaration", Span: span(previous.module.file.Path, previous.declaration.Span),
				})
				invalidPackages[module.key.packageID] = struct{}{}
				continue
			}
			owner := sourceStateOwner{module: module, declaration: declaration}
			owners[module.key.packageID] = owner
			ordered = append(ordered, owner)
		}
	}
	var layouts []hir.StateLayout
	for _, owner := range ordered {
		if _, invalid := invalidPackages[owner.module.key.packageID]; invalid {
			continue
		}
		if layout, ok := a.checkStateLayout(owner.module, owner.declaration); ok {
			layouts = append(layouts, layout)
		}
	}
	return layouts
}

func (a *analyzer) checkStateLayout(module *moduleScope, declaration *ast.StateDeclaration) (hir.StateLayout, bool) {
	valid := true
	uid := ""
	if declaration.Public {
		a.addDiagnostic("E7201", span(module.file.Path, declaration.Span))
		valid = false
	}
	for _, attribute := range declaration.Attributes {
		if attribute.Name != contractUIDAttribute || uid != "" || len(attribute.Arguments) != 1 ||
			len(attribute.Arguments[0].Raw) < 2 || attribute.Arguments[0].Raw[0] != '"' || attribute.Arguments[0].Cooked == "" {
			a.addDiagnostic("E7201", span(module.file.Path, attribute.Span))
			valid = false
			continue
		}
		uid = attribute.Arguments[0].Cooked
	}
	if uid == "" {
		a.addDiagnostic("E7201", span(module.file.Path, declaration.Span))
		valid = false
	}

	result := hir.StateLayout{Package: module.key.packageID, ContractUID: uid, Span: span(module.file.Path, declaration.Span)}
	seen := make(map[string]hir.Span, len(declaration.Fields))
	for _, field := range declaration.Fields {
		fieldSpan := span(module.file.Path, field.NameSpan)
		if previous, duplicate := seen[field.Name]; duplicate {
			a.addDiagnostic("E7202", fieldSpan, RelatedSpan{Role: "previous-declaration", Span: previous})
			valid = false
			continue
		}
		seen[field.Name] = fieldSpan
		valueType, valueOK := stateCodecTypeID(field.ValueType)
		if !valueOK {
			a.addDiagnostic("E7203", span(module.file.Path, field.ValueType.Span))
			valid = false
			continue
		}
		hirField := hir.StateField{Name: field.Name, ValueTypeID: valueType, Span: span(module.file.Path, field.Span)}
		switch field.Kind {
		case ast.StateFieldCell:
			hirField.Kind = hir.StateFieldCell
		case ast.StateFieldMap:
			hirField.Kind = hir.StateFieldMap
			if field.KeyType == nil {
				a.addDiagnostic("E7203", span(module.file.Path, field.Span))
				valid = false
				continue
			}
			keyType, keyOK := stateCodecTypeID(*field.KeyType)
			if !keyOK {
				a.addDiagnostic("E7203", span(module.file.Path, field.KeyType.Span))
				valid = false
				continue
			}
			hirField.KeyTypeID = keyType
		default:
			a.addDiagnostic("E7203", span(module.file.Path, field.Span))
			valid = false
			continue
		}
		result.Fields = append(result.Fields, hirField)
	}
	if !valid {
		return hir.StateLayout{}, false
	}
	builder, err := contractstate.NewLayoutBuilder(uid)
	if err != nil {
		a.addDiagnostic("E7201", span(module.file.Path, declaration.Span))
		return hir.StateLayout{}, false
	}
	for _, field := range result.Fields {
		if field.Kind == hir.StateFieldCell {
			_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
		} else {
			_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
		}
		if err != nil {
			break
		}
	}
	if err == nil {
		_, err = builder.Build()
	}
	if err != nil {
		a.addDiagnostic("E7203", span(module.file.Path, declaration.Span))
		return hir.StateLayout{}, false
	}
	return result, true
}

func stateCodecTypeID(reference ast.TypeReference) (string, bool) {
	if reference.Kind != ast.TypeKindNamed || len(reference.Arguments) != 0 || strings.Contains(reference.Path, "::") {
		return "", false
	}
	switch reference.Path {
	case "bool":
		return contractstate.Bool.TypeID(), true
	case "u32":
		return contractstate.U32.TypeID(), true
	case "u64":
		return contractstate.U64.TypeID(), true
	case "bytes20":
		return contractstate.Byte20.TypeID(), true
	case "bytes32":
		return contractstate.Byte32.TypeID(), true
	default:
		return "", false
	}
}
