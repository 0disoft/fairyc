package semantic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
)

func enumCopyGraphFiles(t testing.TB, text string) []File {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse enum graph: %v", err)
	}
	return []File{{Path: "main.fy", Module: parsed.AST}}
}

func TestEnumCopyGraphKeepsEveryInvalidPayloadDiagnostic(t *testing.T) {
	for _, declarations := range []string{
		`enum Bad { Value { ref i32 invalid; }; } enum Uses { Value { Bad first; Bad second; }; }`,
		`enum Bad { Value { Bad invalid; }; } enum Uses { Value { Bad first; Bad second; }; }`,
		`resource R { cleanup {} } enum Bad { Value { Option<R> invalid; }; } enum Uses { Value { Bad first; Bad second; }; }`,
	} {
		files := enumCopyGraphFiles(t, "module main;"+declarations+"fn i32 main(){return 0;}")
		// Enum iteration order must not turn a cached failed search into success.
		for range 8 {
			result := Analyze(files)
			if len(result.Diagnostics) != 3 || len(result.Unsupported) != 0 {
				t.Fatalf("expected three payload rejections: %+v", result.Diagnostics)
			}
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code != "E1004" {
					t.Fatalf("unexpected rejection: %+v", diagnostic)
				}
			}
		}
	}
}

func TestInertCopyGraphSharesCompoundResultsAndRejectsCycles(t *testing.T) {
	a := &analyzer{}
	value := hir.TypeI32
	for range 80 {
		value = hir.Option(hir.Result(value, value))
	}
	if !a.inertCopyable(value, make(map[hir.Type]uint8)) {
		t.Fatal("shared copy graph rejected")
	}
	bad := hir.ParseType("main::InvalidCopy")
	a.enumTypes = map[hir.Type]*enumInfo{bad: {orderedVariants: []enumVariant{{orderedFields: []resourceField{{typ: bad}}}}}}
	state := make(map[hir.Type]uint8)
	for range 2 {
		if a.inertCopyable(bad, state) {
			t.Fatal("cycle or cached rejection accepted")
		}
	}
	// A new pass must use the updated definitions, not a stale rejection.
	a.enumTypes[bad].orderedVariants = nil
	if !a.inertCopyable(bad, make(map[hir.Type]uint8)) {
		t.Fatal("new pass reused old rejection")
	}
}

func BenchmarkAnalyzeEnumCopyChain(b *testing.B) {
	for _, count := range []int{32, 256} {
		b.Run(fmt.Sprintf("enums=%d", count), func(b *testing.B) {
			var text strings.Builder
			text.WriteString("module main; enum E0 { Zero; }")
			for i := 1; i < count; i++ {
				fmt.Fprintf(&text, "enum E%d { Value { E%d item; }; }", i, i-1)
			}
			text.WriteString("fn i32 main(){return 0;}")
			files := enumCopyGraphFiles(b, text.String())
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result := Analyze(files)
				if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
					b.Fatalf("enum graph rejected: %d diagnostics, %d unsupported", len(result.Diagnostics), len(result.Unsupported))
				}
			}
		})
	}
}
