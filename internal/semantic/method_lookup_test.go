package semantic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
)

func TestMethodLookupUsesAggregateIdentityAndReceiver(t *testing.T) {
	const text = `module main;
struct Box { i32 value; }
enum Tag { Value; }
resource Token { cleanup {} }
fn i32 get() { return 0; }
fn i32 Box::get(ref Box this) { return this.value; }
fn i32 Tag::get(ref Tag this) { return 0; }
fn i32 Token::get(ref Token this) { return 0; }
fn i32 Box::create() { return 0; }
fn i32 Box::plain(i32 value) { return value; }
fn i32 main() { return 0; }
`
	var files []File
	for _, packageID := range []hir.PackageID{1, 2} {
		path := "main.fy"
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			t.Fatalf("parse methods: %v, diagnostics: %+v", err, parsed.Tree.Diagnostics)
		}
		files = append(files, File{Package: packageID, Path: path, Module: parsed.AST})
	}
	a := newAnalyzer(files, nil)
	a.resolve()
	a.prepareAggregateIdentities()
	if len(a.diagnostics) != 0 {
		t.Fatalf("resolve methods: %+v", a.diagnostics)
	}
	for _, module := range a.moduleOrder {
		for _, name := range []string{"Box", "Tag", "Token"} {
			owner := a.ownerSymbol(module, name).typ
			want := module.functions[name+"::get"]
			if got := a.methodSymbol(owner, "get"); got == nil || got != want {
				t.Fatalf("%v %s::get = %p, want %p", module.key, name, got, want)
			}
			for _, name := range []string{"missing", "create", "plain"} {
				if got := a.methodSymbol(owner, name); got != nil {
					t.Fatalf("unexpected method %s: %+v", name, got)
				}
			}
		}
	}
	if got := a.methodSymbol(hir.TypeI32, "get"); got != nil {
		t.Fatalf("free function selected as method: %+v", got)
	}
}

func TestMethodLookupPreservesFirstDeclaration(t *testing.T) {
	for _, resolved := range []bool{false, true} {
		t.Run(fmt.Sprint(resolved), func(t *testing.T) {
			functions := []ast.Function{
				{Owner: "Box", Name: "get", Parameters: []ast.Parameter{{Name: "this"}}},
				{Owner: "Box", Name: "get", Parameters: []ast.Parameter{{Name: "this"}}},
			}
			owner := hir.ParseType("main::Box")
			module := &moduleScope{file: File{Module: &ast.Module{Functions: functions}}, structs: map[string]*symbol{"Box": {typ: owner}}}
			a := newAnalyzer(nil, nil)
			a.moduleOrder = []*moduleScope{module}
			a.functions[&functions[1]] = &symbol{id: 2}
			var want *symbol
			if resolved {
				want = &symbol{id: 1}
				a.functions[&functions[0]] = want
			}
			for range 2 {
				if got := a.methodSymbol(owner, "get"); got != want {
					t.Fatalf("method = %p, want first declaration %p", got, want)
				}
			}
		})
	}
}

func methodLookupFiles(tb testing.TB, count int) []File {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main; struct Box { i32 value; }")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&text, "fn i32 unused_%d(){return 0;}", i)
	}
	text.WriteString("fn i32 Box::get(ref Box this){return this.value;}fn i32 main(){let Box box=Box{.value=42};")
	text.WriteString(strings.Repeat("discard box.get();", count))
	text.WriteString("return 0;}")
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		tb.Fatalf("parse methods: %v", err)
	}
	return []File{{Path: "main.fy", Module: parsed.AST}}
}

func BenchmarkMethodLookup(b *testing.B) {
	for _, count := range []int{64, 2048} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			a := newAnalyzer(methodLookupFiles(b, count), nil)
			a.resolve()
			a.prepareAggregateIdentities()
			owner := hir.ParseType("main::Box")
			want := a.methodSymbol(owner, "get")
			if want == nil || len(a.diagnostics) != 0 {
				b.Fatal("method fixture failed resolution")
			}
			b.ReportAllocs()
			for b.Loop() {
				if got := a.methodSymbol(owner, "get"); got != want {
					b.Fatal("method identity changed")
				}
			}
		})
	}
}

func BenchmarkAnalyzeMethodCalls(b *testing.B) {
	for _, count := range []int{64, 2048} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			files := methodLookupFiles(b, count)
			b.ReportAllocs()
			for b.Loop() {
				result := Analyze(files)
				if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
					b.Fatalf("method analysis: %+v %+v", result.Diagnostics, result.Unsupported)
				}
			}
		})
	}
}
