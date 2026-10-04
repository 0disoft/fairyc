package interpreter

import (
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestMutableStructViewCopiesAndBounds(t *testing.T) {
	child := copyTestStruct(t)
	token, err := structValue(&mir.StructType{Type: "main::Token", Fields: []mir.StructField{{Name: "nested", Type: child.typ}}}, []runtimeValue{child})
	if err != nil {
		t.Fatal(err)
	}
	typ := hir.MutableBorrow(hir.Type("[main::Token]"))
	backing := &arrayState{elements: []runtimeValue{token.cloneForValueCopy(), token.cloneForValueCopy()}}
	view := runtimeValue{kind: valueView, typ: typ, view: &indexedViewState{array: backing, length: 2, mutable: true}}
	if err := view.storeIndexed(0, token); err != nil {
		t.Fatal(err)
	}
	token.structure.fields["nested"].structure.fields["count"] = i32Value(42)
	loaded, err := view.indexedAt(0)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.structure.fields["nested"].structure.fields["count"].i32 != 7 {
		t.Fatal("stored value aliases source")
	}
	loaded.structure.fields["nested"].structure.fields["count"] = i32Value(99)
	if backing.elements[0].structure.fields["nested"].structure.fields["count"].i32 != 7 {
		t.Fatal("copied load aliases owner")
	}
	sub, err := view.sliceIndexedView(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := sub.storeIndexed(0, token); err != nil {
		t.Fatal(err)
	}
	if backing.elements[1].structure.fields["nested"].structure.fields["count"].i32 != 42 {
		t.Fatal("subview store missed backing")
	}
	for _, index := range []uint64{2, ^uint64(0)} {
		if err := view.storeIndexed(index, token); err == nil {
			t.Fatal("out of bounds store accepted")
		}
	}
	wrong := token
	wrong.typ = "main::Other"
	if err := view.storeIndexed(0, wrong); err == nil {
		t.Fatal("wrong nominal accepted")
	}
	shared := view
	shared.typ = hir.SharedBorrow(hir.Type("[main::Token]"))
	if err := shared.storeIndexed(0, token); err == nil {
		t.Fatal("shared store accepted")
	}
}

func TestIndexedStructFieldReference(t *testing.T) {
	token := copyTestStruct(t)
	typ := hir.MutableBorrow(hir.Type("[" + string(token.typ) + "]"))
	backing := &arrayState{elements: []runtimeValue{token.cloneForValueCopy()}}
	view := runtimeValue{kind: valueView, typ: typ, view: &indexedViewState{array: backing, length: 1, mutable: true}}
	borrowed, err := view.borrowStructElement(0, hir.MutableBorrow(token.typ))
	if err != nil {
		t.Fatal(err)
	}
	borrowed.structure.fields["count"] = i32Value(42)
	copied, err := view.indexedAt(0)
	if err != nil || copied.structure.fields["count"].i32 != 42 {
		t.Fatalf("copy: %+v %v", copied, err)
	}
	copied.structure.fields["count"] = i32Value(7)
	if backing.elements[0].structure.fields["count"].i32 != 42 {
		t.Fatal("copy changed borrowed storage")
	}
	for _, index := range []uint64{1, ^uint64(0)} {
		if _, err := view.borrowStructElement(index, hir.SharedBorrow(token.typ)); err == nil {
			t.Fatal("bounds accepted")
		}
	}
	shared := view
	shared.typ = hir.SharedBorrow(hir.Type("[" + string(token.typ) + "]"))
	if _, err := shared.borrowStructElement(0, hir.MutableBorrow(token.typ)); err == nil {
		t.Fatal("shared owner became exclusive")
	}
}

// The private array-backed runtime boundary supplies the descriptor; source
// fixed-array struct conversion and interpretation of host Vec storage stay closed.
func TestMutableStructForInterpreter(t *testing.T) {
	text := `module main;struct Token{i32 value;}fn void edit(ref mut Token[] items){for(let ref mut Token item in items){if(item.value==1){item.value=11;continue;}item.value+=40;if(item.value==43){break;}}}`
	parsed, e := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if e != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("parse: %v", e)
	}
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("analyze: %+v %+v", result.Diagnostics, result.Unsupported)
	}
	program, e := mir.Lower(result.HIR)
	if e != nil {
		t.Fatal(e)
	}
	if errs := mir.Verify(program); len(errs) != 0 {
		t.Fatal(errs)
	}
	r := runtime{byName: map[string]*mir.Function{}, bySymbol: map[hir.SymbolID]*mir.Function{}, structTypes: map[hir.Type]*mir.StructType{}, steps: maxSteps}
	for i := range program.Functions {
		f := &program.Functions[i]
		r.byName[f.Name] = f
		r.bySymbol[f.Symbol] = f
	}
	for i := range program.StructTypes {
		d := &program.StructTypes[i]
		r.structTypes[d.Type] = d
	}
	definition := r.structTypes["main::Token"]
	for _, values := range [][]int32{{}, {1, 2, 3, 4}} {
		backing := &arrayState{}
		for _, v := range values {
			token, e := structValue(definition, []runtimeValue{i32Value(v)})
			if e != nil {
				t.Fatal(e)
			}
			backing.elements = append(backing.elements, token)
		}
		typ := hir.MutableBorrow(hir.Type("[main::Token]"))
		view := runtimeValue{kind: valueView, typ: typ, view: &indexedViewState{array: backing, length: uint64(len(values)), mutable: true}}
		completion, e := r.call(r.byName["main::edit"], []runtimeValue{view}, 0)
		if e != nil || completion.trapCategory != "" {
			t.Fatalf("run: %+v %v", completion, e)
		}
		for i, want := range []int32{11, 42, 43, 4} {
			if len(values) == 0 {
				break
			}
			if got := backing.elements[i].structure.fields["value"].i32; got != want {
				t.Fatalf("element %d=%d want %d", i, got, want)
			}
		}
	}
}
