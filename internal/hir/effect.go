package hir

import "slices"

// Effect identifies one source-visible external capability used by a function.
type Effect string

const (
	EffectBlockContext Effect = "block_context"
	EffectClock        Effect = "clock"
	EffectConsole      Effect = "console"
	EffectEmit         Effect = "emit"
	EffectEnvironment  Effect = "environment"
	EffectExternalCall Effect = "external_call"
	EffectFilesystem   Effect = "filesystem"
	EffectNetwork      Effect = "network"
	EffectProcess      Effect = "process"
	EffectRandom       Effect = "random"
	EffectStorageRead  Effect = "storage_read"
	EffectStorageWrite Effect = "storage_write"
)

var stage0Effects = []Effect{
	EffectClock,
	EffectConsole,
	EffectEnvironment,
	EffectFilesystem,
	EffectNetwork,
	EffectProcess,
	EffectRandom,
}

var experimentalEffects = []Effect{
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

var onchainEffects = []Effect{
	EffectBlockContext,
	EffectEmit,
	EffectExternalCall,
	EffectStorageRead,
	EffectStorageWrite,
}

// external_call remains mutation-capable until a target proves a static-call boundary.
var onchainMutationEffects = []Effect{
	EffectEmit,
	EffectExternalCall,
	EffectStorageWrite,
}

// Stage0Effects returns the closed canonical Stage 0 effect vocabulary in lexical order.
func Stage0Effects() []Effect {
	return slices.Clone(stage0Effects)
}

// ExperimentalEffects returns the complete experimental vocabulary in lexical order.
func ExperimentalEffects() []Effect {
	return slices.Clone(experimentalEffects)
}

// OnchainEffects returns the target-neutral on-chain authority vocabulary in lexical order.
func OnchainEffects() []Effect {
	return slices.Clone(onchainEffects)
}

// OnchainMutationEffects returns effects that a conservative view entry may not declare.
func OnchainMutationEffects() []Effect {
	return slices.Clone(onchainMutationEffects)
}

// IsOnchainEffect reports whether effect represents target-neutral chain authority.
func IsOnchainEffect(effect Effect) bool {
	return slices.Contains(onchainEffects, effect)
}

// IsOnchainMutationEffect reports whether effect may change consensus-visible state or behavior.
func IsOnchainMutationEffect(effect Effect) bool {
	return slices.Contains(onchainMutationEffects, effect)
}

// MutatingOnchainEffects returns the canonical mutation-capable subset of effects.
func MutatingOnchainEffects(effects []Effect) []Effect {
	var result []Effect
	for _, effect := range CanonicalEffects(effects) {
		if IsOnchainMutationEffect(effect) {
			result = append(result, effect)
		}
	}
	return result
}

// OnchainEffectsAreReadOnly reports whether effects contain consensus-visible mutation authority.
// A target separately rejects host-only authority that it cannot implement deterministically.
func OnchainEffectsAreReadOnly(effects []Effect) bool {
	return len(MutatingOnchainEffects(effects)) == 0
}

// ParseEffect returns one exact compiler-known effect. Stable package analysis separately keeps
// experimental on-chain spellings behind the current language-revision boundary.
func ParseEffect(value string) (Effect, bool) {
	effect := Effect(value)
	return effect, slices.Contains(experimentalEffects, effect)
}

// CanonicalEffects returns a sorted, duplicate-free copy of effects.
func CanonicalEffects(effects []Effect) []Effect {
	result := slices.Clone(effects)
	slices.Sort(result)
	return slices.Compact(result)
}

// MissingEffects returns every required effect absent from declared, in canonical order.
func MissingEffects(declared, required []Effect) []Effect {
	declaredSet := make(map[Effect]struct{}, len(declared))
	for _, effect := range declared {
		declaredSet[effect] = struct{}{}
	}
	var missing []Effect
	for _, effect := range CanonicalEffects(required) {
		if _, found := declaredSet[effect]; !found {
			missing = append(missing, effect)
		}
	}
	return missing
}
