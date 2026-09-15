package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func testWindowsFairyFunctionTree(t *testing.T) {
	inputs := []string{
		"module main;fn void empty(){}",
		"module main; /* 헤더 */\r\nfn pkg::Value choose(i32 z,ref mut u8[] a,){let x=z;return a;}",
		"module main;fn Box<Pair<u8,i32>> f(ref ptr mut u8 p,unsafe extern C fn i32(ref u8[],) callback,(u8[0x10]) data){return callback(p);}",
		"module main;fn i32 f(i32 same,u8 same){var local=1;return same;}",
		"module main;fn void f(){" + strings.Repeat("x;", 126) + "return;}", // 256 records including header
		"module main;fn i32" + strings.Repeat("[]", 62) + " f(){}",          // deepest result node: 63
		"module main;fn void f(i32" + strings.Repeat("[]", 61) + " p){}",    // deepest parameter node: 63
		"module main;fn void f(){x" + strings.Repeat("?", 60) + ";}",        // body shares root depth
	}
	var params []string
	for i := 0; i < 126; i++ {
		params = append(params, fmt.Sprintf("i32 p%d", i))
	}
	inputs = append(inputs, "module main;fn void f("+strings.Join(params, ",")+"){return;}") // 256
	rejected := []string{
		"module main;pub fn void f(){}",
		"module main;unsafe fn void f(){}",
		"module main;fn void Owner::f(){}",
		"module main;fn T f<T>(T p){return p;}",
		"module main;@effects(console) fn void f(){}",
		"module main;@effects(console) pub unsafe fn T Owner::f<T>(ref mut T arg,){return arg;}",
		"module main;fn void f(i32){}",
		"module main;fn void f(i32 x,,){}",
		"module main;fn void f(i32 x {}",
		"module main;fn f(){}",
		"module main;fn void f(){" + strings.Repeat("x;", 127) + "}", // 257, not the old body-only boundary
		"module main;fn i32" + strings.Repeat("[]", 63) + " f(){}",
		"module main;fn void f(i32" + strings.Repeat("[]", 62) + " p){}",
		"module main;fn void f(){x" + strings.Repeat("?", 61) + ";}",
		"module main;fn void f(" + strings.Join(params, ",") + "){x;}", // 257 across header and body
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	for _, name := range []string{"syntax.fy", "integrity.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if name == "syntax.fy" {
			text += "\npub fn scanner::TokenInfo comparison_token(ref Parsed p,usize index){return scanner::token_info(ref p.source,index);}\n"
		}
		writeTestSource(t, root, "src/"+name, text)
	}
	writer := strings.ReplaceAll(candidateBodyTreeWriter, "statement_tree(ref parsed,d.body,ref mut budget,0)", "function_tree(ref parsed,ref mut budget)") + candidateFunctionTreeWriter
	program := parserCorpusProgram
	a, b := strings.Index(program, "@effects(console) fn i32 report("), strings.Index(program, "@effects(console) fn i32 main()")
	if a < 0 || b < a {
		t.Fatal("corpus driver boundary drift")
	}
	program = program[:a] + writer + program[b:]
	a = strings.Index(program, " return switch(parsed) {")
	if a < 0 {
		t.Fatal("corpus result boundary drift")
	}
	program = program[:a] + " return switch(parsed){case .ok(value):{report(value)}case .err(_):{4}};\n}\n"
	writeTestSource(t, root, "src/main.fy", program)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		for i, input := range append(append([]string{}, inputs...), rejected...) {
			want, refErr := referenceComparisonTree(input, true)
			if i < len(inputs) && refErr != nil {
				t.Fatalf("reference for %q: %v", input, refErr)
			}
			if i >= len(inputs) && (refErr == nil || want != "") {
				t.Fatalf("reference accepted %q", input)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewBufferString(input))
			if err != nil || out.Stderr != "" || out.TrapControl != "" || out.Stdout != want || (out.ExitCode == 0) != (refErr == nil) {
				t.Fatalf("function tree mismatch for %q: err=%v output=%+v want=%s", input, err, out, want)
			}
		}
		t.Logf("profile=%v complete functions=%d rejections=%d", profile, len(inputs), len(rejected))
	}
}
