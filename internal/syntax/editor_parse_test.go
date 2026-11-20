package syntax

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestEditorParseMatchesBatchAcrossRevisions(t *testing.T) {
	const prefix = "module main;\n/// stable docs\nfn i32 first(Option<Option<i32>> value) { return 1; }\n"
	const second = "fn i32 second() { return "
	limits := source.DefaultLimits()
	var previous *EditorParseSnapshot
	for _, step := range []struct {
		text   string
		reused int
		clean  bool
	}{
		{prefix + second + "2; }\n", 0, true},
		{prefix + second + "200; }\n", 1, true},
		{prefix + second + "300; }\n", 1, true},
		{strings.Replace(prefix, "stable docs", "updated docs", 1) + second + "400; }\n", 0, true},
		{strings.Replace(prefix, "stable docs", "updated docs", 1) + second + "500 }\n", 0, false},
	} {
		file := source.NewFile("main.fy", []byte(step.text))
		got, next := ParseEditor(file, limits, previous)
		if want := Parse(file, limits); !reflect.DeepEqual(got, want) {
			t.Fatalf("incremental and batch syntax differ for %q", step.text)
		}
		if (next != nil) != step.clean || next != nil && next.ReusedItems() != step.reused {
			t.Fatalf("clean=%v, expected clean=%v and %d reused items", next != nil, step.clean, step.reused)
		}
		previous = next
	}
}

func TestEditorParseMatchesBatchAtTokenLimit(t *testing.T) {
	const prefix = "module main; fn i32 first(Option<Option<i32>> value) { return 1; } "
	const tail = "fn i32 second() { return 2; }"
	file := source.NewFile("main.fy", []byte(prefix+tail))
	tree := Parse(file, source.DefaultLimits())
	limits := source.DefaultLimits()
	// Allow the original tokens and one parser-owned split of >>.
	limits.MaxTokens = len(tree.Tokens)
	_, previous := ParseEditor(file, limits, nil)
	if previous == nil {
		t.Fatal("initial parse exceeded exact token limit")
	}
	// Lexing still fits, but the reused prefix's split exceeds the shared budget.
	updated := source.NewFile("main.fy", []byte(prefix+tail+" "))
	got, next := ParseEditor(updated, limits, previous)
	want := Parse(updated, limits)
	if next != nil || want.ErrorFree() || !reflect.DeepEqual(got, want) {
		t.Fatalf("token budget: got %v, batch %v", got.Diagnostics, want.Diagnostics)
	}
}

func TestEditorParseRebasesSuffixAcrossRevisions(t *testing.T) {
	const header = "module main;\nfn i32 first() { return 1; }\n"
	const tail = "/// Tail documentation\nfn i32 tail(Option<Option<i32>> value) { return 7; }\n" +
		"resource Token { i32 id; cleanup {} }\nfn i32 last() { return 9; }\n"
	const oldItem = "fn i32 changed() { return 2; }\n"
	const newItems = "fn i32 inserted() { return 3; }\nfn i32 changed() { return 200 + 1; }\n"
	limits := source.DefaultLimits()
	initial := source.NewFile("main.fy", []byte(header+oldItem+tail))
	_, previous := ParseEditor(initial, limits, nil)
	for _, step := range []struct {
		text string
		want int
	}{
		{header + newItems + tail, 3},
		{header + newItems + strings.Replace(tail, "/// Tail", "// Tail", 1), 3},
		{header + newItems + strings.Replace(tail, "return 7", "return 71", 1), 2},
		{header + oldItem + tail, 2},
		{header + oldItem + tail, 0},
	} {
		file := source.NewFile("main.fy", []byte(step.text))
		got, next := ParseEditor(file, limits, previous)
		if want := Parse(file, limits); !reflect.DeepEqual(got, want) {
			t.Fatalf("incremental and batch differ for %q: diagnostics %v/%v", step.text, got.Diagnostics, want.Diagnostics)
		}
		if next == nil || next.reusedSuffixItems != step.want {
			t.Fatalf("suffix reuse for %q: got %v, want %d", step.text, next.reusedSuffixItems, step.want)
		}
		previous = next
	}
}

func TestEditorParseSuffixMatchesBatchAtNodeLimit(t *testing.T) {
	const header = "module main;\n"
	const item = "fn i32 changed() { return 1; }\n"
	const tail = "fn i32 tail() { return 2; }\nfn i32 last() { return 3; }\n"
	initial := source.NewFile("main.fy", []byte(header+item+tail))
	limits := source.DefaultLimits()
	limits.MaxNodes = len(Parse(initial, limits).Nodes)
	_, previous := ParseEditor(initial, limits, nil)
	if previous == nil {
		t.Fatal("initial parse did not fit")
	}
	updated := source.NewFile("main.fy", []byte(header+strings.Replace(item, "return 1", "return 1 + 2", 1)+tail))
	got, next := ParseEditor(updated, limits, previous)
	want := Parse(updated, limits)
	if next != nil || want.ErrorFree() || !reflect.DeepEqual(got, want) {
		t.Fatalf("node budget: got %v, batch %v", got.Diagnostics, want.Diagnostics)
	}
}

func BenchmarkEditorParseTailEdit(b *testing.B) {
	var src strings.Builder
	src.WriteString("module main;\n")
	for i := range 500 {
		fmt.Fprintf(&src, "fn i32 helper_%d(i32 value) { let i32 local = value + %d; return local; }\n", i, i)
	}
	src.WriteString("fn i32 main() { return helper_499(")
	limits := source.DefaultLimits()
	file := source.NewFile("main.fy", []byte(src.String()+"1); }\n"))
	_, previous := ParseEditor(file, limits, nil)
	updated := source.NewFile("main.fy", []byte(src.String()+"2); }\n"))
	for _, mode := range []struct {
		name     string
		snapshot *EditorParseSnapshot
	}{{"batch", nil}, {"resume", previous}} {
		b.Run(mode.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tree, next := ParseEditor(updated, limits, mode.snapshot)
				if !tree.ErrorFree() || mode.snapshot != nil && next.ReusedItems() != 500 {
					b.Fatal("editor parse did not reuse the expected prefix")
				}
			}
		})
	}
}
