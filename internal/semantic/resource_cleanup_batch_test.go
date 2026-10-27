package semantic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/source"
)

func cleanupBatchFixture(count int) (*analyzer, resourceEnvironment, *symbol) {
	a, root, leaf := resourceOwnerGraphFixture(32)
	a.resourceCaller = &symbol{effectsValid: true}
	leaf.effects, leaf.effectsValid = []hir.Effect{hir.EffectConsole}, true
	a.resourceTypes[leaf.typ].abilities.abilities = resourceAbilityDrop
	environment := make(resourceEnvironment, count)
	for i := range count {
		value := &symbol{typ: root, source: "main.fy", declaration: diagnostic.Span{StartByte: i, EndByte: i + 1}}
		environment[value] = resourceValueState{state: ResourceLive}
	}
	return a, environment, leaf
}

func TestCleanupBatchRechecksAtEachSiteAndAuthority(t *testing.T) {
	a, environment, leaf := cleanupBatchFixture(8)
	for i, test := range []struct {
		authorized bool
		abilities  resourceAbility
		code       string
	}{
		{false, resourceAbilityDrop, "E7102"},
		{true, resourceAbilityDrop, ""},
		{false, resourceAbilityDrop, "E7102"},
		{true, 0, "E1108"},
		{true, resourceAbilityDrop, ""},
	} {
		a.diagnostics = nil
		a.resourceCaller.effects = nil
		if test.authorized {
			a.resourceCaller.effects = []hir.Effect{hir.EffectConsole}
		}
		a.resourceTypes[leaf.typ].abilities.abilities = test.abilities
		at := hir.Span{Source: "main.fy", StartByte: 100 + i, EndByte: 101 + i}
		a.checkResourceCleanupEffects(environment, at, nil, -1)
		if test.code == "" {
			if len(a.diagnostics) != 0 {
				t.Fatalf("authorized cleanup rejected: %+v", a.diagnostics)
			}
		} else if len(a.diagnostics) != 1 || a.diagnostics[0].Code != test.code || a.diagnostics[0].Primary != at {
			t.Fatalf("incorrect batch diagnostic: %+v", a.diagnostics)
		}
	}
}

func TestCleanupBatchKeepsDifferentRootRequirements(t *testing.T) {
	a, environment, _ := cleanupBatchFixture(8)
	other := hir.ParseType("main::OtherCleanup")
	owner := &symbol{typ: other, effects: []hir.Effect{hir.EffectFilesystem}, effectsValid: true}
	a.resourceTypes[other] = &resourceInfo{symbol: owner, abilities: resourceAbilityDeclaration{valid: true}}
	environment[&symbol{typ: other, source: "main.fy"}] = resourceValueState{state: ResourceLive}
	a.checkResourceCleanupEffects(environment, hir.Span{}, nil, -1)
	if len(a.diagnostics) != 2 || a.diagnostics[0].Code != "E7102" || a.diagnostics[1].Code != "E7102" {
		t.Fatalf("distinct cleanup requirements lost: %+v", a.diagnostics)
	}
}

func TestCleanupBatchFiltersBeforeCheckingPendingValues(t *testing.T) {
	a, environment, _ := cleanupBatchFixture(1)
	var value *symbol
	for value = range environment {
	}
	a.resourceCleanupExempt = value
	a.resourcePending = []resourcePendingValue{{typ: value.typ, scopeDepth: 0}, {typ: value.typ, scopeDepth: 2}}
	a.checkResourceCleanupEffects(environment, hir.Span{}, nil, 3)
	if len(a.diagnostics) != 0 {
		t.Fatal("excluded locals or pending scope required cleanup authority")
	}
	a.checkResourceCleanupEffects(environment, hir.Span{}, nil, 2)
	if len(a.diagnostics) != 1 || a.diagnostics[0].Code != "E7102" {
		t.Fatalf("pending value hidden by an exempt local of the same type: %+v", a.diagnostics)
	}
}

func BenchmarkRepeatedCleanupBatch(b *testing.B) {
	for _, count := range []int{1, 256} {
		b.Run(fmt.Sprintf("locals=%d", count), func(b *testing.B) {
			a, environment, _ := cleanupBatchFixture(count)
			a.resourceCaller.effects = []hir.Effect{hir.EffectConsole}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				a.checkResourceCleanupEffects(environment, hir.Span{}, nil, -1)
				if len(a.diagnostics) != 0 {
					b.Fatal(a.diagnostics)
				}
			}
		})
	}
}

func BenchmarkAnalyzeRepeatedCleanup(b *testing.B) {
	var text strings.Builder
	text.WriteString("module main; resource R0{i32 n;cleanup{}}")
	for i := 1; i <= 32; i++ {
		fmt.Fprintf(&text, "resource R%d{R%d inner;cleanup{}}", i, i-1)
	}
	text.WriteString("fn void f(")
	for i := range 256 {
		if i != 0 {
			text.WriteByte(',')
		}
		fmt.Fprintf(&text, "R32 p%d", i)
	}
	text.WriteString("){}")
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		b.Fatalf("parse cleanup source: %v", err)
	}
	files := []File{{Path: "main.fy", Module: parsed.AST}}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result := Analyze(files)
		if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
			b.Fatalf("cleanup source rejected: %+v %+v", result.Diagnostics, result.Unsupported)
		}
	}
}
