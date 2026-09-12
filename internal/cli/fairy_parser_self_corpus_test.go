package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"github.com/0disoft/fairyc/internal/target"
)

const parserCorpusProgram = `module main;
import scanner as scanner; import syntax as syntax;
import integrity as integrity;
import std::bytes as bytes; import std::io as io;
@effects(console) fn void number(usize value) {
 assert(switch(io::write_usize(value)) { case .ok(_): { true } case .err(_): { false } });
 assert(switch(io::write(" ")) { case .ok(_): { true } case .err(_): { false } });
}
@effects(console) fn i32 report(syntax::Parsed parsed) {
 assert(integrity::valid(ref parsed));
 number(0); number(syntax::declaration_count(ref parsed)); number(syntax::problem_count(ref parsed));
 if (syntax::problem_count(ref parsed) > 0) {
  let syntax::Problem p = syntax::problem_at(ref parsed,0);
  number(p.expected as usize); number(p.start); number(p.end);
 } else { number(0); number(0); number(0); }
 var usize i=0;
 while(i<syntax::declaration_count(ref parsed)) {
  let syntax::Declaration d=syntax::declaration_at(ref parsed,i);
  number(d.kind as usize); number(d.start); number(d.end); number(d.name_start); number(d.name_end);
  i+=1;
 }
 return 0;
}
@effects(console) fn i32 main() {
 var bytes::Buffer input = bytes::empty();
 var usize length = 0;
 while (true) {
  var u8[1] data = [0];
  let usize count = switch(io::read_stdin(ref mut data)) { case .ok(n): { n } case .err(_): { 2 } };
  if (count == 0) { break; }
  if (count != 1) { return 8; }
  if (length >= 32768) { return 4; }
  if (!switch(bytes::append_byte(ref mut input,data[0])) { case .ok(_): { true } case .err(_): { false } }) { return 3; }
  length += 1;
 }
 let Result<syntax::Parsed,i32> parsed = switch(scanner::collect(input,false,32768,1048576,256,16777216)) {
  case .ok(stream): { syntax::parse(stream,512) } case .err(status): { .err(status) }
 };
 return switch(parsed) {
  case .ok(value): { report(value) }
  case .err(status): { assert(status == 3 || status == 4 || status == 5); number(if (status == 3) { 3 } else if (status == 4) { 4 } else { 5 }); number(0); number(0); number(0); number(0); number(0); 0 }
 };
}
`

func TestWindowsFairyParserSelfCorpus(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	example := filepath.Join("..", "..", "examples", "lexer_candidate", "src")
	files, err := os.ReadDir(example)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	parser, err := os.ReadFile(filepath.Join(example, "syntax.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/syntax.fy", string(parser))
	check, err := os.ReadFile(filepath.Join(example, "integrity.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "src/integrity.fy", string(check))
	writeTestSource(t, root, "src/main.fy", parserCorpusProgram)
	baseline := map[string]string{}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		// These unchanged sources have passed complete syntax. Keep this explicit
		// set so deletion or a newly unsupported construct cannot silently regress it.
		missing := map[string]bool{}
		for _, name := range []string{"attributes.fy", "bodies.fy", "budget.fy", "diagnostics.fy", "integrity.fy", "main.fy", "numbers.fy", "pages.fy", "parse_tokens.fy", "records.fy", "scanner.fy", "strings.fy", "syntax.fy", "types.fy", "utf8.fy"} {
			missing[name] = true
		}
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		observe := func(input []byte) []int {
			t.Helper()
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(input))
			if err != nil || out.ExitCode != 0 || out.Stderr != "" || out.TrapControl != "" {
				t.Fatalf("native corpus: %+v %v", out, err)
			}
			fields := strings.Fields(out.Stdout)
			if len(fields) < 6 {
				t.Fatalf("invalid observation: %q", out.Stdout)
			}
			result := make([]int, len(fields))
			for i, s := range fields {
				n, err := strconv.Atoi(s)
				if err != nil || n < 0 {
					t.Fatalf("invalid count %q", s)
				}
				result[i] = n
			}
			if len(result) != 6+5*result[1] {
				t.Fatalf("invalid declaration transport: %v", result)
			}
			return result
		}
		good := observe([]byte("module main; fn i32 f() { while(true) { break; } return a.b[0]; }"))
		bad := observe([]byte("module main; fn void f() { return 1 }"))
		if good[0] != 0 || good[2] != 0 || bad[0] != 0 || bad[2] != 1 || bad[3] != 8 {
			t.Fatalf("acceptance controls: %v %v", good, bad)
		}
		count := 0
		for _, file := range files {
			if file.Type()&os.ModeSymlink != 0 {
				t.Fatal("unexpected source symlink")
			}
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".fy") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(example, file.Name()))
			if err != nil || len(data) > 32768 {
				t.Fatalf("source %s: size=%d err=%v", file.Name(), len(data), err)
			}
			full := observe(data)
			if missing[file.Name()] {
				if full[0] != 0 || full[2] != 0 {
					t.Fatalf("complete-source regression %s: %v", file.Name(), full)
				}
				compareParserDeclarations(t, file.Name(), data, full[6:])
				delete(missing, file.Name())
			}
			if full[0] != 0 && full[0] != 4 {
				t.Fatalf("unexpected source outcome %s: %v", file.Name(), full)
			}
			first := full
			// Exhaustion publishes no tree. Probe only original, complete top-level
			// prefixes to locate the first blocking construct without inventing input.
			if full[0] == 4 {
				goResult, err := frontend.AnalyzeSyntax(source.NewFile(file.Name(), data), source.DefaultLimits())
				if err != nil || goResult.AST == nil {
					t.Fatalf("source oracle %s: %v", file.Name(), err)
				}
				top, _ := goResult.Tree.Node(goResult.Tree.Root)
				for _, id := range top.Children {
					node, ok := goResult.Tree.Node(id)
					if !ok {
						t.Fatal("missing source node")
					}
					first = observe(data[:node.Span.EndByte])
					if first[0] == 0 && first[2] > 0 {
						break
					}
				}
				if first[0] != 0 || first[2] == 0 {
					t.Fatalf("no bounded first blocker %s", file.Name())
				}
			}
			token := ""
			if first[2] > 0 {
				if first[4] < 0 || first[5] <= first[4] || first[5] > len(data) {
					t.Fatalf("invalid first-fault span %s: %v", file.Name(), first)
				}
				token = string(data[first[4]:first[5]])
			}
			row := fmt.Sprintf("bytes=%d full_status=%d declarations=%d problems=%d first_expected=%d first_span=%d..%d actual=%q prefix_probe=%t", len(data), full[0], full[1], full[2], first[3], first[4], first[5], token, full[0] == 4)
			if profile == cdriver.LowProfile {
				baseline[file.Name()] = row
				t.Logf("%s %s", file.Name(), row)
			} else if baseline[file.Name()] != row {
				t.Fatalf("profile mismatch %s", file.Name())
			}
			count++
		}
		if count == 0 {
			t.Fatal("empty self-source corpus")
		}
		if len(missing) != 0 {
			t.Fatalf("missing supported source files: %v", missing)
		}
		t.Logf("profile=%v files=%d; only status=0 AND problems=0 counts as complete syntax", profile, count)
	}
}

func compareParserDeclarations(t *testing.T, name string, data []byte, actual []int) {
	t.Helper()
	oracle, err := frontend.AnalyzeSyntax(source.NewFile(name, data), source.DefaultLimits())
	if err != nil || oracle.Tree == nil || len(oracle.Tree.Diagnostics) != 0 {
		t.Fatalf("declaration oracle %s: %v", name, err)
	}
	tree := oracle.Tree
	root, _ := tree.Node(tree.Root)
	want := make([]int, 0, len(root.Children)*5)
	for _, id := range root.Children {
		node, ok := tree.Node(id)
		if !ok {
			t.Fatal("missing oracle declaration")
		}
		kind := map[syntax.Kind]int{syntax.KindModule: 1, syntax.KindImport: 2, syntax.KindFunction: 3, syntax.KindStruct: 4, syntax.KindResource: 6, syntax.KindConst: 7}[node.Kind]
		if kind == 0 {
			t.Fatalf("unsupported declaration oracle %s: %s", name, node.Kind)
		}
		nameSpan := node.NameSpan
		if kind == 1 {
			path, ok := tree.Node(node.Children[0])
			if !ok {
				t.Fatal("missing module path")
			}
			nameSpan = path.Span
		}
		if kind == 2 {
			alias := false
			for _, token := range tree.Tokens[node.TokenStart:node.TokenEnd] {
				if token.Text(data) == "as" {
					alias = true
					continue
				}
				if alias && token.Kind == lexer.Identifier {
					nameSpan = token.Span
					break
				}
			}
		}
		want = append(want, kind, node.Span.StartByte, node.Span.EndByte, nameSpan.StartByte, nameSpan.EndByte)
	}
	if fmt.Sprint(actual) != fmt.Sprint(want) {
		t.Fatalf("declaration mismatch %s: got %v want %v", name, actual, want)
	}
}
