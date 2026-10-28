package semantic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func borrowLiveFiles(tb testing.TB, text string) []File {
	tb.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		tb.Fatalf("parse borrow liveness: %v", err)
	}
	return []File{{Path: "main.fy", Module: parsed.AST}}
}

func TestBorrowLiveEmptyTransitions(t *testing.T) {
	for _, test := range []struct {
		name, text string
		rejected   bool
	}{
		{"read", "fn i32 read(ref i32 p){return *p;}", false},
		{"write", "fn void write(ref mut i32 p){*p=42;}", false},
		{"branches", "fn i32 read(ref i32 p, bool b){if(b){return *p;}return *p;}", false},
		{"loop", "fn i32 read(ref i32 p, bool b){while(b){if(b){break;}continue;}return *p;}", false},
		{"live-loan", "fn i32 read(){var i32 x=1;let ref i32 p=ref x;x=2;return *p;}", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := Analyze(borrowLiveFiles(t, "module main;"+test.text))
			if len(result.Unsupported) != 0 || !test.rejected && result.CompletedThrough != PhaseFlow || (len(result.Diagnostics) != 0) != test.rejected {
				t.Fatalf("flow = %v, diagnostics = %+v, unsupported = %+v", result.CompletedThrough, result.Diagnostics, result.Unsupported)
			}
			if test.rejected && (len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" || len(result.Diagnostics[0].Related) != 2) {
				t.Fatalf("live loan rejection changed: %+v", result.Diagnostics)
			}
		})
	}
}

func BenchmarkAnalyzeScalarLiveness(b *testing.B) {
	for _, count := range []int{64, 2048} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			text := "module main;fn i32 main(){var i32 x=0;" + strings.Repeat("x=x+1;", count) + "return x;}"
			files := borrowLiveFiles(b, text)
			b.ReportAllocs()
			for b.Loop() {
				result := Analyze(files)
				if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
					b.Fatalf("scalar analysis: %+v %+v", result.Diagnostics, result.Unsupported)
				}
			}
		})
	}
}

func BenchmarkBorrowScalarLiveness(b *testing.B) {
	text := "module main;fn i32 main(){var i32 x=0;" + strings.Repeat("x=x+1;", 2048) + "return x;}"
	a := newAnalyzer(borrowLiveFiles(b, text), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		b.Fatalf("scalar fixture: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	function := &module.file.Module.Functions[0]
	b.ReportAllocs()
	for b.Loop() {
		flow := borrowFlowAnalyzer{analyzer: a, module: module, function: function,
			liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap)}
		if live := flow.liveBlock(function.Body, nil, borrowLiveContext{}); len(live) != 0 {
			b.Fatalf("unexpected live borrows: %+v", live)
		}
	}
}
