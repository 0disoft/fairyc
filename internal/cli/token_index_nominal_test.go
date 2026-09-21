package cli

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/nominalhir"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
)

func checkedTokenScanner(t *testing.T, wrapped bool) compiler.Result {
	t.Helper()
	root, _ := tokenIndexWorkspace(t, wrapped)
	workspace, _, err := loadWorkspaceWithLock(root, false)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("scanner check: %v %+v", err, checked.Semantic.Diagnostics)
	}
	return checked
}

// The annotation plan is test-owned and exact. It models explicit construction
// and extraction; it does not pretend the stable source parser supports newtypes.
func nominalTokenScanner(t *testing.T, checked compiler.Result) nominalhir.Program {
	t.Helper()
	data, err := json.Marshal(checked.Semantic.HIR.Modules)
	if err != nil {
		t.Fatal(err)
	}
	var modules []hir.Module
	if err = json.Unmarshal(data, &modules); err != nil {
		t.Fatal(err)
	}
	return annotateTokenScanner(t, modules)
}

// Copy only the two edited function headers. The shared annotation helper clones
// their bodies before changing expressions; all other modules remain untouched.
func nominalTokenScannerPatch(t *testing.T, checked compiler.Result) nominalhir.BoundaryPatch {
	t.Helper()
	var modules []hir.Module
	for _, m := range checked.Semantic.HIR.Modules {
		copy := hir.Module{Package: m.Package, Path: m.Path, SourcePath: m.SourcePath}
		for _, f := range m.Functions {
			if !(m.Path == "scanner" && f.Name == "token_info" || m.Path == "main" && f.Name == "inspect") {
				continue
			}
			f.Parameters = slices.Clone(f.Parameters)
			f.Locals = slices.Clone(f.Locals)
			copy.Functions = append(copy.Functions, f)
		}
		if len(copy.Functions) > 0 {
			modules = append(modules, copy)
		}
	}
	annotated := annotateTokenScanner(t, modules)
	patch := nominalhir.BoundaryPatch{Declarations: annotated.Declarations}
	for _, m := range annotated.Modules {
		patch.Functions = append(patch.Functions, m.Functions...)
	}
	return patch
}

func annotateTokenScanner(t *testing.T, modules []hir.Module) nominalhir.Program {
	t.Helper()
	var callee hir.SymbolID
	var index, offset nominalhir.Declaration
	found := false
	for _, m := range modules {
		if m.Path == "scanner" {
			for _, f := range m.Functions {
				if f.Name == "token_info" {
					if len(f.Parameters) != 2 || f.Parameters[1].Name != "index" || f.Parameters[1].Type != hir.TypeUSize {
						t.Fatal("scanner boundary drift")
					}
					callee = f.Symbol
					index = nominalhir.Declaration{Package: m.Package, Module: m.Path, Name: "TokenIndex", Underlying: hir.TypeUSize}
					offset = index
					offset.Name = "ByteOffset"
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("missing scanner boundary")
	}
	bindingsFound := 0
	for mi := range modules {
		for fi := range modules[mi].Functions {
			f := &modules[mi].Functions[fi]
			bindings := map[hir.SymbolID]hir.Type{}
			if f.Symbol == callee {
				f.Parameters[1].Type = index.Type()
				bindings[f.Parameters[1].Symbol] = index.Type()
			}
			if modules[mi].Path == "main" && f.Name == "inspect" {
				for i, l := range f.Locals {
					switch l.Name {
					case "token_index":
						f.Locals[i].Type = index.Type()
					case "byte_offset":
						f.Locals[i].Type = offset.Type()
					default:
						continue
					}
					bindings[l.Symbol] = f.Locals[i].Type
					bindingsFound++
				}
			}
			f.Body = mapNominalHIR(reflect.ValueOf(f.Body), func(v reflect.Value) reflect.Value {
				switch e := v.Interface().(type) {
				case hir.Expression:
					if e.Kind == "path" {
						if typ, ok := bindings[e.Symbol]; ok {
							e.Type = typ
							return reflect.ValueOf(hir.Expression{Kind: "nominal-unwrap", Type: hir.TypeUSize, Span: e.Span, Children: []hir.Expression{e}})
						}
					}
					if e.Kind == "call" && e.Symbol == callee {
						if len(e.Children) != 3 {
							t.Fatal("call boundary drift")
						}
						a := e.Children[2]
						if a.Kind == "nominal-unwrap" {
							e.Children[2] = a.Children[0]
						}
						return reflect.ValueOf(e)
					}
				case hir.Statement:
					if e.Kind == "binding" {
						if typ, ok := bindings[e.Binding]; ok {
							child := *e.Expression
							e.Expression = &hir.Expression{Kind: "nominal-wrap", Type: typ, Span: child.Span, Children: []hir.Expression{child}}
							return reflect.ValueOf(e)
						}
					}
				}
				return v
			}).Interface().(hir.Block)
		}
	}
	if bindingsFound != 2 {
		t.Fatal("caller binding boundary drift")
	}
	return nominalhir.Program{Declarations: []nominalhir.Declaration{index, offset}, Modules: modules}
}

func mapNominalHIR(v reflect.Value, visit func(reflect.Value) reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			out.Field(i).Set(mapNominalHIR(v.Field(i), visit))
		}
		return visit(out)
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(mapNominalHIR(v.Index(i), visit))
		}
		return out
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(mapNominalHIR(v.Elem(), visit))
		return out
	default:
		return v
	}
}

func TestTokenIndexNominalScannerBoundary(t *testing.T) {
	checked := checkedTokenScanner(t, false)
	input := nominalTokenScanner(t, checked)
	before, _ := json.Marshal(input)
	got, err := nominalhir.LowerBoundary(checked, input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, checked.MIR) {
		t.Fatal("scanner MIR changed")
	}
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("nominal input mutated")
	}
	entry := tokenScannerEntry(t, checked)
	base, err := cbackend.EmitStorageTransfers(checked.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := cbackend.EmitStorageTransfers(got, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	if base.Source != candidate.Source {
		t.Fatal("scanner generated C changed")
	}
	for _, wrong := range []string{"raw", "offset", "missing-extraction", "baseline-drift"} {
		t.Run(wrong, func(t *testing.T) {
			p := nominalTokenScanner(t, checked)
			changed := 0
			for mi := range p.Modules {
				for fi := range p.Modules[mi].Functions {
					f := &p.Modules[mi].Functions[fi]
					f.Body = mapNominalHIR(reflect.ValueOf(f.Body), func(v reflect.Value) reflect.Value {
						e, ok := v.Interface().(hir.Expression)
						if !ok {
							return v
						}
						if wrong == "missing-extraction" && e.Kind == "nominal-unwrap" {
							changed++
							return reflect.ValueOf(e.Children[0])
						}
						if wrong == "baseline-drift" && e.Kind == "literal" && e.Type == hir.TypeI32 {
							e.Value = "17"
							changed++
							return reflect.ValueOf(e)
						}
						if e.Kind == "call" && len(e.Children) == 3 && e.Children[2].Type == p.Declarations[0].Type() {
							if wrong == "raw" {
								e.Children[2] = hir.Expression{Kind: "literal", Type: hir.TypeUSize, Value: "2"}
								changed++
							}
							if wrong == "offset" {
								e.Children[2].Type = p.Declarations[1].Type()
								changed++
							}
						}
						return reflect.ValueOf(e)
					}).Interface().(hir.Block)
				}
			}
			if changed == 0 {
				t.Fatal("mutation missed boundary")
			}
			if _, err := nominalhir.LowerBoundary(checked, p); err == nil {
				t.Fatal("invalid overlay accepted")
			}
		})
	}
}

func tokenScannerEntry(t *testing.T, checked compiler.Result) string {
	t.Helper()
	for _, m := range checked.Semantic.HIR.Modules {
		if m.Path == "main" {
			for _, f := range m.Functions {
				if f.Name == "main" {
					return hir.QualifiedModulePath(m.Package, m.Path) + "::main"
				}
			}
		}
	}
	t.Fatal("missing scanner entry")
	return ""
}
