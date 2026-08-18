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

// external_call is deliberately mutation-capable. Until a target proves a static-call boundary,
// the callee may mutate its own state, re-enter the caller, or emit consensus-visible output.
var onchainMutationEffects = []Effect{
	EffectEmit,
	EffectExternalCall,
	EffectStorageWrite,
}

// Stage0Effects returns the closed canonical effect vocabulary in lexical order.
func Stage0Effects() []Effect {
	return slices.Clone(stage0Effects)
}

// OnchainEffects returns the closed target-neutral on-chain authority vocabulary in lexical order.
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

// OnchainEffectsAreReadOnly reports whether an effect set is eligible for conservative view
// execution. Contract-target validation separately rejects host-only authority such as filesystem
// or process access; this predicate owns only consensus-visible mutation classification.
func OnchainEffectsAreReadOnly(effects []Effect) bool {
	return len(MutatingOnchainEffects(effects)) == 0
}

// ParseEffect returns the canonical effect for one exact Stage 0 spelling.
func ParseEffect(value string) (Effect, bool) {
	effect := Effect(value)
	return effect, slices.Contains(stage0Effects, effect)
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
