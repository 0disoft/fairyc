package semantic_test

import (
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

const nestedEffectPrefix = `module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}`

func TestResourceNestedCleanupEffectsAreRequired(t *testing.T) {
	for _, body := range []string{
		`fn void f(){let Box value=make();}`,
		`resource Outer{Box inner;cleanup{}}fn void f(){let Outer value=Outer{.inner=make()};}`,
		`resource Pair{Box a;Box b;cleanup{}}fn void f(){let Pair value=Pair{.a=make(),.b=make()};}`,
		`fn void f(){let Option<Box> value=.some(make());}`,
		`fn void f(){let Result<Box,i32> value=.ok(make());discard value.is_ok;}`,
		`fn Box f(){var Box value=make();value=make();return value;}`,
		`resource Holder{i32 n;cleanup{let Box value=make();}}`,
	} {
		text := nestedEffectPrefix + body
		result := analyzeText(t, map[string]string{"main.fy": text})
		if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7102" || len(result.Unsupported) != 0 {
			t.Fatalf("body=%q phase=%s diagnostics=%+v unsupported=%+v", body, result.FailedPhase, result.Diagnostics, result.Unsupported)
		}
		if len(result.Diagnostics[0].Related) != 1 || result.Diagnostics[0].Related[0].Role != "effect-declaration" {
			t.Fatalf("related=%+v", result.Diagnostics[0].Related)
		}
		start := strings.Index(text, "@effects(console)")
		assertSpan(t, result.Diagnostics[0].Related[0].Span, "main.fy", start, start+len("@effects(console)"))
	}
}

func TestResourceNestedCleanupEffectsAllowAuthorityOrTransfer(t *testing.T) {
	for _, body := range []string{
		`@effects(console)fn void f(){let Box value=make();}`,
		`fn Box f(Box value){return value;}`,
		`resource Outer{Box inner;cleanup{}}fn Outer f(Outer value){return value;}`,
		`@effects(console)resource Holder{i32 n;cleanup{let Box value=make();}}`,
	} {
		assertCleanTypeResult(t, analyzeText(t, map[string]string{"main.fy": nestedEffectPrefix + body}))
	}
}
