package hir

import (
	"slices"
	"testing"
)

func TestStage0EffectsAreCanonicalAndClosed(t *testing.T) {
	t.Parallel()
	want := []Effect{
		EffectClock,
		EffectConsole,
		EffectEnvironment,
		EffectFilesystem,
		EffectNetwork,
		EffectProcess,
		EffectRandom,
	}
	if got := Stage0Effects(); !slices.Equal(got, want) {
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

func TestMissingEffectsCanonicalizesInputs(t *testing.T) {
	t.Parallel()
	declared := []Effect{EffectConsole, EffectConsole}
	required := []Effect{EffectRandom, EffectFilesystem, EffectRandom, EffectConsole}
	want := []Effect{EffectFilesystem, EffectRandom}
	if got := MissingEffects(declared, required); !slices.Equal(got, want) {
		t.Fatalf("MissingEffects() = %v, want %v", got, want)
	}
}
