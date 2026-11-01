package foreignstructprobe

import (
	"fmt"
	"slices"
	"strings"
)

// PointerUnit adds fixture-only C bridges to ordinary generated Fairy pointer
// identity functions. It does not publish a source export or callback signature.
type PointerUnit struct {
	Unit
	types []cArrayProbeType
}

func GenerateCScalarPointers(posix bool) PointerUnit {
	var fairy, c strings.Builder
	fairy.WriteString("module main;\n")
	c.WriteString("#include <stdint.h>\n#include <stddef.h>\n#include <limits.h>\n#include <float.h>\n")
	if posix {
		c.WriteString("#include <sys/types.h>\n")
	}
	types := cArrayProbeTypes(posix)
	var calls []string
	var fragments []string
	for _, typ := range types {
		fmt.Fprintf(&fairy, cScalarPointerFairy, typ.name)
		fmt.Fprintf(&c, cScalarPointerC, typ.name, typ.c, typ.first, typ.middle, typ.last)
		for choice := 0; choice < 3; choice++ {
			calls = append(calls, fmt.Sprintf("check_pointer_%s(%d)", typ.name, choice))
		}
		fragments = append(fragments, "const "+typ.c+" * shared;", typ.c+" * writable;", typ.c+" * * slot;")
	}
	fmt.Fprintf(&fairy, "fn i32 main(){return if(%s){42}else{1};}\n", strings.Join(calls, "&&"))
	return PointerUnit{Unit: Unit{FairySource: fairy.String(), CompanionC: c.String(), ExpectedExit: 42, RequiredCFragments: fragments}, types: types}
}

func (p PointerUnit) Bind(emitted string, functions []string) (string, error) {
	var bridges strings.Builder
	for _, typ := range p.types {
		for _, shape := range []struct{ name, c string }{{"shared", "const " + typ.c + " *"}, {"mut", typ.c + " *"}, {"slot", typ.c + " * *"}} {
			name := "id_pointer_" + shape.name + "_" + typ.name
			index := slices.Index(functions, "main::"+name)
			if index < 0 || !strings.Contains(emitted, fmt.Sprintf("static %s fy_fn_%d(%s ", shape.c, index, shape.c)) {
				return "", fmt.Errorf("missing exact generated pointer identity %s", name)
			}
			fmt.Fprintf(&bridges, "\n%s reverse_pointer_%s_%s(%s p){return fy_fn_%d(p);}\n", shape.c, shape.name, typ.name, shape.c, index)
		}
	}
	return emitted + bridges.String(), nil
}

const cScalarPointerFairy = `
@repr(C) struct Pointer_%[1]s{ptr c::%[1]s shared;ptr mut c::%[1]s writable;ptr mut ptr mut c::%[1]s slot;}
unsafe extern C fn c::%[1]s pointer_value_%[1]s(i32 choice);
unsafe extern C fn ptr c::%[1]s pointer_shared_%[1]s(i32 choice);
unsafe extern C fn ptr mut c::%[1]s pointer_mut_%[1]s(i32 choice);
unsafe extern C fn ptr mut ptr mut c::%[1]s pointer_slot_%[1]s(i32 choice);
unsafe extern C fn i32 pointer_fill_%[1]s(i32 choice,ptr mut ptr mut c::%[1]s output);
unsafe extern C fn i32 pointer_clear_%[1]s(ptr mut ptr mut c::%[1]s output);
unsafe extern C fn i32 pointer_observe_%[1]s(i32 choice,Pointer_%[1]s packet);
unsafe extern C fn i32 pointer_mutate_%[1]s(i32 choice,ptr c::%[1]s shared,ptr mut c::%[1]s writable);
unsafe extern C fn i32 pointer_check_value_%[1]s(i32 choice,c::%[1]s value);
unsafe extern C fn i32 pointer_read_%[1]s(i32 choice,ptr c::%[1]s value);
unsafe extern C fn i32 pointer_reverse_%[1]s(i32 choice,ptr c::%[1]s shared,ptr mut c::%[1]s writable,ptr mut ptr mut c::%[1]s slot);
fn ptr c::%[1]s id_pointer_shared_%[1]s(ptr c::%[1]s p){return p;}
fn ptr mut c::%[1]s id_pointer_mut_%[1]s(ptr mut c::%[1]s p){return p;}
fn ptr mut ptr mut c::%[1]s id_pointer_slot_%[1]s(ptr mut ptr mut c::%[1]s p){return p;}
fn i32 pointer_fill_borrow_%[1]s(i32 choice,ref mut ptr mut c::%[1]s slot){return unsafe{pointer_fill_%[1]s(choice,slot as ptr mut ptr mut c::%[1]s)};}
fn i32 pointer_mutate_borrow_%[1]s(i32 choice,ref mut c::%[1]s value){return unsafe{pointer_mutate_%[1]s(choice,value as ptr c::%[1]s,value as ptr mut c::%[1]s)};}
fn i32 pointer_read_borrow_%[1]s(i32 choice,ref c::%[1]s value){return unsafe{pointer_read_%[1]s(choice,value as ptr c::%[1]s)};}
fn bool check_pointer_%[1]s(i32 choice){
 let ptr c::%[1]s shared=id_pointer_shared_%[1]s(unsafe{pointer_shared_%[1]s(choice)});
 let ptr mut c::%[1]s writable=id_pointer_mut_%[1]s(unsafe{pointer_mut_%[1]s(choice)});
 var ptr mut c::%[1]s output=null;
 let i32 direct_fill=unsafe{pointer_fill_%[1]s(choice,ref mut output as ptr mut ptr mut c::%[1]s)};
 let Pointer_%[1]s packet=Pointer_%[1]s{.shared=shared,.writable=writable,.slot=unsafe{ref mut output as ptr mut ptr mut c::%[1]s}};
 let i32 observed=unsafe{pointer_observe_%[1]s(choice,packet)};
 let i32 cleared=unsafe{pointer_clear_%[1]s(ref mut output as ptr mut ptr mut c::%[1]s)};
 let bool was_null=output==null;
 let i32 borrowed_fill=pointer_fill_borrow_%[1]s(choice,ref mut output);
 let bool same=output==writable;
 let ptr mut ptr mut c::%[1]s nested=id_pointer_slot_%[1]s(unsafe{pointer_slot_%[1]s(choice)});
 let i32 reversed=unsafe{pointer_reverse_%[1]s(choice,shared,writable,nested)};
 var c::%[1]s value=unsafe{pointer_value_%[1]s(choice)};
 let i32 shared_read=pointer_read_borrow_%[1]s(choice,ref value);
 let i32 direct_mutation=unsafe{pointer_mutate_%[1]s(choice,ref value as ptr c::%[1]s,ref mut value as ptr mut c::%[1]s)};
 let i32 direct_after=unsafe{pointer_check_value_%[1]s(choice+1,value)};
 let i32 borrowed_mutation=pointer_mutate_borrow_%[1]s(choice+1,ref mut value);
 let i32 borrowed_after=unsafe{pointer_check_value_%[1]s(choice+2,value)};
 let i32 null_slot=unsafe{pointer_fill_%[1]s(choice,null)};
 return direct_fill==42&&observed==42&&cleared==42&&was_null&&borrowed_fill==42&&same&&reversed==42&&shared_read==42&&direct_mutation==42&&direct_after==42&&borrowed_mutation==42&&borrowed_after==42&&null_slot==7;
}
`

const cScalarPointerC = `
struct Pointer_%[1]s{const %[2]s *shared;%[2]s *writable;%[2]s **slot;};
static const %[2]s pointer_expected_%[1]s[3]={%[3]s,%[4]s,%[5]s};
static %[2]s pointer_values_%[1]s[3];
static %[2]s *pointer_global_slot_%[1]s;
extern const %[2]s *reverse_pointer_shared_%[1]s(const %[2]s *);
extern %[2]s *reverse_pointer_mut_%[1]s(%[2]s *);
extern %[2]s **reverse_pointer_slot_%[1]s(%[2]s **);
%[2]s pointer_value_%[1]s(int32_t choice){return pointer_expected_%[1]s[(unsigned)choice%%3];}
const %[2]s *pointer_shared_%[1]s(int32_t choice){return &pointer_expected_%[1]s[(unsigned)choice%%3];}
%[2]s *pointer_mut_%[1]s(int32_t choice){unsigned i=(unsigned)choice%%3;pointer_values_%[1]s[i]=pointer_expected_%[1]s[i];return &pointer_values_%[1]s[i];}
%[2]s **pointer_slot_%[1]s(int32_t choice){pointer_global_slot_%[1]s=pointer_mut_%[1]s(choice);return &pointer_global_slot_%[1]s;}
int32_t pointer_fill_%[1]s(int32_t choice,%[2]s **output){if(output==NULL){return 7;}*output=pointer_mut_%[1]s(choice);return 42;}
int32_t pointer_clear_%[1]s(%[2]s **output){if(output==NULL){return 7;}*output=NULL;return 42;}
int32_t pointer_observe_%[1]s(int32_t choice,struct Pointer_%[1]s packet){
 unsigned i=(unsigned)choice%%3;
 if(packet.shared!=&pointer_expected_%[1]s[i]||packet.writable!=&pointer_values_%[1]s[i]||packet.slot==NULL||*packet.slot!=packet.writable){return 0;}
 return *packet.shared==pointer_expected_%[1]s[i]&&*packet.writable==pointer_expected_%[1]s[i]?42:0;
}
int32_t pointer_mutate_%[1]s(int32_t choice,const %[2]s *shared,%[2]s *writable){
 if(shared==NULL||shared!=writable||*shared!=pointer_expected_%[1]s[(unsigned)choice%%3]){return 0;}
 *writable=pointer_expected_%[1]s[((unsigned)choice+1)%%3];return 42;
}
int32_t pointer_read_%[1]s(int32_t choice,const %[2]s *p){return p!=NULL&&*p==pointer_expected_%[1]s[(unsigned)choice%%3]?42:0;}
int32_t pointer_check_value_%[1]s(int32_t choice,%[2]s value){return value==pointer_expected_%[1]s[(unsigned)choice%%3]?42:0;}
int32_t pointer_reverse_%[1]s(int32_t choice,const %[2]s *shared,%[2]s *writable,%[2]s **slot){
 unsigned i=(unsigned)choice%%3;
 return reverse_pointer_shared_%[1]s(shared)==&pointer_expected_%[1]s[i]&&
 reverse_pointer_mut_%[1]s(writable)==&pointer_values_%[1]s[i]&&
 slot==&pointer_global_slot_%[1]s&&reverse_pointer_slot_%[1]s(slot)==slot&&
 reverse_pointer_shared_%[1]s(NULL)==NULL&&reverse_pointer_mut_%[1]s(NULL)==NULL&&reverse_pointer_slot_%[1]s(NULL)==NULL?42:0;
}
`
