package languageserver

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"sort"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
)

func TestCompletionCandidatesMatchSortedLastDeclarations(t *testing.T) {
	for _, count := range []int{0, 1, 255, 256, 257, 4096} {
		for _, order := range []string{"ascending", "descending", "shuffled"} {
			t.Run(fmt.Sprintf("%d/%s", count, order), func(t *testing.T) {
				input := make([]Completion, 0, 2*count)
				for i := range count {
					label := fmt.Sprintf("symbol_%06d", i)
					input = append(input, Completion{label, 3, "function"}, Completion{label, 6, "local"})
				}
				if order == "descending" {
					for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
						input[i], input[j] = input[j], input[i]
					}
				} else if order == "shuffled" {
					rand.New(rand.NewPCG(1, 2)).Shuffle(len(input), func(i, j int) { input[i], input[j] = input[j], input[i] })
				}
				// Independent reference: all declarations, last one wins, then sort.
				reference := make(map[string]Completion)
				var candidates completionCandidates
				for _, item := range input {
					reference[item.Label] = item
					candidates.add(item)
					if len(candidates.items) > 256 || len(candidates.labels) > 256 {
						t.Fatal("completion retention exceeded the response limit")
					}
				}
				want := make([]Completion, 0, len(reference))
				for _, item := range reference {
					want = append(want, item)
				}
				sort.Slice(want, func(i, j int) bool { return want[i].Label < want[j].Label })
				want = want[:min(len(want), 256)]
				if got := candidates.sorted(); !reflect.DeepEqual(got, want) {
					t.Fatalf("bounded completions differ: got %v, want %v", got, want)
				}
			})
		}
	}
}

func BenchmarkCompletionLargeModule(b *testing.B) {
	for _, count := range []int{64, 4096, 65536} {
		for _, reverse := range []bool{false, true} {
			b.Run(fmt.Sprintf("symbols=%d/reverse=%v", count, reverse), func(b *testing.B) {
				module := &ast.Module{Path: "main", Functions: make([]ast.Function, count)}
				for i := range count {
					index := i
					if reverse {
						index = count - 1 - i
					}
					module.Functions[i].Name = fmt.Sprintf("symbol_%06d", index)
				}
				d := &document{Path: "main.fy", Text: []byte("symbol_"), View: &analysis{files: map[sourceKey]sourceView{
					{Path: "main.fy"}: {Path: "main.fy", Module: module},
				}}}
				s := &server{}
				want := min(count, 256)
				if got := s.completion(d, len(d.Text)); len(got) != want || got[0].Label != "symbol_000000" {
					b.Fatalf("unexpected completion result: %v", got)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if got := s.completion(d, len(d.Text)); len(got) != want {
						b.Fatalf("completion count = %d, want %d", len(got), want)
					}
				}
			})
		}
	}
}
