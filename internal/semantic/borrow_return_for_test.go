package semantic

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestBorrowReturnSummaryForDeclarationOrder(t *testing.T) {
	callee := "fn ref i32 identity(ref i32 p){return p;}"
	consumer := "fn ref i32 consume(ref i32 p,ref u8[] bytes){return relay(p,bytes);}"
	for name, loop := range map[string]string{
		"classic": "for(var i32 i=0;i<1;i=i+1){return identity(p);}",
		"view":    "for(let u8 b in bytes){return identity(p);}",
		"nested":  "for(var i32 i=0;i<1;i=i+1){for(let u8 b in bytes){return identity(p);}}",
	} {
		caller := "fn ref i32 relay(ref i32 p,ref u8[] bytes){" + loop + "return p;}"
		for _, forward := range []bool{true, false} {
			order := "callee-first"
			declarations := callee + caller + consumer
			if forward {
				order = "caller-first"
				declarations = caller + callee + consumer
			}
			t.Run(name+"/"+order, func(t *testing.T) {
				result := Analyze(borrowLiveFiles(t, "module main;"+declarations))
				if result.CompletedThrough != PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
					t.Fatalf("declaration order changed borrow proof: phase=%s diagnostics=%v unsupported=%v", result.CompletedThrough, result.Diagnostics, result.Unsupported)
				}
				escaping := 0
				for _, fact := range result.Borrows {
					if fact.Escapes && fact.Origin != "Param(0)" {
						t.Fatalf("wrong return origin: %+v", fact)
					}
					if fact.Escapes {
						escaping++
					}
				}
				if escaping != 4 {
					t.Fatalf("escaping returns=%d, want 4", escaping)
				}
			})
		}
	}
}

func TestFlowInventoryForVisitsEachRegionOnce(t *testing.T) {
	initializer, condition, returned := &ast.Expression{}, &ast.Expression{}, &ast.Expression{}
	target, post, tail := &ast.Expression{}, &ast.Expression{}, &ast.Expression{}
	loop := &ast.ForLoop{
		Initializer: &ast.Statement{Binding: &ast.Binding{Expression: initializer}},
		Condition:   condition,
		Body:        &ast.Block{Statements: []ast.Statement{{Kind: syntax.KindReturn, Expression: returned}}},
		Post:        &ast.Statement{Target: target, Expression: post},
	}
	classic := &ast.Block{Statements: []ast.Statement{{For: loop}}, Tail: tail}
	prepared := &ast.Block{Statements: []ast.Statement{{
		ByteFor:    &ast.ByteForLoop{Body: loop.Body},
		Expression: &ast.Expression{Kind: syntax.KindBlock, Block: classic},
	}}}
	for name, body := range map[string]*ast.Block{"classic": classic, "prepared-view": prepared} {
		t.Run(name, func(t *testing.T) {
			inventory := flowInventory{}
			inventory.collectBlock(body)
			want := []*ast.Expression{initializer, condition, returned, target, post, tail}
			if body == prepared {
				want = append([]*ast.Expression{prepared.Statements[0].Expression}, want...)
			}
			if !slices.Equal(inventory.expressions, want) || !slices.Equal(inventory.returns, []*ast.Expression{returned}) {
				t.Fatalf("loop traversal missing or duplicated nodes: expressions=%v returns=%v", inventory.expressions, inventory.returns)
			}
		})
	}
}

func TestBorrowReturnSummaryForUnprovedRecursion(t *testing.T) {
	text := "module main;fn ref i32 recurse(ref i32 p){for(var i32 i=0;i<1;i=i+1){return recurse(p);}return recurse(p);}"
	result := Analyze(borrowLiveFiles(t, text))
	if result.CompletedThrough == PhaseFlow || len(result.Unsupported) == 0 {
		t.Fatalf("unproved recursive origin accepted: %+v", result)
	}
	for _, unsupported := range result.Unsupported {
		if !strings.Contains(unsupported.Feature, "unknown origin") {
			t.Fatalf("unexpected refusal: %+v", unsupported)
		}
	}
}
