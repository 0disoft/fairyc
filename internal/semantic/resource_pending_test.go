package semantic_test

import (
	"github.com/0disoft/fairyc/internal/semantic"
	"strings"
	"testing"
)

const pendingPrefix = `module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}`

func TestResourcePendingCleanupEffects(t *testing.T) {
	for _, body := range []string{
		`fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n=number()?});}`,
		`fn Box take(R value,i32 n){return Box{.first=value,.n=n};}fn Result<Box,i32> f(){return .ok(take(make(),number()?));}`,
		`fn Result<Box,i32> f(){let R value=make();return .ok(Box{.first=value,.n=number()?});}`,
		`fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n={return .err(1);0}});}`,
		`fn Result<Box,i32> f(){let Box box=Box{.first=make(),.n={return .err(1);0}};return .ok(box);}`,
		`fn Result<Box,i32> f(bool c){while(c){let Box box=Box{.first=make(),.n={break;0}};}return .err(1);}`,
		`fn Result<Box,i32> f(bool c){while(c){let Box box=Box{.first=make(),.n={continue;0}};}return .err(1);}`,
	} {
		text := pendingPrefix + body
		result := analyzeText(t, map[string]string{"main.fy": text})
		if result.FailedPhase != semantic.PhaseFlow || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E7102" {
			t.Fatalf("body=%s phase=%s diagnostics=%+v unsupported=%+v", body, result.FailedPhase, result.Diagnostics, result.Unsupported)
		}
		d := result.Diagnostics[0]
		if len(d.Related) != 1 || d.Related[0].Role != "effect-declaration" {
			t.Fatalf("related=%+v", d.Related)
		}
		if at := strings.Index(text, "number()?"); at >= 0 {
			assertSpan(t, d.Primary, "main.fy", at, at+len("number()?"))
		}
	}
}

func TestResourcePendingTransferDoesNotChargeCompletedCalls(t *testing.T) {
	for _, body := range []string{
		`@effects(console)fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n=number()?});}`,
		`fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n=1});}`,
		`fn Result<Box,i32> f(){return .ok(Box{.n=number()?,.first=make()});}`,
		`fn R pass(R value){return value;}fn Result<Box,i32> f(){return .ok(Box{.n=number()?,.first=pass(make())});}`,
		`fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n={while(true){break;}0}});}`,
	} {
		assertCleanTypeResult(t, analyzeText(t, map[string]string{"main.fy": pendingPrefix + body}))
	}
}
