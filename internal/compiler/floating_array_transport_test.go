package compiler_test

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

const floatingArrayTransportSource = `module main;
@repr(C) struct Packet{f32[3] single;f64[3] wide;}
@repr(C) struct Envelope{Packet packet;}
unsafe extern C fn Packet produce();
unsafe extern C fn i32 consume(Packet original,Envelope saved,Packet rebuilt,Packet changed);
fn f32[3] copy_single(f32[3] value){return value;}
fn f64[3] copy_wide(f64[3] value){return value;}
fn void replace(ref mut Packet p,Packet q){p.single=q.single;p.wide=q.wide;}
fn i32 main(){
 let Packet p=unsafe{produce()};
 let Envelope e=Envelope{.packet=p};
 let ref Envelope view=ref e;
 let f32[3] single=copy_single(view.packet.single);
 let wide=copy_wide(p.wide);
 let Packet rebuilt=Packet{.single=single,.wide=wide};
 var Packet changed=rebuilt;
 changed.single=p.single;
 replace(ref mut changed,p);
 return unsafe{consume(p,e,rebuilt,changed)};
}`

func TestFloatingArrayTransportMIRRoundTrip(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", floatingArrayTransportSource)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	packet := -1
	for i, definition := range program.StructTypes {
		if definition.Name == "Packet" {
			packet = i
		}
	}
	if packet < 0 {
		t.Fatal("missing Packet")
	}
	for i, typ := range []hir.Type{hir.TypeF32, hir.TypeF64} {
		if program.StructTypes[packet].Fields[i].Type != hir.FixedArray(typ, 3) {
			t.Fatalf("array identity lost: %s", typ)
		}
	}
	for _, mutation := range []string{"private", "oversize", "c-float"} {
		t.Run(mutation, func(t *testing.T) {
			forged, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "private":
				forged.StructTypes[packet].ReprC = false
			case "oversize":
				forged.StructTypes[packet].Fields[0].Type = hir.FixedArray(hir.TypeF32, 4097)
			case "c-float":
				forged.StructTypes[packet].Fields[0].Type = hir.FixedArray(hir.TypeCFloat, 3)
			}
			if len(mir.Verify(forged)) == 0 {
				t.Fatal("forged array declaration accepted")
			}
		})
	}
}

func TestFloatingArrayTransportBoundaries(t *testing.T) {
	for _, typ := range []string{"f32", "f64"} {
		for name, body := range map[string]string{
			"private":       `struct P{%[1]s[3] values;} fn i32 main(){return 0;}`,
			"zero":          `@repr(C) struct P{%[1]s[0] values;} fn i32 main(){return 0;}`,
			"oversize":      `@repr(C) struct P{%[1]s[4097] values;} fn i32 main(){return 0;}`,
			"nested":        `@repr(C) struct P{%[1]s[3][2] values;} fn i32 main(){return 0;}`,
			"index":         `@repr(C) struct P{%[1]s[3] values;} unsafe extern C fn P produce(); fn %[1]s bad(){let P p=unsafe{produce()};return p.values[0];} fn i32 main(){return 0;}`,
			"literal":       `fn %[1]s[3] bad(%[1]s v){return [v;3];} fn i32 main(){return 0;}`,
			"list":          `fn %[1]s[3] bad(%[1]s v){return [v,v,v];} fn i32 main(){return 0;}`,
			"view":          `fn void bad(%[1]s[3] v){let ref %[1]s[] s=ref v;} fn i32 main(){return 0;}`,
			"field-ref":     `@repr(C) struct P{%[1]s[3] values;} fn void bad(ref P p){let ref %[1]s[3] v=ref p.values;} fn i32 main(){return 0;}`,
			"field-mut-ref": `@repr(C) struct P{%[1]s[3] values;} fn void bad(ref mut P p){let ref mut %[1]s[3] v=ref mut p.values;} fn i32 main(){return 0;}`,
			"element-store": `@repr(C) struct P{%[1]s[3] values;} fn void bad(ref mut P p,%[1]s v){p.values[0]=v;} fn i32 main(){return 0;}`,
			"nested-store":  `@repr(C) struct P{%[1]s[3] values;} @repr(C) struct E{P p;} fn void bad(ref mut E e,P q){e.p.values=q.values;} fn i32 main(){return 0;}`,
		} {
			t.Run(typ+"/"+name, func(t *testing.T) {
				checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+fmt.Sprintf(body, typ))})
				if err != nil {
					t.Fatal(err)
				}
				if checked.Verified {
					t.Fatal("unadmitted array operation verified")
				}
			})
		}
	}
}
