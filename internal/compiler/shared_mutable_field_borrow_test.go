package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"testing"
)

func TestSharedMutableFieldBorrowRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/shared_mutable_field_borrow.fy")
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v %+v", err, result.Semantic)
	}
	encoded, err := mircodec.Encode(result.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for name, program := range map[string]mir.Program{"fresh": result.MIR, "decoded": decoded} {
		t.Run(name, func(t *testing.T) {
			outcome, err := interpreter.Execute(program, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
				t.Fatalf("execute: %+v %v", outcome, err)
			}
		})
	}
	shortened := 0
	for _, fn := range decoded.Functions {
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				if op.Kind == mir.OperationSharedReborrow {
					shortened++
				}
				if op.Kind == mir.OperationFieldBorrow {
					_, ownerMut, _ := hir.BorrowElement(op.OwnerType)
					_, resultMut, _ := hir.BorrowElement(op.Type)
					if ownerMut != resultMut {
						t.Fatal("field operation bypassed shared root")
					}
				}
			}
		}
	}
	if shortened < 6 {
		t.Fatalf("lost explicit shared roots: %d", shortened)
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"exclusive result": func(op *mir.Operation) {
			element, _, _ := hir.BorrowElement(op.Type)
			op.Type = hir.MutableBorrow(element)
		},
		"exclusive owner": func(op *mir.Operation) {
			element, _, _ := hir.BorrowElement(op.OwnerType)
			op.OwnerType = hir.MutableBorrow(element)
		},
		"missing field":  func(op *mir.Operation) { op.Field = "missing" },
		"wrong referent": func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeUSize) },
	} {
		t.Run("mutation/"+name, func(t *testing.T) {
			program, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationFieldBorrow && !found {
							mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(program)) == 0 {
				t.Fatal("forged shared projection accepted")
			}
		})
	}

}
func TestSharedMutableFieldBorrowConflicts(t *testing.T) {
	const prelude = `module main;struct Span{i64 start;i64 end;}struct Token{Span position;i64 sibling;}
 fn Token make(){return Token{.position=Span{.start=3,.end=4},.sibling=7};}
 fn ref Span project(ref mut Token t){return ref t.position;}
 fn Option<ref Span> optional(ref mut Token t){return .some(project(t));}
 fn Result<ref Span,u32> result(ref mut Token t){return .ok(project(t));}
 fn void edit(ref mut Token t){t.sibling=8;}
 fn void use(ref Span s){assert(s.start==3);}
 `
	for name, body := range map[string]string{
		"direct-write":           `fn void bad(ref mut Token t){let ref Span s=ref t.position;t.position.start=8;use(s);}`,
		"sibling-write":          `fn void bad(ref mut Token t){let ref Span s=ref t.position;t.sibling=8;use(s);}`,
		"parent-call":            `fn void bad(ref mut Token t){let ref Span s=ref t.position;edit(t);use(s);}`,
		"parent-alias":           `fn void bad(ref mut Token t){let ref mut Token alias=t;let ref Span s=ref t.position;alias.sibling=8;use(s);}`,
		"shared-child-alias":     `fn void bad(ref mut Token t){let ref Span s=ref t.position;let ref Span alias=s;t.sibling=8;use(alias);}`,
		"mutable-sibling":        `fn void bad(ref mut Token t){let ref Span s=ref t.position;let ref mut i64 n=ref mut t.sibling;*n=8;use(s);}`,
		"returned":               `fn void bad(ref mut Token t){let ref Span s=project(t);t.sibling=8;use(s);}`,
		"rebound-old-owner":      `fn void bad(ref mut Token first,ref mut Token second){var ref mut Token t=first;let ref Span s=project(t);t=second;first.sibling=8;use(s);}`,
		"returned-alias":         `fn void bad(ref mut Token t){let ref mut Token a=t;let ref Span s=project(t);a.sibling=8;use(s);}`,
		"tagged-option":          `fn void bad(ref mut Token t){let Option<ref Span> s=optional(t);t.sibling=8;switch(s){case .some(v):{use(v);}case .none:{}};}`,
		"tagged-result":          `fn void bad(ref mut Token t){let Result<ref Span,u32> s=result(t);t.sibling=8;switch(s){case .ok(v):{use(v);}case .err(_):{}};}`,
		"owner-write":            `fn i32 main(){var Token t=make();let ref mut Token p=ref mut t;let ref Span s=ref p.position;t.sibling=8;use(s);return 0;}`,
		"owner-replace":          `fn i32 main(){var Token t=make();let ref mut Token p=ref mut t;let ref Span s=project(p);t=make();use(s);return 0;}`,
		"later-argument-write":   `fn void both(ref Span s,i64 n){}fn void bad(ref mut Token t){both(ref t.position,{t.sibling=8;0});}`,
		"shared-then-mutable":    `fn void both(ref Span s,ref mut Token t){}fn void bad(ref mut Token t){both(ref t.position,t);}`,
		"mutable-then-shared":    `fn void both(ref mut Token t,ref Span s){}fn void bad(ref mut Token t){both(t,ref t.position);}`,
		"parent-while-exclusive": `fn void bad(ref mut Token t){let ref mut Span s=ref mut t.position;let ref Span v=ref t.position;s.end=9;use(v);}`,
		"shared-store":           `fn void bad(ref mut Token t){let ref Span s=ref t.position;s.start=8;}`,
		"local-escape":           `fn ref Span bad(){var Token t=make();let ref mut Token p=ref mut t;return project(p);}`,
		"two-origins":            `fn ref Span bad(bool flag,ref mut Token a,ref mut Token b){return if(flag){project(a)}else{project(b)};}`,
		"resource":               `resource Owner{Span position;cleanup{}}fn ref Span bad(ref mut Owner t){return ref t.position;}`,
		"nonplain":               `struct Owner{Span position;u8[2] bytes;}fn ref Span bad(ref mut Owner t){return ref t.position;}`,
		"temporary":              `fn ref mut Token identity(ref mut Token t){return t;}fn ref Span bad(ref mut Token t){return ref identity(t).position;}`,
		"implicit-whole":         `fn ref Token bad(ref mut Token t){let ref Token shared=t;return shared;}`,
		"index":                  `fn void bad(ref mut Token[] t){let ref Span s=ref t[0].position;use(s);}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", prelude+body)})
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified || len(checked.Semantic.Diagnostics)+len(checked.Semantic.Unsupported) == 0 {
				t.Fatalf("accepted %s", name)
			}
			code := "E2101"
			switch name {
			case "local-escape":
				code = "E2102"
			case "two-origins":
				code = "E2103"
			case "shared-store":
				code = "E1008"
			case "resource", "nonplain", "temporary", "implicit-whole", "index":
				code = ""
			}
			if code != "" {
				found := false
				for _, d := range checked.Semantic.Diagnostics {
					if d.Code == code {
						found = true
						if d.Primary.EndByte <= d.Primary.StartByte {
							t.Fatal("missing span")
						}
						if code == "E2101" && len(d.Related) == 0 {
							t.Fatal("missing related loan")
						}
					}
				}
				if !found {
					t.Fatalf("missing %s: %+v", code, checked.Semantic)
				}
			}
		})
	}
}
