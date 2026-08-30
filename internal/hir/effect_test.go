package hir

import (
	"slices"
	"testing"
)

func TestStage0EffectsAreCanonicalAndClosed(t *testing.T) {
	t.Parallel()
	want := []Effect{
		EffectBlockContext,
		EffectClock,
		EffectConsole,
		EffectEmit,
		EffectEnvironment,
		EffectExternalCall,
		EffectFilesystem,
		EffectNetwork,
		EffectProcess,
		EffectRandom,
		EffectStorageRead,
		EffectStorageWrite,
	}
	got := Stage0Effects()
	if !slices.Equal(got, want) {
		t.Fatalf("Stage0Effects() = %v, want %v", got, want)
	}
	for _, effect := range want {
		if parsed, ok := ParseEffect(string(effect)); !ok || parsed != effect {
			t.Fatalf("ParseEffect(%q) = %q, %t", effect, parsed, ok)
		}
	}
	if _, ok := ParseEffect("unsafe"); ok {
		t.Fatal("ParseEffect(unsafe) accepted a boundary as an effect")
	}
}

func TestExperimentalOnchainEffectsClassifyAuthorityAndMutation(t *testing.T) {
	t.Parallel()
	wantAll := []Effect{
		EffectBlockContext,
		EffectClock,
		EffectConsole,
		EffectEmit,
		EffectEnvironment,
		EffectExternalCall,
		EffectFilesystem,
		EffectNetwork,
		EffectProcess,
		EffectRandom,
		EffectStorageRead,
		EffectStorageWrite,
	}
	if got := ExperimentalEffects(); !slices.Equal(got, wantAll) {
		t.Fatalf("ExperimentalEffects() = %v, want %v", got, wantAll)
	}
	for _, effect := range wantAll {
		if parsed, valid := ParseEffect(string(effect)); !valid || parsed != effect {
			t.Fatalf("ParseEffect(%q) = %q, %v", effect, parsed, valid)
		}
	}

	want := []Effect{
		EffectBlockContext,
		EffectEmit,
		EffectExternalCall,
		EffectStorageRead,
		EffectStorageWrite,
	}
	if got := OnchainEffects(); !slices.Equal(got, want) {
		t.Fatalf("OnchainEffects() = %v, want %v", got, want)
	}
	for _, effect := range want {
		if !IsOnchainEffect(effect) {
			t.Fatalf("IsOnchainEffect(%q) = false", effect)
		}
	}
	if IsOnchainEffect(EffectFilesystem) {
		t.Fatal("filesystem was classified as on-chain authority")
	}

	wantMutation := []Effect{EffectEmit, EffectExternalCall, EffectStorageWrite}
	if got := OnchainMutationEffects(); !slices.Equal(got, wantMutation) {
		t.Fatalf("OnchainMutationEffects() = %v, want %v", got, wantMutation)
	}
	for _, effect := range wantMutation {
		if !IsOnchainMutationEffect(effect) {
			t.Fatalf("IsOnchainMutationEffect(%q) = false", effect)
		}
	}
	if IsOnchainMutationEffect(EffectBlockContext) || IsOnchainMutationEffect(EffectStorageRead) {
		t.Fatal("read-only on-chain authority was classified as mutation-capable")
	}

	input := []Effect{
		EffectStorageWrite,
		EffectFilesystem,
		EffectEmit,
		EffectStorageWrite,
		EffectStorageRead,
	}
	if got := MutatingOnchainEffects(input); !slices.Equal(got, []Effect{EffectEmit, EffectStorageWrite}) {
		t.Fatalf("MutatingOnchainEffects() = %v", got)
	}
}

func TestOnchainViewEligibilityIsDerivedFromEffects(t *testing.T) {
	t.Parallel()
	readOnly := [][]Effect{
		nil,
		{EffectBlockContext},
		{EffectStorageRead, EffectBlockContext},
		{EffectFilesystem, EffectStorageRead},
	}
	for _, effects := range readOnly {
		if !OnchainEffectsAreReadOnly(effects) {
			t.Fatalf("OnchainEffectsAreReadOnly(%v) = false", effects)
		}
	}

	mutating := [][]Effect{
		{EffectEmit},
		{EffectExternalCall},
		{EffectStorageWrite},
		{EffectStorageRead, EffectStorageWrite},
	}
	for _, effects := range mutating {
		if OnchainEffectsAreReadOnly(effects) {
			t.Fatalf("OnchainEffectsAreReadOnly(%v) = true", effects)
		}
	}
}

func TestMissingEffectsCanonicalizesInputs(t *testing.T) {
	t.Parallel()
	declared := []Effect{EffectConsole, EffectConsole}
	required := []Effect{EffectRandom, EffectFilesystem, EffectRandom, EffectConsole}
	want := []Effect{EffectFilesystem, EffectRandom}
	if got := MissingEffects(declared, required); !slices.Equal(got, want) {
		t.Fatalf("MissingEffects() = %v, want %v", got, want)
	}
}
