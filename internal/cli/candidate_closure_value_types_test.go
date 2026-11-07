package cli

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func candidateClosureTypeSources() []string {
	return []string{
		"module sample; // 한글\r\nfn fn /* result */ i32(i32) identity(fn i32(i32) action){let fn /* binding */ i32(i32) copied=action;return copied;}\r\n",
		"module sample; // 반환\r\nfn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}fn i32 apply(fn i32(i32) action,i32 x){return action(x);}",
		"module sample; // 순서\r\nstruct Job{fn /* 결과 한글 */ Result<u32,u16>(i8,ref u8[],fn bool(u64,f32),) action;fn fn i32(i32)(u32) factory;fn void() empty;}",
		"module sample;struct Holder{Option<fn i32(i32)> optional;Result<fn i32(i32),u32> result;ref mut fn i32(i32) borrowed;ptr fn i32(i32) raw;(fn i32(i32))[2] array;}",
		"module sample;struct Mixed{fn u64(i8,u16,f32,i32,u32,u8,usize,bool,i64,f64,) ordered;fn (fn i32(i32))(u32) grouped_result;fn unsafe extern C fn i32(ptr void)(i64) callback_result;}",
		"module sample;fn fn T(T,Option<fn T(T)>) generic<T>(fn T(T) value){let fn T(T) local=value;return local;}",
		"module sample; // old callbacks\r\nstruct Callbacks{unsafe extern C fn i32(i32,ptr mut void) callback;unsafe extern C fn void() empty;unsafe extern C fn fn i32(i32)(u32) safe_result;}",
		"module sample;fn Result<Option<T>,ptr mut u8> old<T>(ref mut T[] values,T[4] array,(pkg::Thing) grouped,unsafe extern C fn i32(ptr void,i64) callback){return .err(null);}",
		"module sample; // legacy\r\nfn ref (T[Size]) old<T>(ref T[2][3] data,ptr ptr mut void raw){return ref data;}",
		"module sample;fn i32 old(i32 value){return value;}",
	}
}

func candidateClosureTypeKind(t *testing.T, typ ast.TypeReference) int {
	t.Helper()
	switch typ.Kind {
	case ast.TypeKindNamed:
		if len(typ.Arguments) != 0 {
			return 6
		}
		return 1
	case ast.TypeKindReference:
		if typ.Mutable {
			return 3
		}
		return 2
	case ast.TypeKindRawPointer:
		if typ.Mutable {
			return 12
		}
		return 11
	case ast.TypeKindSlice:
		return 4
	case ast.TypeKindGrouped:
		return 5
	case ast.TypeKindArray:
		return 8
	case ast.TypeKindForeignFunction:
		return 15
	case ast.TypeKindClosureFunction:
		return 17
	default:
		t.Fatalf("unowned AST type form: %s", typ.Kind)
		return 0
	}
}

func candidateClosureTypeReferences(module *ast.Module) []ast.TypeReference {
	var roots, all []ast.TypeReference
	for _, structure := range module.Structs {
		for _, field := range structure.Fields {
			roots = append(roots, field.Type)
		}
	}
	for _, fn := range module.Functions {
		roots = append(roots, fn.ResultType)
		for _, parameter := range fn.Parameters {
			roots = append(roots, parameter.Type)
		}
		if fn.Body != nil {
			for _, statement := range fn.Body.Statements {
				if statement.Binding != nil && statement.Binding.Type != nil {
					roots = append(roots, *statement.Binding.Type)
				}
			}
		}
	}
	var visit func(ast.TypeReference)
	visit = func(typ ast.TypeReference) {
		all = append(all, typ)
		if typ.Element != nil {
			visit(*typ.Element)
		}
		for _, argument := range typ.Arguments {
			visit(argument)
		}
		for _, parameter := range typ.Parameters {
			visit(parameter)
		}
		if typ.Result != nil {
			visit(*typ.Result)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	return all
}

// These old-shape expectations are constructed without the parser, including
// nil/empty fields, pointer mutability, callback ABI and original byte spans.
func candidateClosureTypeLegacyJSON(t *testing.T) {
	t.Helper()
	raw := "module sample;struct Legacy{unsafe extern C fn i32(i32,ptr mut void) callback;i32 count;}"
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(raw)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("old callback control: %v", err)
	}
	span := func(text string) diagnostic.Span {
		start := strings.Index(raw, text)
		if start < 0 {
			t.Fatal(text)
		}
		return diagnostic.Span{StartByte: start, EndByte: start + len(text)}
	}
	result := ast.NamedTypeReference("i32", diagnostic.Span{StartByte: strings.Index(raw, "i32("), EndByte: strings.Index(raw, "i32(") + 3})
	parameter := ast.NamedTypeReference("i32", diagnostic.Span{StartByte: strings.Index(raw, "i32,"), EndByte: strings.Index(raw, "i32,") + 3})
	element := ast.NamedTypeReference("void", span("void"))
	pointer := ast.TypeReference{Kind: ast.TypeKindRawPointer, Text: "ptr mut void", Span: span("ptr mut void"), Mutable: true, Element: &element}
	callback := "unsafe extern C fn i32(i32,ptr mut void)"
	countStart := strings.Index(raw, "i32 count")
	for i, want := range []ast.TypeReference{
		{Kind: ast.TypeKindForeignFunction, Text: callback, Span: span(callback), Parameters: []ast.TypeReference{parameter, pointer}, Result: &result, ABI: "C"},
		ast.NamedTypeReference("i32", diagnostic.Span{StartByte: countStart, EndByte: countStart + 3}),
	} {
		gotJSON, err := json.Marshal(parsed.AST.Structs[0].Fields[i].Type)
		if err != nil {
			t.Fatal(err)
		}
		wantJSON, err := json.Marshal(want)
		if err != nil || !bytes.Equal(gotJSON, wantJSON) {
			t.Fatalf("old type JSON changed: %s want %s (%v)", gotJSON, wantJSON, err)
		}
	}
}

func candidateClosureTypeOracle(t *testing.T, root string) string {
	t.Helper()
	candidateClosureTypeLegacyJSON(t)
	var wire bytes.Buffer
	wire.WriteString("FYCLTYP1")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(text string) { number(len(text)); wire.WriteString(text) }
	sources := candidateClosureTypeSources()
	number(len(sources))
	total, closures := 0, 0
	for i, raw := range sources {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(raw)), source.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("Go type source %d: %+v", i, parsed.Tree.Diagnostics)
		}
		refs := candidateClosureTypeReferences(parsed.AST)
		field(raw)
		number(len(refs))
		for _, typ := range refs {
			kind := candidateClosureTypeKind(t, typ)
			number(kind)
			number(typ.Span.StartByte)
			number(typ.Span.EndByte)
			if raw[typ.Span.StartByte:typ.Span.EndByte] != typ.Text {
				t.Fatalf("Go written span lost: %+v", typ)
			}
			encoded, err := json.Marshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			field(string(encoded))
			total++
			if kind == 17 || kind == 15 {
				if typ.Result == nil {
					t.Fatal("missing structural function result")
				}
				number(typ.Result.Span.StartByte)
				number(typ.Result.Span.EndByte)
				number(len(typ.Parameters))
				for _, parameter := range typ.Parameters {
					number(parameter.Span.StartByte)
					number(parameter.Span.EndByte)
				}
				if kind == 17 {
					closures++
					if typ.ABI != "" {
						t.Fatal("safe closure aliased C ABI")
					}
				}
			}
		}
	}
	invalid := []string{"fn i32(i32", "fn i32(,i32)", "fn i32(i32 value)", "fn [i32 offset] i32(i32)", "unsafe fn i32(i32)", "extern C fn i32(i32)", "fn i32", "fn ()"}
	number(len(invalid))
	for _, signature := range invalid {
		prefix := "module sample; // 한글\r\nstruct Bad{"
		input := signature + " action;"
		raw := prefix + input + "}fn i32 tail(){return 42;}"
		tree := syntax.Parse(source.NewFile("sample.fy", []byte(raw)), source.DefaultLimits())
		if tree.ErrorFree() {
			t.Fatalf("Go malformed function type admitted: %s", signature)
		}
		field(input)
		expected := 0
		if signature == "fn i32(i32" {
			if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0007" {
				t.Fatalf("missing-close origin: %+v", tree.Diagnostics)
			}
			expected = tree.Diagnostics[0].Span.StartByte - len(prefix)
		}
		number(expected)
	}
	for _, bound := range []string{"depth", "nodes", "work"} {
		limits := source.DefaultLimits()
		switch bound {
		case "depth":
			limits.MaxDepth = 16
		case "nodes":
			limits.MaxNodes = 16
		case "work":
			limits.MaxWorkUnits = 48
		}
		raw := "module sample;struct Deep{" + strings.Repeat("fn ", 24) + "i32" + strings.Repeat("(i32)", 24) + " value;}"
		tree := syntax.Parse(source.NewFile("sample.fy", []byte(raw)), limits)
		if tree.ErrorFree() {
			t.Fatalf("Go %s budget bypassed", bound)
		}
	}
	path := filepath.Join(root, "closure-type-oracle.bin")
	if err := os.WriteFile(path, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("Go oracle: %d original source cases, %d structural types (%d safe signatures), %d malformed controls; old JSON and bounded syntax checked", len(sources), total, closures, len(invalid))
	return path
}

func TestCandidateClosureValueTypeOracle(t *testing.T) {
	root := t.TempDir()
	candidateClosureTypeOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_types.fy"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("tests/closure_value_types.fy", raw), source.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AST == nil || !parsed.Tree.ErrorFree() {
		t.Fatalf("new Fairy fixture syntax: %+v", parsed.Tree.Diagnostics)
	}
}

func TestCandidateClosureValueTypes(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("small parser/type native probe requires Windows AMD64; Linux deferred")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	input := candidateClosureTypeOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_types.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::closure_value_types;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(filesystem,console) fn i32 main(){closure_types_match_original_go_structure_and_json();closure_type_bindings_retain_written_types_without_execution();closure_type_parser_keeps_depth_and_node_bounds();closure_type_json_keeps_work_and_internal_link_boundaries();retained_closure_type_does_not_publish_an_executable_graph();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "syntax_parse_driver", "types", "generic_source_type_json", "generic_source_json", "module_types", "module_catalog", "scanner", "syntax", "file_source")
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("small parser/type probe source check: %d %s %s", code, out, diagnostic)
	}
	for _, args := range [][]string{{"run", "--", input}, {"run", "--release", "--locked", "--", input}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("small parser/type probe %v: %d %q %s", args, code, out, diagnostic)
		}
	}
}
