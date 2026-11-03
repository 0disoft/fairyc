package prevrevision

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

func spanMappingFixture(count int) SpanMapper {
	input := bytes.Repeat([]byte("abcdef\n"), count)
	declarations, surface := make([]edit, count), make([]edit, count)
	for i := range count {
		declarations[i] = edit{i*7 + 1, i*7 + 2, "XYZ"}
		surface[i] = edit{i*9 + 4, i*9 + 5, "q"}
	}
	return SpanMapper{
		original: input, intermediate: bytes.Repeat([]byte("aXYZcdef\n"), count),
		final: bytes.Repeat([]byte("aXYZqdef\n"), count), declarations: declarations, surface: surface,
	}
}

func TestSpanMapperTwoPassMatchesUnorderedMapping(t *testing.T) {
	mapper := spanMappingFixture(3)
	// Reference calls intentionally receive reverse-ordered plans.
	declarations, surface := slices.Clone(mapper.declarations), slices.Clone(mapper.surface)
	slices.Reverse(declarations)
	slices.Reverse(surface)
	for start := -1; start <= len(mapper.original)+1; start++ {
		for end := start; end <= len(mapper.original)+1; end++ {
			wantStart, wantEnd, wantErr := mapSpan(mapper.original, mapper.intermediate, declarations, start, end)
			if wantErr == nil {
				wantStart, wantEnd, wantErr = mapSpan(mapper.intermediate, mapper.final, surface, wantStart, wantEnd)
			}
			gotStart, gotEnd, err := mapper.Map(start, end)
			if gotStart != wantStart || gotEnd != wantEnd || fmt.Sprint(err) != fmt.Sprint(wantErr) {
				t.Fatalf("span %d..%d: got %d..%d/%v, want %d..%d/%v", start, end, gotStart, gotEnd, err, wantStart, wantEnd, wantErr)
			}
		}
	}
}

func TestPreparedSpanPlanOwnsEditsAndKeepsInsertionBias(t *testing.T) {
	edits := []edit{{5, 5, ";"}, {0, 0, "return "}}
	before := slices.Clone(edits)
	mapper := SpanMapper{
		original: []byte("value"), intermediate: []byte("return value;"), final: []byte("return value;"),
		declarations: ascendingSpanEdits(edits),
	}
	if !slices.Equal(edits, before) {
		t.Fatal("preparing a plan reordered caller-owned edits")
	}
	edits[0].replacement = "changed by caller"
	for range 3 {
		start, end, err := mapper.Map(0, 5)
		if err != nil || start != 7 || end != 12 {
			t.Fatalf("prepared plan lost ownership or insertion bias: %d..%d/%v", start, end, err)
		}
	}
}

func TestBuildSpanMapperOwnsOriginalBytes(t *testing.T) {
	input := []byte("module main;\nfn i32 main() { return 42; }\n")
	output, mapper, err := BuildSpanMapper("main.fy", input)
	if err != nil || !bytes.Equal(input, output) {
		t.Fatalf("build mapper: %q/%v", output, err)
	}
	input[0] = '!'
	if mapper.original[0] != 'm' {
		t.Fatal("mapper retained caller-owned source bytes")
	}
	start, end, err := mapper.Map(0, len(output))
	if err != nil || start != 0 || end != len(output) {
		t.Fatalf("unchanged source mapping: %d..%d/%v", start, end, err)
	}
}

func BenchmarkSpanMapperPreparedPlan(b *testing.B) {
	for _, count := range []int{1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			mapper := spanMappingFixture(count)
			start := len(mapper.original) - 2
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				gotStart, gotEnd, err := mapper.Map(start, start+1)
				if err != nil || gotStart != len(mapper.final)-2 || gotEnd != len(mapper.final)-1 {
					b.Fatalf("mapping: %d..%d/%v", gotStart, gotEnd, err)
				}
			}
		})
	}
}
