package semanticsnapshot

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
)

type cleanupSelectionFixture struct {
	result          compiler.Result
	implementations map[DeclarationKey]SpecializationSummary
	metadata        map[hir.PackageID]packageMetadata
	symbols         map[hir.SymbolID]DeclarationKey
	private         map[hir.SymbolID]struct{}
}

func newCleanupSelectionFixture() cleanupSelectionFixture {
	original := DeclarationKey{PackageInstance: "instance", Module: "main", Kind: DeclarationResource, Name: "Token"}
	generated := original
	generated.Name = "_fy_generic_token"
	span := hir.Span{Source: "main.fy", EndByte: 80}
	return cleanupSelectionFixture{
		result: compiler.Result{
			MIR: mir.Program{Functions: []mir.Function{{
				Symbol: 1, Name: "drop_token", DropGlueFor: hir.ParseType("main::_fy_generic_token"), Span: span,
				Blocks: []mir.Block{
					{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: 2, Span: span, Successors: []mir.BlockID{1}}},
					{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}},
				},
			}}},
			Semantic: semantic.Result{Borrows: []semantic.BorrowFact{
				{Function: 2, Borrow: span, Kind: "mutable"},
				{Function: 1, Borrow: span, Kind: "shared"},
			}},
		},
		implementations: map[DeclarationKey]SpecializationSummary{generated: {Declaration: original, Aggregate: &Declaration{
			Type: TypeRef{Kind: TypeNominal, Name: "main::Token", PackageCoordinate: "coordinate"}, DropKind: "non-empty",
		}}},
		metadata: map[hir.PackageID]packageMetadata{hir.DefaultPackage: {instance: "instance", coord: "coordinate"}},
		symbols:  map[hir.SymbolID]DeclarationKey{2: {PackageInstance: "instance", Module: "main", Kind: DeclarationFunction, Name: "callee"}},
		private:  map[hir.SymbolID]struct{}{1: {}},
	}
}

func (fixture cleanupSelectionFixture) build() (map[DeclarationKey]FunctionSummary, []semantic.BorrowFact, int, int, error) {
	return buildGenericCleanupSummaries(fixture.result, fixture.implementations, fixture.metadata, fixture.symbols, nil, fixture.private)
}

func TestGenericCleanupSelectionPreservesInputsAndCallTargets(t *testing.T) {
	t.Parallel()
	fixture := newCleanupSelectionFixture()
	before := newCleanupSelectionFixture()
	summaries, ordinary, resolved, external, err := fixture.build()
	if err != nil || len(summaries) != 1 || resolved != 1 || external != 0 {
		t.Fatalf("summaries=%+v resolved=%d external=%d error=%v", summaries, resolved, external, err)
	}
	if !reflect.DeepEqual(ordinary, fixture.result.Semantic.Borrows[:1]) {
		t.Fatalf("ordinary borrows = %+v", ordinary)
	}
	for key, summary := range summaries {
		if summary.Function.Name != "Token" || len(summary.Borrows) != 1 || summary.Borrows[0].Kind != "shared" ||
			len(summary.Calls) != 1 || summary.Calls[0].Target != fixture.symbols[2] || summary.Calls[0].TargetKind != CallTargetResolved {
			t.Fatalf("cleanup %v: %+v", key, summary)
		}
	}
	if !reflect.DeepEqual(fixture, before) {
		t.Fatal("cleanup selection modified caller-owned compiler facts or symbols")
	}
}

func TestGenericCleanupSelectionSkipsUnrelatedBodiesAndCopiesBorrowPartition(t *testing.T) {
	t.Parallel()
	fixture := newCleanupSelectionFixture()
	fixture.implementations = nil
	fixture.private = nil
	fixture.result.MIR.Functions[0].Span.Source = ""
	summaries, ordinary, resolved, external, err := fixture.build()
	if err != nil || len(summaries) != 0 || resolved != 0 || external != 0 {
		t.Fatalf("unrelated cleanup: summaries=%+v counts=%d/%d error=%v", summaries, resolved, external, err)
	}
	if !reflect.DeepEqual(ordinary, fixture.result.Semantic.Borrows) {
		t.Fatalf("ordinary borrow order changed: %+v", ordinary)
	}
	ordinary[0].Kind = "changed"
	if fixture.result.Semantic.Borrows[0].Kind != "mutable" {
		t.Fatal("returned borrow partition aliases caller input")
	}
}

func TestGenericCleanupSelectionErrorOrder(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		change func(*cleanupSelectionFixture)
		want   string
	}{
		{
			name: "unselected cleanup still validates its type",
			change: func(f *cleanupSelectionFixture) {
				f.implementations = nil
				f.result.MIR.Functions[0].DropGlueFor = hir.ParseType("$p99::main::Token")
			},
			want: `nominal type "$p99::main::Token" references package 99 without metadata`,
		},
		{
			name: "private owner before span",
			change: func(f *cleanupSelectionFixture) {
				f.private = nil
				f.result.MIR.Functions[0].Span.Source = ""
			},
			want: "generic cleanup has no private compiler owner",
		},
		{
			name: "duplicate before second span",
			change: func(f *cleanupSelectionFixture) {
				second := f.result.MIR.Functions[0]
				second.Span.Source = ""
				f.result.MIR.Functions = append(f.result.MIR.Functions, second)
			},
			want: "generic resource has duplicate cleanup implementations",
		},
		{
			name: "required cleanup after selection",
			change: func(f *cleanupSelectionFixture) {
				f.result.MIR.Functions = nil
			},
			want: "non-empty aggregate cleanup has no executable summary",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCleanupSelectionFixture()
			test.change(&fixture)
			summaries, ordinary, resolved, external, err := fixture.build()
			if err == nil || err.Error() != test.want || summaries != nil || ordinary != nil || resolved != 0 || external != 0 {
				t.Fatalf("error=%v, want %q; summaries=%v ordinary=%v counts=%d/%d", err, test.want, summaries, ordinary, resolved, external)
			}
		})
	}
}

func BenchmarkGenericCleanupUnselectedSymbols(b *testing.B) {
	for _, count := range []int{32, 4096, 16384} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			symbols := make(map[hir.SymbolID]DeclarationKey, count)
			for index := range count {
				symbols[hir.SymbolID(index+1)] = DeclarationKey{Module: "main", Kind: DeclarationFunction, Name: fmt.Sprint(index)}
			}
			b.ReportAllocs()
			for b.Loop() {
				summaries, ordinary, resolved, external, err := buildGenericCleanupSummaries(compiler.Result{}, nil, nil, symbols, nil, nil)
				if err != nil || len(summaries) != 0 || len(ordinary) != 0 || resolved != 0 || external != 0 {
					b.Fatalf("unselected cleanup changed: %v", err)
				}
			}
		})
	}
}
