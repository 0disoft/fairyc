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

func TestTokenIndexNominalScannerReuseIsolation(t *testing.T) {
	checked := checkedTokenScanner(t, false)
	patch := nominalTokenScannerPatch(t, checked)
	want, _ := json.Marshal(checked.MIR)
	program, err := nominalhir.LowerBoundaryPatch(checked, patch)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := nominalhir.MeasureBoundaryPatch(checked, patch, false)
	if err != nil {
		t.Fatal(err)
	}
	measured, _, err := nominalhir.MeasureBoundaryPatch(checked, patch, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(program, control) || !reflect.DeepEqual(program, measured) {
		t.Fatal("reuse and relowering disagree")
	}
	// Mutate every scalar leaf, including nested operands, branch tables and
	// cleanup facts, so any shared mutable backing slice changes the baseline.
	if changed := mutateMIRLeaves(reflect.ValueOf(&program).Elem()); changed < 100 {
		t.Fatalf("insufficient mutation coverage: %d", changed)
	}
	got, _ := json.Marshal(checked.MIR)
	if string(got) != string(want) {
		t.Fatal("result mutation leaked into baseline")
	}
	second, err := nominalhir.LowerBoundaryPatch(checked, patch)
	if err != nil {
		t.Fatal(err)
	}
	got, _ = json.Marshal(second)
	if string(got) != string(want) {
		t.Fatal("one caller corrupted another result")
	}
	mutateMIRLeaves(reflect.ValueOf(&checked.MIR).Elem())
	got, _ = json.Marshal(second)
	if string(got) != string(want) {
		t.Fatal("baseline mutation leaked into result")
	}
}

func mutateMIRLeaves(v reflect.Value) int {
	count := 0
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			count += mutateMIRLeaves(v.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			count += mutateMIRLeaves(v.Index(i))
		}
	case reflect.String:
		v.SetString(v.String() + "!")
		count++
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(v.Int() + 1)
		count++
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(v.Uint() + 1)
		count++
	case reflect.Bool:
		v.SetBool(!v.Bool())
		count++
	}
	return count
}
