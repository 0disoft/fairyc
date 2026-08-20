package semantic_test

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestAnalyzeCanonicalizesOnchainEffectsIntoHIR(t *testing.T) {
	t.Parallel()
	text := `module main;

@effects(storage_write, block_context, external_call, emit, storage_read)
fn i32 dispatch() {
    return 42;
}
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	assertCleanTypeResult(t, result)
	want := []hir.Effect{
		hir.EffectBlockContext,
		hir.EffectEmit,
		hir.EffectExternalCall,
		hir.EffectStorageRead,
		hir.EffectStorageWrite,
	}
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 ||
		!slices.Equal(result.HIR.Modules[0].Functions[0].Effects, want) {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	if len(result.Effects) != 1 || !slices.Equal(result.Effects[0].Effects, want) {
		t.Fatalf("effect facts = %+v", result.Effects)
	}
}

func TestAnalyzeRejectsMissingOnchainCallAuthority(t *testing.T) {
	t.Parallel()
	text := `module main;

@effects(storage_write)
fn i32 write_state() { return 1; }

@effects(storage_read)
fn i32 query() { return write_state(); }
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7102" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeAcceptsOnchainAuthoritySuperset(t *testing.T) {
	t.Parallel()
	text := `module main;

@effects(storage_read)
fn i32 read_state() { return 20; }

@effects(storage_write)
fn i32 write_state() { return 22; }

@effects(block_context, storage_read, storage_write)
fn i32 dispatch() { return read_state() + write_state(); }
`
	assertCleanTypeResult(t, analyzeText(t, map[string]string{"main.fy": text}))
}

func TestAnalyzeKeepsOnchainEffectVocabularyClosed(t *testing.T) {
	t.Parallel()
	text := `module main;

@effects(storage_delete)
fn i32 invalid() { return 0; }
`
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7101" {
		t.Fatalf("result = %+v", result)
	}
}
