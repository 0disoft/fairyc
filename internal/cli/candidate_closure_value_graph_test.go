package cli

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/closureprep"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

// The oracle uses the original Go structural AST and admitted HIR identity.
// Raw source bytes (including Korean/CRLF/trivia) are passed unchanged to Fairy.
func candidateClosureGraphOracle(t *testing.T, root string) string {
	t.Helper()
	signatures := []string{
		"fn i32(i32)",
		"fn /* result */ i32(/* parameter */ i32,)",
		"fn\r\n i32(\r\n i32\r\n)",
		"fn u32(i32)", "fn i32(u32)", "fn i32()", "fn i32(i32,i32)",
		"fn (i32)(i32)", "fn i32((i32))", "fn fn i32(i32)(i32)", "fn i32(fn i32(i32))",
		"(fn i32(i32))", "Option<fn i32(i32)>", "Result<fn i32(i32),u32>",
		"Result<i32,fn i32(i32)>", "ptr fn i32(i32)", "ref fn i32(i32)",
		"ref mut fn i32(i32)", "(fn i32(i32))[2]",
		"unsafe extern C fn i32(i32,ptr mut void)",
	}
	var wire bytes.Buffer
	wire.WriteString("FYCVGR01")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(s string) { number(len(s)); wire.WriteString(s) }
	number(len(signatures))
	admitted := 0
	for _, signature := range signatures {
		raw := "module sample; // 원문 바이트\r\nfn " + signature + " identity(" + signature + " action){return action;}"
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() || len(parsed.AST.Functions) != 1 {
			t.Fatalf("Go graph source %q: %v %+v", signature, err, parsed.Tree.Diagnostics)
		}
		typ := parsed.AST.Functions[0].ResultType
		if typ.Text != signature || raw[typ.Span.StartByte:typ.Span.EndByte] != signature {
			t.Fatalf("Go original graph source interval changed: %+v", typ)
		}
		field(raw)
		number(typ.Span.StartByte)
		number(typ.Span.EndByte)
		kind := candidateClosureTypeKind(t, typ)
		number(kind)
		if closureprep.ClosureValueType(typ) {
			result := semantic.Analyze([]semantic.File{{Package: hir.DefaultPackage, Path: "sample.fy", Module: parsed.AST}})
			if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
				t.Fatalf("Go admitted closure source %q: %+v %+v", signature, result.Diagnostics, result.Unsupported)
			}
			function := result.HIR.Modules[0].Functions[0]
			if function.ResultType != hir.TypeClosureI32 || len(function.Parameters) != 1 || function.Parameters[0].Type != hir.TypeClosureI32 || hir.IsCCallback(function.ResultType) || function.ResultType == hir.TypeCCallbackI32UserData {
				t.Fatalf("Go direct leaf lost distinct closure identity: %+v", function)
			}
			number(1)
			field(function.ResultType.String())
			admitted++
		} else {
			number(0)
			field("")
		}
		if kind == 17 {
			if typ.Result == nil {
				t.Fatal("Go safe function lost structural result")
			}
			number(typ.Result.Span.StartByte)
			number(typ.Result.Span.EndByte)
			number(len(typ.Parameters))
			for _, parameter := range typ.Parameters {
				number(parameter.Span.StartByte)
				number(parameter.Span.EndByte)
			}
		}
	}
	malformed := []string{"fn i32(i32", "fn i32(,i32)", "fn i32(i32 value)", "fn [i32 offset] i32(i32)", "unsafe fn i32(i32)", "extern C fn i32(i32)", "fn i32", "fn ()"}
	number(len(malformed))
	for _, signature := range malformed {
		prefix := "module sample; // 오류 위치\r\nstruct Broken{"
		input := signature + " action;"
		tree := syntax.Parse(source.NewFile("sample.fy", []byte(prefix+input+"}")), source.DefaultLimits())
		if tree.ErrorFree() {
			t.Fatalf("Go malformed closure graph control accepted: %q", signature)
		}
		field(input)
		start := 0
		if signature == "fn i32(i32" {
			if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0007" {
				t.Fatalf("Go malformed signature diagnostic changed: %+v", tree.Diagnostics)
			}
			start = tree.Diagnostics[0].Span.StartByte - len(prefix)
		}
		number(start)
	}
	path := filepath.Join(root, "closure-value-graph-oracle.bin")
	if err := os.WriteFile(path, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("Go oracle: %d written graph cases, %d distinct TypeClosureI32 admissions, %d malformed controls", len(signatures), admitted, len(malformed))
	return path
}

func TestCandidateClosureValueGraphOracle(t *testing.T) {
	candidateClosureGraphOracle(t, t.TempDir())
	for _, name := range []string{"src/module_types.fy", "src/closure_value_type_source.fy", "src/closure_value_type_nominal.fy", "tests/closure_value_graph.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		syntaxPath := name
		if strings.HasPrefix(name, "src/") {
			syntaxPath = filepath.Base(name)
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(syntaxPath, data), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("Fairy graph owner syntax %s: %v %+v", name, err, parsed.Tree.Diagnostics)
		}
	}
}

func TestCandidateClosureValueGraph(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("small native graph probe requires Windows AMD64; Linux remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	input := candidateClosureGraphOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_graph.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::closure_value_graph;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(filesystem) fn i32 main(){closure_graph_matches_go_identity_and_source_spans();closure_graph_preserves_roots_batches_and_source_authority();closure_graph_keeps_nominal_storage_and_generic_boundaries();closure_graph_validates_payloads_capacity_and_execution_separation();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "module_types", "module_catalog", "closure_value_type_source", "closure_value_type_nominal", "syntax", "syntax_parse_driver", "types", "scanner", "file_source", "module_unit_cleanup_mir_model", "module_unit_cleanup_mir_types")
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("small graph source check: %d %s %s", code, out, diagnostic)
	}
	for _, args := range [][]string{{"run", "--", input}, {"run", "--release", "--locked", "--", input}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("small graph probe %v: %d %q %s", args, code, out, diagnostic)
		}
		t.Logf("small graph native %v: exit 0, empty stdout/stderr", args)
	}
}
