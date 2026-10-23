package compiler_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func cScalarArraySource(scalar string, length int) string {
	return fmt.Sprintf(`module main;
@repr(C) struct Packet{u8 lead;%[1]s[%[2]d] values;u8 tail;}
@repr(C) struct Envelope{Packet packet;}
unsafe extern C fn Packet produce();
unsafe extern C fn i32 consume(Packet original,Envelope saved,Packet rebuilt,Packet changed);
fn %[1]s[%[2]d] copy_values(%[1]s[%[2]d] value){return value;}
fn void replace(ref mut Packet p,Packet q){p.values=q.values;}
fn i32 main(){
 let Packet original=unsafe{produce()};
 let Envelope saved=Envelope{.packet=original};
 let ref Envelope view=ref saved;
 let %[1]s[%[2]d] typed=copy_values(view.packet.values);
 let inferred=copy_values(typed);
 let Packet rebuilt=Packet{.lead=original.lead,.values=inferred,.tail=original.tail};
 var Packet changed=rebuilt;
 changed.values=original.values;
 replace(ref mut changed,original);
 return unsafe{consume(original,saved,rebuilt,changed)};
}`, scalar, length)
}

func TestCScalarArrayTransport(t *testing.T) {
	cTypes := map[string]string{
		"c::bool": "_Bool", "c::char": "char", "c::double": "double", "c::float": "float",
		"c::int": "int", "c::long": "long", "c::longlong": "long long", "c::schar": "signed char",
		"c::short": "short", "c::size_t": "size_t", "c::ssize_t": "ssize_t", "c::uchar": "unsigned char",
		"c::uint": "unsigned int", "c::ulong": "unsigned long", "c::ulonglong": "unsigned long long", "c::ushort": "unsigned short",
	}
	for _, scalar := range hir.CScalarTypes() {
		t.Run(scalar.String(), func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", cScalarArraySource(scalar.String(), 3))})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked.Semantic)
			}
			encoded, err := mircodec.Encode(checked.MIR, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			program, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, definition := range program.StructTypes {
				if definition.Name == "Packet" {
					found = definition.Fields[1].Type == hir.FixedArray(scalar, 3)
				}
			}
			if !found {
				t.Fatal("exact C array identity lost in MIR round trip")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
				unit, err := cbackend.Emit(program, "main::main", profile)
				if scalar == hir.TypeCSSizeT && profile.Target == cbackend.Stage0WindowsMSVCProfile().Target {
					if err == nil {
						t.Fatal("POSIX ssize_t admitted on Windows")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				fragment := cTypes[scalar.String()] + " values[UINT64_C(3)];"
				if !strings.Contains(unit.Source, fragment) {
					t.Fatalf("missing exact C declaration %q", fragment)
				}
				if scalar == hir.TypeCLong {
					size, tail := 40, 32
					if profile.Target == cbackend.Stage0WindowsMSVCProfile().Target {
						size, tail = 20, 16
					}
					for _, expected := range []string{
						fmt.Sprintf("sizeof(struct Packet) == UINT64_C(%d)", size),
						fmt.Sprintf("offsetof(struct Packet, tail) == UINT64_C(%d)", tail),
					} {
						if !strings.Contains(unit.Source, expected) {
							t.Fatalf("missing target-owned layout: %s", expected)
						}
					}
				}
			}
		})
	}
}

func TestCScalarArrayTransportBoundaries(t *testing.T) {
	for name, source := range map[string]string{
		"private":           `struct P{c::int[3] values;} fn i32 main(){return 0;}`,
		"zero":              `@repr(C) struct P{c::int[0] values;} fn i32 main(){return 0;}`,
		"oversize":          `@repr(C) struct P{c::int[4097] values;} fn i32 main(){return 0;}`,
		"nested":            `@repr(C) struct P{c::int[3][2] values;} fn i32 main(){return 0;}`,
		"index":             `fn c::int bad(c::int[3] v){return v[0];} fn i32 main(){return 0;}`,
		"literal":           `fn c::int[3] bad(c::int v){return [v;3];} fn i32 main(){return 0;}`,
		"view":              `fn void bad(c::int[3] v){let ref c::int[] s=ref v;} fn i32 main(){return 0;}`,
		"identity":          `fn i32[3] bad(c::int[3] v){return v;} fn i32 main(){return 0;}`,
		"floating-identity": `fn f32[3] bad(c::float[3] v){return v;} fn i32 main(){return 0;}`,
		"bool-identity":     `fn bool[3] bad(c::bool[3] v){return v;} fn i32 main(){return 0;}`,
		"element-store":     `@repr(C) struct P{c::int[3] values;} fn void bad(ref mut P p,c::int v){p.values[0]=v;} fn i32 main(){return 0;}`,
		"field-ref":         `@repr(C) struct P{c::int[3] values;} fn void bad(ref P p){let ref c::int[3] v=ref p.values;} fn i32 main(){return 0;}`,
		"foreign":           `unsafe extern C fn c::int[3] produce(); fn i32 main(){let c::int[3] v=unsafe{produce()};return 0;}`,
		"option":            `fn Option<c::int[3]> bad(c::int[3] v){return Some(v);} fn i32 main(){return 0;}`,
		"resource":          `resource R{c::int[3] values;cleanup{}} fn i32 main(){return 0;}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main;"+source)})
			if err != nil {
				t.Fatal(err)
			}
			if checked.Verified {
				if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err == nil {
					t.Fatal("unadmitted C array operation emitted")
				}
			}
		})
	}
	for _, length := range []int{1, 4096} {
		checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", cScalarArraySource("c::int", length))})
		if err != nil || !checked.Verified {
			t.Fatalf("length %d: %v %+v", length, err, checked.Semantic)
		}
		if _, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile()); err != nil {
			t.Fatal(err)
		}
	}
}
