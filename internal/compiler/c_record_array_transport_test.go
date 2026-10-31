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

func TestCRecordArrayTransport(t *testing.T) {
	for name, fields := range map[string]string{
		"plain":              "i32 code; u8 byte;",
		"mixed":              "c::long code; f64 ratio; Leaf nested;",
		"array-fields":       "c::int[2] codes; f32[3] samples; Status[2] states;",
		"record-array-field": "Leaf[2] nested;",
	} {
		for _, length := range []int{1, 3, 4096} {
			t.Run(fmt.Sprintf("%s/%d", name, length), func(t *testing.T) {
				source := cScalarArraySource("Element", length) + "\n@repr(C) struct Element{" + fields + "} @repr(C) struct Leaf{c::double value;} @repr(c::uint) enum Status{Ready;Busy;}"
				checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
				if err != nil || !checked.Verified {
					t.Fatalf("check: %v semantic=%+v", err, checked.Semantic)
				}
				encoded, err := mircodec.Encode(checked.MIR, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				program, err := mircodec.Decode(encoded, 4<<20)
				if err != nil {
					t.Fatal(err)
				}
				if failures := mir.Verify(program); len(failures) != 0 {
					t.Fatal(failures)
				}
				for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
					unit, err := cbackend.Emit(program, "main::main", profile)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(unit.Source, fmt.Sprintf("struct Element values[UINT64_C(%d)];", length)) {
						t.Fatal("missing actual C record array")
					}
					if strings.Index(unit.Source, "struct Element {") > strings.Index(unit.Source, "struct Packet {") {
						t.Fatal("record array used before element is complete")
					}
				}
			})
		}
	}
}

func TestCRecordArrayBoundaries(t *testing.T) {
	declarations := `module main; @repr(C) struct Element{f64 number;} @repr(C) struct Packet{Element[3] values;}`
	for name, body := range map[string]string{
		"zero":            `@repr(C) struct Bad{Element[0] values;}`,
		"oversize":        `@repr(C) struct Bad{Element[4097] values;}`,
		"nested-array":    `@repr(C) struct Bad{Element[3][2] values;}`,
		"private-element": `struct Private{i32 v;} @repr(C) struct Bad{Private[2] values;}`,
		"empty-element":   `@repr(C) struct Empty{} @repr(C) struct Bad{Empty[2] values;}`,
		"cycle":           `@repr(C) struct Cycle{Cycle[2] values;}`,
		"indirect-cycle":  `@repr(C) struct A{B[2] values;} @repr(C) struct B{A value;}`,
		"private-array":   `struct Bad{Element[2] values;}`,
		"bool-field":      `@repr(C) struct Bool{bool value;} @repr(C) struct Bad{Bool[2] values;}`,
		"length":          `fn usize bad(Element[3] v){return v.len;}`,
		"index":           `fn Element bad(Element[3] v){return v[0];}`,
		"literal":         `fn Element[3] bad(Element v){return [v,v,v];}`,
		"repeat":          `fn Element[3] bad(Element v){return [v;3];}`,
		"view":            `fn void bad(Element[3] v){let ref Element[] s=ref v;}`,
		"field-ref":       `fn void bad(ref Packet p){let ref Element[3] v=ref p.values;}`,
		"store":           `fn void bad(ref mut Packet p,Element v){p.values[0]=v;}`,
		"nested-store":    `@repr(C) struct E{Packet p;} fn void bad(ref mut E e,Packet p){e.p.values=p.values;}`,
		"identity":        `@repr(C) struct Other{f64 number;} fn Other[3] bad(Element[3] v){return v;}`,
		"length-identity": `fn Element[2] bad(Element[3] v){return v;}`,
		"foreign":         `unsafe extern C fn Element[3] produce(); fn void bad(){let v=unsafe{produce()};}`,
		"option":          `fn Option<Element[3]> bad(Element[3] v){return Some(v);}`,
		"resource":        `resource R{Element[3] values;cleanup{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			entry := " fn i32 main(){return 0;}"
			if name == "foreign" {
				entry = " fn i32 main(){bad();return 0;}"
			}
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", declarations+body+entry)})
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
					t.Fatal("unadmitted record array emitted")
				}
			}
		})
	}
}

func TestCRecordArrayPreservesExistingPlainOperations(t *testing.T) {
	source := `module main; @repr(C) struct Element{i32 code;} fn i32 main(){var Element[2] v=[Element{.code=7};2];v[1]=Element{.code=9};return v[0].code+v[1].code;}`
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !checked.Verified {
		t.Fatalf("plain regression: %v %+v", err, checked)
	}
	if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
		t.Fatal(err)
	}
}

func TestCRecordArrayMIRRejectsElementOperations(t *testing.T) {
	source := `module main; @repr(C) struct Element{f64 number;} fn Element identity(Element v){return v;} fn usize length(i32[3] v){return v.len;}`
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !checked.Verified {
		t.Fatalf("baseline: %v %+v", err, checked)
	}
	array := hir.FixedArray(checked.MIR.StructTypes[0].Type, 3)
	for i := range checked.MIR.Functions {
		fn := &checked.MIR.Functions[i]
		if fn.Name != "main::length" {
			continue
		}
		fn.ParameterTypes[0] = array
		for j := range fn.Blocks {
			for k := range fn.Blocks[j].Operations {
				op := &fn.Blocks[j].Operations[k]
				if op.Kind == mir.OperationArrayLength {
					op.OwnerType = array
				}
			}
		}
	}
	failures := mir.Verify(checked.MIR)
	if len(failures) != 1 || !strings.Contains(failures[0].Detail, "whole-value transport only") {
		t.Fatalf("forged array operation: %+v", failures)
	}
}

func TestCRecordArrayTargetLayoutAndMIRMutations(t *testing.T) {
	source := cScalarArraySource("Element", 3) + " @repr(C) struct Element{c::long value;}"
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
	if err != nil || !checked.Verified {
		t.Fatalf("baseline: %v %+v", err, checked)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(checked.MIR, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		size, tail := 40, 32
		if profile.Target == cbackend.Stage0WindowsMSVCProfile().Target {
			size, tail = 20, 16
		}
		for _, fragment := range []string{fmt.Sprintf("sizeof(struct Packet) == UINT64_C(%d)", size), fmt.Sprintf("offsetof(struct Packet, tail) == UINT64_C(%d)", tail)} {
			if !strings.Contains(unit.Source, fragment) {
				t.Fatalf("missing target layout %s", fragment)
			}
		}
	}
	data, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"private", "empty", "cycle"} {
		t.Run(name, func(t *testing.T) {
			program, err := mircodec.Decode(data, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for i := range program.StructTypes {
				definition := &program.StructTypes[i]
				if definition.Name != "Element" {
					continue
				}
				found = true
				switch name {
				case "private":
					definition.ReprC = false
				case "empty":
					definition.Fields = nil
				case "cycle":
					definition.Fields[0].Type = hir.FixedArray(definition.Type, 3)
				}
			}
			if !found || len(mir.Verify(program)) == 0 {
				t.Fatal("forged element graph accepted")
			}
			if _, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile()); err == nil {
				t.Fatal("forged element graph emitted")
			}
		})
	}
}
