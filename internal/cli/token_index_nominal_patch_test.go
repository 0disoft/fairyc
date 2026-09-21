package cli

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/nominalhir"
)

func TestTokenIndexNominalScannerPatch(t *testing.T) {
	checked := checkedTokenScanner(t, false)
	before, _ := json.Marshal(checked)
	patch := nominalTokenScannerPatch(t, checked)
	if len(patch.Functions) != 2 {
		t.Fatalf("patched %d functions", len(patch.Functions))
	}
	program, err := nominalhir.LowerBoundaryPatch(checked, patch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(program, checked.MIR) {
		t.Fatal("patch changed complete MIR")
	}
	legacy, err := nominalhir.LowerBoundary(checked, nominalTokenScanner(t, checked))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(program, legacy) {
		t.Fatal("patch differs from full closure oracle")
	}
	after, _ := json.Marshal(checked)
	if string(before) != string(after) {
		t.Fatal("baseline mutated")
	}
	for _, kind := range []string{"missing-caller", "missing-callee", "raw", "wrong-domain", "erasure-drift", "no-resolutions", "uncovered-source-use"} {
		t.Run(kind, func(t *testing.T) {
			p := nominalTokenScannerPatch(t, checked)
			base := checked
			switch kind {
			case "missing-caller", "missing-callee":
				name := "inspect"
				if kind == "missing-callee" {
					name = "token_info"
				}
				p.Functions = slices.DeleteFunc(p.Functions, func(f hir.Function) bool { return f.Name == name })
			case "no-resolutions":
				base.Semantic.Resolutions = nil
			case "uncovered-source-use":
				base.Semantic.Resolutions = slices.Clone(base.Semantic.Resolutions)
				changed := false
				for i, r := range base.Semantic.Resolutions {
					if r.Target.Module == "scanner" && r.Target.Name == "token_info" {
						base.Semantic.Resolutions[i].Use.Source = "uncovered.fy"
						changed = true
						break
					}
				}
				if !changed {
					t.Fatal("no source resolution to mutate")
				}
			default:
				changes := 0
				for i := range p.Functions {
					if p.Functions[i].Name != "inspect" {
						continue
					}
					p.Functions[i].Body = mapNominalHIR(reflect.ValueOf(p.Functions[i].Body), func(v reflect.Value) reflect.Value {
						e, ok := v.Interface().(hir.Expression)
						if !ok {
							return v
						}
						if kind == "erasure-drift" && e.Kind == "literal" && e.Type == hir.TypeI32 {
							e.Value = "19"
							changes++
						}
						if e.Kind == "call" && len(e.Children) == 3 && e.Children[2].Type == p.Declarations[0].Type() {
							if kind == "raw" {
								e.Children[2] = hir.Expression{Kind: "literal", Type: hir.TypeUSize, Value: "2"}
								changes++
							}
							if kind == "wrong-domain" {
								e.Children[2].Type = p.Declarations[1].Type()
								changes++
							}
						}
						return reflect.ValueOf(e)
					}).Interface().(hir.Block)
				}
				if changes == 0 {
					t.Fatal("mutation missed its target")
				}
			}
			got, err := nominalhir.LowerBoundaryPatch(base, p)
			if err == nil || !reflect.DeepEqual(got, mir.Program{}) {
				t.Fatalf("invalid patch published MIR: %v", err)
			}
			if kind == "missing-caller" && !strings.Contains(err.Error(), "caller") {
				t.Fatalf("wrong missing caller rejection: %v", err)
			}
		})
	}
}
