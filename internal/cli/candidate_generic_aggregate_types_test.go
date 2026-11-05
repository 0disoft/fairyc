package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericsource"
)

func TestCandidateGenericAggregateTypes(t *testing.T) {
	wire := candidateGenericAggregateOracle(t, true)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_plan.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_plan;", "module main;", 1)
	main = strings.Replace(main, "verify(bytes::view(ref packet))", "verify_types(bytes::view(ref packet))", 1)
	main += "\n@effects(console) fn i32 main(){return run();}\n"
	candidateAtlasProbeSize(t, "generic_aggregate_types", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, wire); code != 0 || out != "" {
			t.Fatalf("aggregate concrete types: %d %q", code, out)
		}
	}, "module_public_templates", "project_source_plan", "generic_aggregate_types")
}

// The oracle uses the Go adapter's actual detached concrete fields/signatures,
// not the Fairy preparer's type tags or a parallel text-substitution routine.
func candidateGenericAggregateTypeFacts(t *testing.T, expanded genericsource.PackageExpansion, number func(int), field func(string)) {
	t.Helper()
	names := map[string]string{}
	for _, origin := range expanded.Origins {
		names[origin.GeneratedName] = string(origin.Specialization)
	}
	var render func(ast.TypeReference) string
	render = func(r ast.TypeReference) string {
		switch r.Kind {
		case ast.TypeKindNamed:
			name := r.Path
			parts := strings.Split(name, "::")
			if id, ok := names[parts[len(parts)-1]]; ok {
				name = id
			}
			if len(r.Arguments) != 0 {
				var args []string
				for _, a := range r.Arguments {
					args = append(args, render(a))
				}
				name += "<" + strings.Join(args, ",") + ">"
			}
			return name
		case ast.TypeKindGrouped:
			return render(*r.Element)
		case ast.TypeKindArray:
			return render(*r.Element) + "[" + r.Length.Text + "]"
		case ast.TypeKindRawPointer:
			if r.Mutable {
				return "ptr mut " + render(*r.Element)
			}
			return "ptr " + render(*r.Element)
		case ast.TypeKindReference:
			if r.Mutable {
				return "ref mut " + render(*r.Element)
			}
			return "ref " + render(*r.Element)
		default:
			t.Fatalf("uncovered oracle type: %+v", r)
			return ""
		}
	}
	type shape struct {
		id     string
		module int
		fields []ast.FieldDeclaration
	}
	var shapes []shape
	for m, module := range expanded.Modules {
		for _, s := range module.Structs {
			if id, ok := names[s.Name]; ok {
				shapes = append(shapes, shape{id, m, s.Fields})
			}
		}
		for _, s := range module.Resources {
			if id, ok := names[s.Name]; ok {
				shapes = append(shapes, shape{id, m, s.Fields})
			}
		}
	}
	if len(shapes) != 5 {
		t.Fatal("concrete aggregate coverage", len(shapes))
	}
	number(len(shapes))
	for _, s := range shapes {
		field(s.id)
		number(s.module)
		number(len(s.fields))
		for _, f := range s.fields {
			field(f.Name)
			number(f.Span.StartByte)
			number(f.Span.EndByte)
			field(render(f.Type))
		}
	}
	type function struct {
		id         string
		result     ast.TypeReference
		parameters []ast.Parameter
	}
	var functions []function
	for _, module := range expanded.Modules {
		for _, f := range module.Functions {
			if id, ok := names[f.Name]; ok {
				functions = append(functions, function{id, f.ResultType, f.Parameters})
			}
		}
	}
	if len(functions) != 4 {
		t.Fatal("concrete function coverage", len(functions))
	}
	number(len(functions))
	for _, f := range functions {
		field(f.id)
		field(render(f.result))
		number(len(f.parameters))
		for _, p := range f.parameters {
			field(render(p.Type))
		}
	}
}
