package hir

import "slices"

// Effect identifies one source-visible external capability used by a function.
type Effect string

const (
	EffectClock       Effect = "clock"
	EffectConsole     Effect = "console"
	EffectEnvironment Effect = "environment"
	EffectFilesystem  Effect = "filesystem"
	EffectNetwork     Effect = "network"
	EffectProcess     Effect = "process"
	EffectRandom      Effect = "random"
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

// Stage0Effects returns the closed canonical Stage 0 effect vocabulary in lexical order.
func Stage0Effects() []Effect {
	return slices.Clone(stage0Effects)
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
