package semantic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
)

func regionOriginFiles(tb testing.TB, count int) []File {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;import std::alloc as alloc;fn void many(Region scratch){")
	for i := range count {
		fmt.Fprintf(&text, "let Result<ref i32,alloc::AllocError> v%d=scratch.place(%d);discard v%d;", i, i, i)
	}
	text.WriteString("}")
	return regionOriginTextFiles(tb, text.String())
}

func regionOriginTextFiles(tb testing.TB, text string) []File {
	tb.Helper()
	var files []File
	for _, input := range []struct{ path, text string }{
		{"main.fy", text},
		{"std/alloc.fy", "module std::alloc;pub struct AllocError{u32 status;}"},
	} {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(input.path, []byte(input.text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
			tb.Fatalf("parse region origins: %v", err)
		}
		files = append(files, File{Path: input.path, Module: parsed.AST})
	}
	return files
}

func TestRegionOriginsKeepShadowedLastUses(t *testing.T) {
	const text = `module main;import std::alloc as alloc;
fn void many(Region scratch){
let Result<ref i32,alloc::AllocError> value=scratch.place(1);
if(true){let Result<ref i32,alloc::AllocError> value=scratch.place(2);discard value;}
discard value;
}`
	result := Analyze(regionOriginTextFiles(t, text))
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || len(result.RegionOrigins) != 2 {
		t.Fatalf("region facts: %+v %+v %+v", result.RegionOrigins, result.Diagnostics, result.Unsupported)
	}
	wantStarts := []int{strings.LastIndex(text, "discard value") + len("discard "), strings.Index(text, "discard value") + len("discard ")}
	for i, fact := range result.RegionOrigins {
		if fact.LastUse == nil || fact.LastUse.Source != "main.fy" || fact.LastUse.StartByte != wantStarts[i] || fact.LastUse.EndByte != wantStarts[i]+len("value") || fact.Escapes {
			t.Fatalf("fact %d = %+v, last use = %+v, want start %d without escape", i, fact, fact.LastUse, wantStarts[i])
		}
	}
}

func TestRegionOriginsKeepLexicalEscapeIdentity(t *testing.T) {
	const text = `module main;import std::alloc as alloc;
fn Result<ref i32,alloc::AllocError> choose(Region left,Region right){
let Result<ref i32,alloc::AllocError> first=left.place(1);discard first;
discard right.place(2);
return left.place(3);
}`
	result := Analyze(regionOriginTextFiles(t, text))
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || len(result.RegionOrigins) != 3 {
		t.Fatalf("region facts: %+v %+v %+v", result.RegionOrigins, result.Diagnostics, result.Unsupported)
	}
	for i, fact := range result.RegionOrigins {
		// Both left allocations share the lexical origin despite different sites.
		if fact.Escapes != (i != 1) || (fact.LastUse != nil) != (i == 0) {
			t.Fatalf("fact %d changed escape or last-use presence: %+v", i, fact)
		}
	}
}

func BenchmarkAnalyzeRegionOrigins(b *testing.B) {
	for _, count := range []int{64, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			files := regionOriginFiles(b, count)
			b.ReportAllocs()
			for b.Loop() {
				result := Analyze(files)
				if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 || len(result.RegionOrigins) != count {
					b.Fatalf("region analysis: %+v %+v, facts=%d", result.Diagnostics, result.Unsupported, len(result.RegionOrigins))
				}
			}
		})
	}
}
