package genericsource

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func TestSourceDependenciesBoundedOrderedAndExact(t *testing.T) {
	x := &expander{owner: "owner", module: &ast.Module{Path: "main", SourceSize: 100}}
	d := sourceDependencies{x: x, span: diagnostic.Span{StartByte: 2, EndByte: 90}}
	declaration := genericplan.Declaration{ID: "owner::main::root"}
	for index := range 32 {
		dependency := genericplan.Dependency{Declaration: genericplan.DeclarationID(fmt.Sprintf("leaf%d", index)), Arguments: []genericplan.TypeExpression{genericplan.ParameterType("T")}}
		if err := d.add(&declaration, dependency); err != nil {
			t.Fatal(err)
		}
	}
	before := append([]genericplan.Dependency(nil), declaration.Dependencies...)
	for index := range 10000 {
		if err := d.add(&declaration, before[index%32]); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(before, declaration.Dependencies) || len(d.seen) != 32 {
		t.Fatal("duplicates changed order or expanded the bounded index")
	}
	err := d.add(&declaration, genericplan.Dependency{Declaration: "extra"})
	var located *SourceError
	if genericplan.ErrorCode(err) != genericplan.CodeCatalogBudgetExceeded || !errors.As(err, &located) || located.Kind != FailureDeclaration || located.Span != d.span || len(declaration.Dependencies) != 32 || len(d.seen) != 32 {
		t.Fatalf("overflow changed provenance/state: %v", err)
	}
	// Parameter and concrete names, tuple order and tuple length remain distinct.
	d = sourceDependencies{x: x, span: d.span}
	declaration.Dependencies = nil
	for _, arguments := range [][]genericplan.TypeExpression{
		{genericplan.ParameterType("T")}, {genericplan.ConcreteType("T")},
		{genericplan.ParameterType("T"), genericplan.ConcreteType("i32")},
		{genericplan.ConcreteType("i32"), genericplan.ParameterType("T")},
	} {
		if err := d.add(&declaration, genericplan.Dependency{Declaration: "leaf", Arguments: arguments}); err != nil {
			t.Fatal(err)
		}
	}
	if len(declaration.Dependencies) != 4 {
		t.Fatal("distinct dependency tuples collapsed")
	}
}

func TestSourceDependencyAdmissionFunctionsAndAggregates(t *testing.T) {
	for _, aggregate := range []bool{false, true} {
		for _, count := range []int{32, 33, 4000} {
			t.Run(fmt.Sprintf("aggregate=%t/count=%d", aggregate, count), func(t *testing.T) {
				var text strings.Builder
				text.WriteString("module main;")
				if aggregate {
					text.WriteString("struct Root<T>{")
				} else {
					text.WriteString("fn T root<T>(T value){")
				}
				for index := range count {
					if aggregate {
						fmt.Fprintf(&text, "Leaf%d<T> field%d;", index, index)
					} else {
						fmt.Fprintf(&text, "discard leaf%d<T>(value);", index)
					}
				}
				if aggregate {
					text.WriteString("}")
				} else {
					text.WriteString("return value;}")
				}
				for index := range count {
					if aggregate {
						fmt.Fprintf(&text, "struct Leaf%d<T>{T value;}", index)
					} else {
						fmt.Fprintf(&text, "fn T leaf%d<T>(T value){return value;}", index)
					}
				}
				text.WriteString("fn i32 main(){return 0;}")
				parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
				if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
					t.Fatalf("parse: %v", err)
				}
				before, err := json.Marshal(parsed.AST)
				if err != nil {
					t.Fatal(err)
				}
				expansion, err := PrepareModule(parsed.AST, "owner", genericplan.DefaultPlanLimits())
				if count <= 32 {
					if err != nil || len(expansion.Modules) != 1 {
						t.Fatalf("exact limit: %v", err)
					}
				} else {
					var located *SourceError
					if genericplan.ErrorCode(err) != genericplan.CodeCatalogBudgetExceeded || !errors.As(err, &located) || located.Kind != FailureDeclaration || len(expansion.Modules) != 0 {
						t.Fatalf("excess dependencies admitted or wrong provenance: %v", err)
					}
				}
				after, err := json.Marshal(parsed.AST)
				if err != nil || string(before) != string(after) {
					t.Fatal("source AST mutated during admission")
				}
			})
		}
	}
}
