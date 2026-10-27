package languageserver

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func linearDefinition(d *document, offset int) []Location {
	locations := []Location{}
	if d.View == nil {
		return locations
	}
	for _, r := range d.View.result.Resolutions {
		use, ok := d.View.file(r.Use)
		if !ok || use.Path != d.Path || offset < r.Use.StartByte || offset >= r.Use.EndByte {
			continue
		}
		target, ok := d.View.file(r.Target.Declaration)
		if !ok || target.Path == "" {
			continue
		}
		locations = append(locations, Location{fileURI(target.Path), d.View.sourceRange(target.Path, target.Text, r.Target.Declaration.StartByte, r.Target.Declaration.EndByte)})
	}
	return locations
}

func TestDefinitionLookupMatchesLinearSpans(t *testing.T) {
	view := &analysis{files: map[sourceKey]sourceView{
		{1, "main.fy"}: {Path: "main.fy", Text: []byte("target")},
		{2, "main.fy"}: {Path: "other.fy", Text: []byte("other")},
		{3, "main.fy"}: {Path: "main.fy", Text: []byte("shared physical path")},
		{4, "sdk.fy"}:  {Text: []byte("sdk")},
	}}
	random := rand.New(rand.NewSource(7))
	for index := range 600 {
		start := random.Intn(110) - 5
		use := hir.Span{Package: hir.PackageID(index%6 + 1), Source: "main.fy", StartByte: start, EndByte: start + random.Intn(25) - 3}
		if index%7 == 0 {
			use.Source = hir.QualifiedSourcePath(use.Package, use.Source)
			use.Package = 0
		}
		target := hir.Span{Package: 1, Source: "main.fy", StartByte: index % 5, EndByte: index%5 + 1}
		if index%11 == 0 {
			target = hir.Span{Package: 4, Source: "sdk.fy"}
		} else if index%13 == 0 {
			target.Source = "missing.fy"
		}
		view.result.Resolutions = append(view.result.Resolutions, semantic.Resolution{Use: use, Target: semantic.SymbolCoordinate{Declaration: target}})
	}
	s := &server{}
	for _, path := range []string{"main.fy", "other.fy", "missing.fy"} {
		d := &document{Path: path, View: view}
		for offset := -6; offset < 135; offset++ {
			want := linearDefinition(d, offset)
			if got := s.definition(d, offset); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s at %d: got %+v, want %+v", path, offset, got, want)
			}
		}
	}
	if got := s.definition(&document{}, 0); len(got) != 0 {
		t.Fatal("nil analysis returned definitions")
	}
}

func TestDefinitionIndexFollowsAnalysisReplacement(t *testing.T) {
	s := &server{}
	text := "module main;\nfn i32 helper() { return 1; }\nfn i32 main() { return helper(); }\n"
	d := &document{Path: "main.fy", Text: []byte(text)}
	d.View = s.analyze(d, "")
	before := s.definition(d, strings.LastIndex(text, "helper"))
	if len(before) != 1 || before[0].Range.Start.Line != 1 {
		t.Fatalf("initial definition: %+v", before)
	}
	old := d.View
	text = "// moved\n// again\n" + strings.ReplaceAll(text, "helper", "changed")
	d.Text = []byte(text)
	d.View = s.analyze(d, "")
	if d.View == old || d.View.definitions != nil {
		t.Fatal("replacement analysis retained the old definition index")
	}
	after := s.definition(d, strings.LastIndex(text, "changed"))
	if len(after) != 1 || after[0].Range.Start.Line != 3 {
		t.Fatalf("updated definition: %+v", after)
	}
	if d.View.definitions == nil {
		t.Fatal("definition request did not build its index")
	}
}

func definitionBenchmarkDocument(count int) (*document, int) {
	view := &analysis{files: make(map[sourceKey]sourceView)}
	for index := range 8 {
		path := fmt.Sprintf("file-%d.fy", index)
		view.files[sourceKey{1, path}] = sourceView{Path: filepath.Join("C:/fairyc-definition", path), Text: []byte("x")}
	}
	for index := range count {
		span := hir.Span{Package: 1, Source: fmt.Sprintf("file-%d.fy", index%8), StartByte: index * 3, EndByte: index*3 + 1}
		view.result.Resolutions = append(view.result.Resolutions, semantic.Resolution{Use: span, Target: semantic.SymbolCoordinate{Declaration: hir.Span{Package: 1, Source: "file-0.fy", EndByte: 1}}})
	}
	return &document{Path: view.files[sourceKey{1, fmt.Sprintf("file-%d.fy", (count-1)%8)}].Path, View: view}, (count - 1) * 3
}

func BenchmarkDefinitionLookup(b *testing.B) {
	for _, count := range []int{1000, 10000, 100000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			d, offset := definitionBenchmarkDocument(count)
			s := &server{}
			if got := s.definition(d, offset); len(got) != 1 {
				b.Fatal(got)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if got := s.definition(d, offset); len(got) != 1 {
					b.Fatal(got)
				}
			}
		})
	}
}

func BenchmarkDefinitionFirstLookup(b *testing.B) {
	for _, count := range []int{1000, 100000} {
		for _, indexed := range []bool{false, true} {
			b.Run(fmt.Sprintf("%d/indexed=%v", count, indexed), func(b *testing.B) {
				d, offset := definitionBenchmarkDocument(count)
				s := &server{}
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					d.View.definitions = nil
					d.View.positions = nil
					var got []Location
					if indexed {
						got = s.definition(d, offset)
					} else {
						got = linearDefinition(d, offset)
					}
					if len(got) != 1 {
						b.Fatal(got)
					}
				}
			})
		}
	}
}
