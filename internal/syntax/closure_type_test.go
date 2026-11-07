package syntax_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestClosureValueTypeSyntaxStructureAndSource(t *testing.T) {
	signature := "fn /* 결과 한글 */ Result<u32, u16>(i8, ref u8[], fn bool(u64, f32),)"
	text := "module main; // 한글\r\nstruct Job { " + signature + " action; }\r\n"
	tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if !tree.ErrorFree() {
		t.Fatal(tree.Diagnostics)
	}
	assertTreeInvariants(t, []byte(text), tree)
	var root *syntax.Node
	for i := range tree.Nodes {
		n := &tree.Nodes[i]
		if n.Kind == syntax.KindType && n.Text == signature {
			root = n
			break
		}
	}
	if root == nil || root.Cooked != string(syntax.TypeFormClosureFunction) || len(root.Children) != 4 {
		t.Fatalf("structural safe function type lost: %+v", root)
	}
	if root.Span.StartByte != strings.Index(text, signature) || text[root.Span.StartByte:root.Span.EndByte] != signature {
		t.Fatalf("original UTF-8/CRLF span lost: %+v", root.Span)
	}
	for i, want := range []string{"i8", "ref u8[]", "fn bool(u64, f32)", "Result<u32, u16>"} {
		child, ok := tree.Node(root.Children[i])
		if !ok || child.Kind != syntax.KindType || child.Text != want {
			t.Fatalf("signature child %d: %+v want %q", i, child, want)
		}
	}
	nested, _ := tree.Node(root.Children[2])
	if nested.Cooked != string(syntax.TypeFormClosureFunction) {
		t.Fatal("nested function type lost its distinct parser-owned form")
	}
}

func TestClosureValueTypeSyntaxRetentionContexts(t *testing.T) {
	for _, text := range []string{
		`module main;fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}fn i32 apply(fn i32(i32) action,i32 x){return action(x);}`,
		`module main;struct Job{fn void() empty;fn bool(i8,u64) predicate;fn fn i32(i32)(u32) factory;unsafe extern C fn i32(i32,ptr mut void) callback;}`,
		`module main;struct Job{ref fn i32(i32) borrowed;ptr fn i32(i32) raw;Option<fn i32(i32)> optional;(fn i32(i32))[2] array;}`,
	} {
		tree := syntax.Parse(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
		if !tree.ErrorFree() {
			t.Fatalf("type retention rejected syntax: %+v", tree.Diagnostics)
		}
		assertTreeInvariants(t, []byte(text), tree)
	}
}

func TestClosureValueTypeSyntaxKeepsPreviousRevisionClosed(t *testing.T) {
	input := []byte("module main;struct Job{fn i32(i32) action;}")
	tree := syntax.ParsePreviousRevision(source.NewFile("main.fy", input), source.DefaultLimits())
	if tree.ErrorFree() {
		t.Fatal("current-only safe function types leaked into the migration grammar")
	}
	assertTreeInvariants(t, input, tree)
}

func TestClosureValueTypeSyntaxInvalidRecoveryAndBounds(t *testing.T) {
	for _, signature := range []string{"fn i32(i32", "fn i32(,i32)", "fn i32(i32 value)", "fn [i32 offset] i32(i32)", "unsafe fn i32(i32)", "extern C fn i32(i32)"} {
		t.Run(signature, func(t *testing.T) {
			text := []byte("module main; // 한글\r\nstruct Job{" + signature + " action;}fn i32 tail(){return 42;}")
			first := parseWithinTestDeadline(t, "main.fy", text, source.DefaultLimits())
			second := parseWithinTestDeadline(t, "main.fy", text, source.DefaultLimits())
			if first.ErrorFree() || !reflect.DeepEqual(first, second) {
				t.Fatalf("invalid signature accepted or recovery nondeterministic: %+v", first.Diagnostics)
			}
			assertTreeInvariants(t, text, first)
			if signature == "fn i32(i32" {
				want := strings.Index(string(text), " action;") + 1
				if len(first.Diagnostics) != 1 || first.Diagnostics[0].Code != "E0007" || first.Diagnostics[0].Token != ")" || first.Diagnostics[0].Span.StartByte != want || first.Diagnostics[0].Span.EndByte != want {
					t.Fatalf("missing ')' repair lost original byte origin: %+v", first.Diagnostics)
				}
				found := false
				for _, node := range first.Nodes {
					if node.Kind == syntax.KindFunction && node.Text == "tail" {
						found = true
					}
				}
				if !found {
					t.Fatal("missing ')' recovery discarded the following ordinary function")
				}
			}
		})
	}
	makeInput := func(depth int) []byte {
		return []byte("module main;struct Job{" + strings.Repeat("fn ", depth) + "i32" + strings.Repeat("(i32)", depth) + " action;}")
	}
	admitted := parseWithinTestDeadline(t, "main.fy", makeInput(24), source.DefaultLimits())
	if !admitted.ErrorFree() {
		t.Fatal(admitted.Diagnostics)
	}
	for _, tc := range []struct {
		name  string
		depth int
		limit func(*source.Limits)
	}{
		{"default depth", 600, func(*source.Limits) {}},
		{"lowered depth", 24, func(l *source.Limits) { l.MaxDepth = 16 }},
		{"lowered nodes", 24, func(l *source.Limits) { l.MaxNodes = 16 }},
		{"lowered work", 24, func(l *source.Limits) { l.MaxWorkUnits = 48 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits := source.DefaultLimits()
			tc.limit(&limits)
			input := makeInput(tc.depth)
			tree := parseWithinTestDeadline(t, "main.fy", input, limits)
			assertSingleExhaustion(t, tree.Diagnostics)
			assertTreeInvariants(t, input, tree)
		})
	}
}
