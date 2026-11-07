package cli

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Public text comes from the original frontend/HIR and interface extractor.
// The authored Fairy renderer receives these exact sources, never a Go-generated
// replacement for a candidate module or a fabricated positive graph/body.
func candidateClosurePublicTextOracle(t *testing.T, root string) string {
	t.Helper()
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{
		PackageUID: "example.closure-public-text", VersionIdentity: "1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	signatures := []string{
		"fn i32(i32)", "fn /* result */ i32(/* parameter */ i32,)",
		"fn\r\n i32(\r\n i32\r\n)",
		"i32", "i64", "u32", "bool", "Option<i32>", "Result<Option<i32>,u32>",
		"ref Leaf", "ref mut Leaf", "Leaf[2]", "ref Leaf[]", "ref mut Leaf[]",
		"ptr Leaf", "ptr mut Leaf", "Result<Option<ref Leaf>,u32>",
	}
	var wire bytes.Buffer
	wire.WriteString("FYCVPT01")
	number := func(n int) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, uint64(n)); err != nil {
			t.Fatal(err)
		}
	}
	field := func(value string) { number(len(value)); wire.WriteString(value) }
	number(len(signatures))
	field(coordinate.String())
	for index, signature := range signatures {
		declarations := "pub struct Leaf{i32 value;}"
		if index < 3 {
			declarations += "pub struct Job{fn /* field */ i32(i32) action;}"
		}
		raw := "module sample; // 원문 바이트\r\n" + declarations + "pub fn " + signature + " keep(" + signature + " action){return action;}"
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(raw)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("Go public text syntax %q: %v %+v", signature, err, parsed.Tree.Diagnostics)
		}
		typed := semantic.Analyze([]semantic.File{{Package: hir.DefaultPackage, Path: "sample.fy", Module: parsed.AST}})
		if len(typed.Diagnostics) != 0 || len(typed.Unsupported) != 0 {
			t.Fatalf("Go public text semantics %q: %+v %+v", signature, typed.Diagnostics, typed.Unsupported)
		}
		descriptor, err := moduleinterface.Extract(typed.HIR, hir.DefaultPackage, map[hir.PackageID]packagegraph.CoordinateID{hir.DefaultPackage: coordinate})
		if err != nil {
			t.Fatalf("Go public text extraction %q: %v", signature, err)
		}
		var result, parameter, fieldType string
		for _, module := range descriptor.Modules {
			for _, item := range module.Items {
				if item.Kind == "function" && item.Name == "keep" && len(item.Parameters) == 1 {
					result, parameter = item.ResultType, item.Parameters[0]
				}
				if item.Kind == "struct" && item.Name == "Job" && len(item.Fields) == 1 {
					fieldType = item.Fields[0].Type
				}
			}
		}
		if result == "" || parameter != result {
			t.Fatalf("Go public text missing identity signature %q: %+v", signature, descriptor)
		}
		if index < 3 && (result != "fn i32(i32)" || fieldType != result) {
			t.Fatalf("Go closure public text changed: result=%q field=%q", result, fieldType)
		}
		field(raw)
		field(result)
		field(parameter)
		field(fieldType)
		encoded, err := moduleinterface.Encode(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		hash, err := moduleinterface.Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		field(string(encoded))
		field(hash.String())
	}
	path := filepath.Join(root, "closure-value-public-text-oracle.bin")
	if err := os.WriteFile(path, wire.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("original Go interface oracle: 3 closure spellings, %d feature-free signatures", len(signatures)-3)
	return path
}

func TestCandidateClosureValuePublicTextOracle(t *testing.T) {
	candidateClosurePublicTextOracle(t, t.TempDir())
	for _, name := range []string{"src/module_public_type_text.fy", "tests/closure_value_public_text.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		path := name
		if strings.HasPrefix(name, "src/") {
			path = filepath.Base(name)
		}
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, data), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("authored public text syntax %s: %v %+v", name, err, parsed.Tree.Diagnostics)
		}
	}
}

func TestCandidateClosureValuePublicText(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("small native public text probe requires Windows AMD64; Linux remains batched")
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	input := candidateClosurePublicTextOracle(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "closure_value_public_text.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program := strings.Replace(string(raw), "module tests::closure_value_public_text;", "module main;", 1)
	program = strings.ReplaceAll(program, "@test", "") + "\n@effects(filesystem) fn i32 main(){closure_public_text_matches_go_and_preserves_legacy_bytes();closure_public_text_exact_work_and_errors();closure_public_text_rejects_forged_payloads_and_wrappers();closure_public_text_keeps_source_and_name_authentication();return 0;}\n"
	writeTestSource(t, root, "src/main.fy", program)
	copyCandidateParityModules(t, root, "module_public_type_text", "module_public_type_ids", "module_public_interface_model", "module_public_interface_extract", "module_public_interface", "module_signatures", "module_catalog", "module_types", "module_unit_interfaces", "module_unit_bind", "syntax", "scanner", "file_source")
	if code, out, diagnostic := runForTest([]string{"check"}, root); code != ExitSuccess || diagnostic != "" {
		t.Fatalf("small public text source check: %d %s %s", code, out, diagnostic)
	}
	for _, args := range [][]string{{"run", "--", input}, {"run", "--release", "--locked", "--", input}} {
		if code, out, diagnostic := runForTest(args, root); code != ExitSuccess || out != "" || diagnostic != "" {
			t.Fatalf("small public text probe %v: %d %q %s", args, code, out, diagnostic)
		}
		t.Logf("small authored public text native %v: exit 0, empty stdout/stderr", args)
	}
}
