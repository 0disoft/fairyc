package semanticsnapshot_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestGenericAggregateDeclaredAttributes(t *testing.T) {
	for _, instantiate := range []bool{false, true} {
		t.Run(map[bool]string{false: "unused", true: "instantiated"}[instantiate], func(t *testing.T) {
			text := `module main;
@repr(C) struct Marker<T> { c::int value; }
@effects(filesystem, console) resource Token<T> { T value; cleanup { discard this.value; } }
@effects(console, filesystem) fn i32 main() { BODY return 42; }`
			body := ""
			if instantiate {
				body = "let a = Marker<i32> { .value = 42 as c::int }; let b = Token<i32> { .value = 7 };"
			}
			text = strings.Replace(text, "BODY", body, 1)
			file := semantic.File{Package: hir.DefaultPackage, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", text)}
			compiled, _, err := compiler.CheckGenericFunctions(file, genericplan.DefaultPlanLimits())
			if err != nil || !compiled.Verified {
				t.Fatalf("compile: %v %+v", err, compiled.Semantic)
			}
			input := packageInput(t, hir.DefaultPackage, "example.aggregate.attributes", "attributes", "1", "main.fy", text, "root")
			read := func() []semanticsnapshot.Declaration {
				t.Helper()
				declarations, err := semanticsnapshot.GenericDeclarations(compiled.GenericProvenance, []semanticsnapshot.PackageInput{input})
				if err != nil || len(declarations) != 2 {
					t.Fatalf("headers: %v %+v", err, declarations)
				}
				for _, d := range declarations {
					if d.Key.Name == "Marker" && (!d.ReprC || len(d.DropEffects) != 0) {
						t.Fatalf("representation lost: %+v", d)
					}
					if d.Key.Name == "Token" && (d.ReprC || d.DropKind != "source-declared" || len(d.Effects) != 0 || !reflect.DeepEqual(d.DropEffects, []string{"console", "filesystem"})) {
						t.Fatalf("declared cleanup authority lost: %+v", d)
					}
				}
				return declarations
			}
			before := read()
			for _, d := range read() {
				if len(d.DropEffects) != 0 {
					d.DropEffects[0] = "corrupted"
				}
			}
			if !reflect.DeepEqual(before, read()) {
				t.Fatal("header effects alias an earlier view")
			}
			snapshot, err := semanticsnapshot.BuildCompiler(compiled, []semanticsnapshot.PackageInput{input})
			if err != nil {
				t.Fatalf("snapshot: %v", err)
			}
			want := 0
			if instantiate {
				want = 2
			}
			if len(snapshot.Specializations()) != want {
				t.Fatalf("specializations=%d, want %d", len(snapshot.Specializations()), want)
			}
		})
	}
}

func TestGenericAggregateHeadersRejectMalformedAttributes(t *testing.T) {
	for _, declaration := range []string{
		"@repr(C) @repr(C) struct Unused<T> { T value; }",
		"@repr(C, C) struct Unused<T> { T value; }",
		"@repr(packed) struct Unused<T> { T value; }",
		"@effects(console) struct Unused<T> { T value; }",
		"@repr(C) resource Unused<T> { T value; cleanup {} }",
		"@effects() resource Unused<T> { T value; cleanup {} }",
		"@effects(console) @effects(console) resource Unused<T> { T value; cleanup {} }",
		"@effects(console, console) resource Unused<T> { T value; cleanup {} }",
		"@effects(unknown) resource Unused<T> { T value; cleanup {} }",
		"@link_name(\"wrong\") resource Unused<T> { T value; cleanup {} }",
	} {
		t.Run(declaration, func(t *testing.T) {
			_, _, err := genericSourceDeclarations(t, "module main; "+declaration+" fn i32 main() { return 0; }")
			if err == nil {
				t.Fatal("malformed or unsupported unused aggregate attribute was accepted")
			}
		})
	}
}
