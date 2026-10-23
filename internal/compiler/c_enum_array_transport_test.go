package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCEnumArrayTransport(t *testing.T) {
	for _, base := range hir.CScalarTypes() {
		if hir.CEnumVariantCapacity(strings.ReplaceAll(base.String(), "::", ".")) == 0 {
			continue
		}
		t.Run(base.String(), func(t *testing.T) {
			// The enum follows its users to exercise declaration-order independence.
			source := cScalarArraySource("Status", 3) + "\n@repr(" + base.String() + ") enum Status{Ready;Busy;Done;}"
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked)
			}
			encoded, err := mircodec.Encode(checked.MIR, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			program, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			if len(program.EnumTypes) != 1 || program.EnumTypes[0].CBase != strings.ReplaceAll(base.String(), "::", ".") {
				t.Fatal("C enum base lost in array MIR round trip")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				unit, err := cbackend.Emit(program, "main::main", profile)
				if base == hir.TypeCSSizeT && profile.Target == cbackend.Stage0WindowsMSVCProfile().Target {
					if err == nil {
						t.Fatal("POSIX enum array emitted for Windows")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(unit.Source, "fy_enum_0 values[UINT64_C(3)];") {
					t.Fatal("missing nominal enum C array member")
				}
			}
			for _, invalidBase := range []string{"", "c.float", "unknown"} {
				program.EnumTypes[0].CBase = invalidBase
				if len(mir.Verify(program)) == 0 {
					t.Fatalf("invalid array element base accepted: %q", invalidBase)
				}
			}
		})
	}
}

func TestCEnumArrayBoundaries(t *testing.T) {
	declarations := "module main; @repr(c::uint) enum Status{Ready;Busy;} @repr(c::uint) enum Other{Ready;Busy;}"
	for name, body := range map[string]string{
		"private":       `struct P{Status[3] values;} fn i32 main(){return 0;}`,
		"plain":         `enum Plain{Ready;Busy;} @repr(C) struct P{Plain[3] values;} fn i32 main(){return 0;}`,
		"oversize":      `@repr(C) struct P{Status[4097] values;} fn i32 main(){return 0;}`,
		"zero":          `@repr(C) struct P{Status[0] values;} fn i32 main(){return 0;}`,
		"nested":        `@repr(C) struct P{Status[3][2] values;} fn i32 main(){return 0;}`,
		"identity":      `fn Other[3] bad(Status[3] v){return v;} fn i32 main(){return 0;}`,
		"base-identity": `fn c::uint[3] bad(Status[3] v){return v;} fn i32 main(){return 0;}`,
		"index":         `fn Status bad(Status[3] v){return v[0];} fn i32 main(){return 0;}`,
		"length":        `fn usize bad(Status[3] v){return v.len;} fn i32 main(){return 0;}`,
		"literal":       `fn Status[3] bad(Status v){return [v;3];} fn i32 main(){return 0;}`,
		"list":          `fn Status[3] bad(Status v){return [v,v,v];} fn i32 main(){return 0;}`,
		"view":          `fn void bad(Status[3] v){let ref Status[] s=ref v;} fn i32 main(){return 0;}`,
		"field-ref":     `@repr(C) struct P{Status[3] values;} fn void bad(ref P p){let ref Status[3] v=ref p.values;} fn i32 main(){return 0;}`,
		"element-store": `@repr(C) struct P{Status[3] values;} fn void bad(ref mut P p,Status v){p.values[0]=v;} fn i32 main(){return 0;}`,
		"nested-store":  `@repr(C) struct P{Status[3] values;} @repr(C) struct E{P p;} fn void bad(ref mut E e,P q){e.p.values=q.values;} fn i32 main(){return 0;}`,
		"foreign":       `unsafe extern C fn Status[3] produce(); fn i32 main(){let Status[3] v=unsafe{produce()};return 0;}`,
		"option":        `fn Option<Status[3]> bad(Status[3] v){return Some(v);} fn i32 main(){return 0;}`,
		"resource":      `resource R{Status[3] values;cleanup{}} fn i32 main(){return 0;}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", declarations+body)})
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err == nil {
					t.Fatal("unadmitted C enum array operation emitted")
				}
			}
		})
	}
}

func TestCEnumArrayMIRRejectsElementOperations(t *testing.T) {
	source := `module main; @repr(c::uint) enum Status{Ready;Busy;}
fn Status identity(Status value){return value;}
fn usize length(i32[3] values){return values.len;}`
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !checked.Verified || len(checked.MIR.EnumTypes) != 1 {
		t.Fatalf("baseline: %v %+v", err, checked)
	}
	program := checked.MIR
	array := hir.FixedArray(program.EnumTypes[0].Type, 3)
	mutated := false
	for i := range program.Functions {
		function := &program.Functions[i]
		if function.Name != "main::length" {
			continue
		}
		function.ParameterTypes[0] = array
		for j := range function.Blocks {
			for k := range function.Blocks[j].Operations {
				op := &function.Blocks[j].Operations[k]
				if op.Kind == mir.OperationArrayLength {
					op.OwnerType = array
					mutated = true
				}
			}
		}
	}
	failures := mir.Verify(program)
	if !mutated || len(failures) != 1 || failures[0].Code != mir.CodeOperationType || failures[0].Detail != "C enum arrays support whole-value transport only" {
		t.Fatalf("forged enum array operation: mutated=%v failures=%+v", mutated, failures)
	}
}

func TestCEnumArrayLengthBounds(t *testing.T) {
	for _, length := range []int{1, 4096} {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			source := cScalarArraySource("Status", length) + "\n@repr(c::long) enum Status{Ready;Busy;}"
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked)
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				if _, err := cbackend.Emit(checked.MIR, "main::main", profile); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
