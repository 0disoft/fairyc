package languageserver

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func completionSourceDocument(count int) *document {
	local := &ast.Module{Path: "main", Imports: []ast.Import{{Path: "util", Alias: "a"}}}
	view := &analysis{
		files: map[sourceKey]sourceView{
			{1, "main.fy"}: {Path: "main.fy", Module: local},
			{2, "util.fy"}: {Path: "util.fy", Module: &ast.Module{Path: "util", Functions: []ast.Function{
				{Name: "value", Public: true}, {Name: "private_value"},
			}}},
			{3, "util.fy"}: {Path: "other-util.fy", Module: &ast.Module{Path: "util", Functions: []ast.Function{{Name: "wrong_package", Public: true}}}},
		},
		imports: map[sourceKey]hir.PackageID{{1, "util"}: 2},
	}
	for i := range count {
		name := fmt.Sprintf("f%05d", i)
		view.files[sourceKey{1, name + ".fy"}] = sourceView{Path: name + ".fy", Module: &ast.Module{Path: name}}
	}
	return &document{Path: "main.fy", Text: []byte("a::"), View: view}
}

func TestCompletionSourcesKeepExactPackageAndPublicVisibility(t *testing.T) {
	d := completionSourceDocument(64)
	s := &server{}
	for range 3 {
		if got := s.completion(d, len(d.Text)); !slices.Equal(got, []Completion{{Label: "value", Kind: 3, Detail: "function"}}) {
			t.Fatalf("completion crossed package or visibility: %+v", got)
		}
	}
	// A fresh analysis must observe a changed exact import provider.
	next := completionSourceDocument(64)
	next.View.imports[sourceKey{1, "util"}] = 3
	d.View = next.View
	if got := s.completion(d, len(d.Text)); len(got) != 1 || got[0].Label != "wrong_package" {
		t.Fatalf("new analysis reused stale provider: %+v", got)
	}
}

func TestCompletionSourcesUseLastCleanLocalWithCurrentImports(t *testing.T) {
	d := completionSourceDocument(0)
	d.LastClean = d.View
	d.View = completionSourceDocument(0).View
	d.View.files[sourceKey{1, "main.fy"}] = sourceView{Path: "main.fy"}
	d.View.imports[sourceKey{1, "util"}] = 3
	s := &server{}
	for range 3 {
		got := s.completion(d, len(d.Text))
		if len(got) != 1 || got[0].Label != "wrong_package" {
			t.Fatalf("last clean local used stale import provider: %+v", got)
		}
	}
}

func TestCompletionSourcesKeepMissesAndInvalidateOnNewAnalysis(t *testing.T) {
	s := &server{}
	for _, absent := range []string{"local", "module", "provider"} {
		t.Run(absent, func(t *testing.T) {
			d := completionSourceDocument(0)
			switch absent {
			case "local":
				delete(d.View.files, sourceKey{1, "main.fy"})
			case "module":
				delete(d.View.files, sourceKey{2, "util.fy"})
			case "provider":
				delete(d.View.imports, sourceKey{1, "util"})
			}
			for range 3 {
				if got := s.completion(d, len(d.Text)); len(got) != 0 {
					t.Fatalf("missing %s guessed symbols: %+v", absent, got)
				}
			}
			d.View = completionSourceDocument(0).View
			if got := s.completion(d, len(d.Text)); len(got) != 1 || got[0].Label != "value" {
				t.Fatalf("new analysis retained a missing %s: %+v", absent, got)
			}
		})
	}
}

func BenchmarkCompletionWorkspaceSourcesCold(b *testing.B) {
	d := completionSourceDocument(8192)
	s := &server{}
	files, imports := d.View.files, d.View.imports
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		d.View = &analysis{files: files, imports: imports}
		if got := s.completion(d, len(d.Text)); len(got) != 1 {
			b.Fatalf("unexpected completion: %+v", got)
		}
	}
}

func BenchmarkCompletionWorkspaceSources(b *testing.B) {
	for _, count := range []int{64, 8192} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			d := completionSourceDocument(count)
			s := &server{}
			if got := s.completion(d, len(d.Text)); len(got) != 1 {
				b.Fatalf("unexpected completion: %+v", got)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if got := s.completion(d, len(d.Text)); len(got) != 1 {
					b.Fatalf("unexpected completion: %+v", got)
				}
			}
		})
	}
}
