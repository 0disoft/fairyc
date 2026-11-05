package genericsource

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/source"
)

func TestOwnerQualifiedFunctionNamespace(t *testing.T) {
	for _, fixture := range []struct {
		name, declarations string
		accept             bool
	}{
		{"distinct owners", "fn i32 Cell::get(){return 1;}fn i32 Other::get(){return 2;}", true},
		{"ordinary same leaf", "fn i32 Cell::get(){return 1;}fn i32 get(){return 2;}", true},
		{"duplicate owner", "fn i32 Cell::get(){return 1;}fn i32 Cell::get(){return 2;}", false},
		{"duplicate ordinary", "fn i32 get(){return 1;}fn i32 get(){return 2;}", false},
		{"generic owner remains closed", "fn T Cell::pick<T>(T value){return value;}", false},
		{"generated name remains reserved", "fn i32 Cell::_fy_generic_hidden(){return 1;}", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			text := "module main;struct Cell{i32 value;}struct Other{i32 value;}" + fixture.declarations
			parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				t.Fatalf("parse: %v", err)
			}
			before, err := json.Marshal(parsed.AST)
			if err != nil {
				t.Fatal(err)
			}
			expansion, err := PrepareModule(parsed.AST, "owner", genericplan.DefaultPlanLimits())
			if fixture.accept {
				if err != nil || len(expansion.Modules) != 1 || len(expansion.Modules[0].Functions) != len(parsed.AST.Functions) {
					t.Fatalf("distinct namespaces rejected: %v", err)
				}
				for i, fn := range expansion.Modules[0].Functions {
					original := parsed.AST.Functions[i]
					if fn.Name != original.Name || fn.Owner != original.Owner {
						t.Fatal("namespace admission rewrote source function identity")
					}
				}
			} else {
				var located *SourceError
				if !errors.As(err, &located) || located.Kind != FailureDeclaration || len(expansion.Modules) != 0 {
					t.Fatalf("invalid namespace lost its declaration refusal: %v", err)
				}
			}
			after, err := json.Marshal(parsed.AST)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("namespace preparation mutated caller source")
			}
		})
	}
}
