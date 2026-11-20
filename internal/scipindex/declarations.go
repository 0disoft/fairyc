package scipindex

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"github.com/scip-code/scip/bindings/go/scip"
)

func (doc *document) addDeclarations(pkg workspaceloader.SemanticPackage) {
	module := doc.source.Module
	identity := &scip.Package{Manager: "fairy", Name: pkg.PackageUID, Version: pkg.Coordinate.String()}
	namespace := make([]*scip.Descriptor, 0, 4)
	for part := range strings.SplitSeq(module.Path, "::") {
		namespace = append(namespace, &scip.Descriptor{Name: part, Suffix: scip.Descriptor_Namespace})
	}
	add := func(span diagnostic.Span, name string, kind scip.Descriptor_Suffix, owners ...*scip.Descriptor) {
		if name == "" || !doc.validSpan(span.StartByte, span.EndByte) {
			return
		}
		descriptors := append(namespace, owners...)
		descriptors = append(descriptors, &scip.Descriptor{Name: name, Suffix: kind})
		doc.symbols[span] = scip.VerboseSymbolFormatter.FormatSymbol(&scip.Symbol{
			Scheme: "scip-fairyc", Package: identity, Descriptors: descriptors,
		})
	}
	fields := func(items []ast.FieldDeclaration, owners ...*scip.Descriptor) {
		for _, field := range items {
			add(field.NameSpan, field.Name, scip.Descriptor_Term, owners...)
		}
	}
	for _, item := range module.Structs {
		add(item.NameSpan, item.Name, scip.Descriptor_Type)
		fields(item.Fields, &scip.Descriptor{Name: item.Name, Suffix: scip.Descriptor_Type})
	}
	for _, item := range module.Resources {
		add(item.NameSpan, item.Name, scip.Descriptor_Type)
		fields(item.Fields, &scip.Descriptor{Name: item.Name, Suffix: scip.Descriptor_Type})
	}
	for _, item := range module.Enums {
		add(item.NameSpan, item.Name, scip.Descriptor_Type)
		owner := &scip.Descriptor{Name: item.Name, Suffix: scip.Descriptor_Type}
		for _, variant := range item.Variants {
			add(variant.NameSpan, variant.Name, scip.Descriptor_Term, owner)
			fields(variant.Fields, owner, &scip.Descriptor{Name: variant.Name, Suffix: scip.Descriptor_Term})
		}
	}
	for _, item := range module.Constants {
		add(item.NameSpan, item.Name, scip.Descriptor_Term)
	}
	for _, item := range module.Functions {
		if item.Owner == "" {
			add(item.NameSpan, item.Name, scip.Descriptor_Method)
		} else {
			add(item.NameSpan, item.Name, scip.Descriptor_Method, &scip.Descriptor{Name: item.Owner, Suffix: scip.Descriptor_Type})
		}
	}
	for _, item := range module.ForeignFunctions {
		add(item.NameSpan, item.Name, scip.Descriptor_Method)
	}
}
