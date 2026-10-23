package foreignstructprobe

import (
	"fmt"
	"strings"
)

// This independent C oracle uses the provider's limits, including LP64/LLP64 long.
// It exercises every portable C scalar, plus ssize_t in the POSIX runner.
func cScalarArraySources(posix bool) (string, string) {
	types := []struct{ name, c, first, middle, last string }{
		{"bool", "_Bool", "0", "1", "0"},
		{"char", "char", "CHAR_MIN", "0", "CHAR_MAX"},
		{"double", "double", "-DBL_MAX", "0.0", "DBL_MAX"},
		{"float", "float", "-FLT_MAX", "0.0f", "FLT_MAX"},
		{"int", "int", "INT_MIN", "0", "INT_MAX"},
		{"long", "long", "LONG_MIN", "0", "LONG_MAX"},
		{"longlong", "long long", "LLONG_MIN", "0", "LLONG_MAX"},
		{"schar", "signed char", "SCHAR_MIN", "0", "SCHAR_MAX"},
		{"short", "short", "SHRT_MIN", "0", "SHRT_MAX"},
		{"size_t", "size_t", "0", "17", "SIZE_MAX"},
		{"uchar", "unsigned char", "0", "17", "UCHAR_MAX"},
		{"uint", "unsigned int", "0", "17", "UINT_MAX"},
		{"ulong", "unsigned long", "0", "17", "ULONG_MAX"},
		{"ulonglong", "unsigned long long", "0", "17", "ULLONG_MAX"},
		{"ushort", "unsigned short", "0", "17", "USHRT_MAX"},
	}
	if posix {
		types = append(types, struct{ name, c, first, middle, last string }{"ssize_t", "ssize_t", "INT64_MIN", "0", "INT64_MAX"})
	}
	var fairy, c strings.Builder
	c.WriteString("\n#include <limits.h>\n#include <float.h>\n#include <stddef.h>\n")
	if posix {
		c.WriteString("#include <sys/types.h>\n")
	}
	var calls []string
	for _, typ := range types {
		fmt.Fprintf(&fairy, cScalarArrayFairyTemplate, typ.name)
		fmt.Fprintf(&c, cScalarArrayCTemplate, typ.name, typ.c, typ.first, typ.middle, typ.last)
		calls = append(calls, "check_c_array_"+typ.name+"(0)", "check_c_array_"+typ.name+"(1)", "check_c_array_"+typ.name+"(2)")
	}
	fmt.Fprintf(&fairy, "fn bool check_c_scalar_arrays(){return %s;}\n", strings.Join(calls, "&&"))
	return fairy.String(), c.String()
}

const cScalarArrayFairyTemplate = `
@repr(C) struct fairyc_c_array_%[1]s {u8 lead;c::%[1]s[3] values;u8 tail;}
@repr(C) struct fairyc_c_array_%[1]s_envelope {fairyc_c_array_%[1]s packet;}
@repr(C) struct fairyc_c_array_%[1]s_small {c::%[1]s[1] values;}
unsafe extern C fn fairyc_c_array_%[1]s fairyc_c_array_%[1]s_make(i32 choice);
unsafe extern C fn fairyc_c_array_%[1]s fairyc_c_array_%[1]s_change(i32 choice,fairyc_c_array_%[1]s value);
unsafe extern C fn fairyc_c_array_%[1]s_small fairyc_c_array_%[1]s_small_make(i32 choice);
unsafe extern C fn i32 fairyc_c_array_%[1]s_observe(i32 choice,fairyc_c_array_%[1]s original,
 fairyc_c_array_%[1]s_envelope saved,fairyc_c_array_%[1]s rebuilt,fairyc_c_array_%[1]s changed,
 fairyc_c_array_%[1]s direct,fairyc_c_array_%[1]s replaced,fairyc_c_array_%[1]s_small small);
fn c::%[1]s[3] copy_c_array_%[1]s(c::%[1]s[3] value){return value;}
fn c::%[1]s[1] copy_c_array_%[1]s_small(c::%[1]s[1] value){return value;}
fn void replace_c_array_%[1]s(ref mut fairyc_c_array_%[1]s p,fairyc_c_array_%[1]s q){p.values=q.values;}
fn bool check_c_array_%[1]s(i32 choice){
 let fairyc_c_array_%[1]s original=unsafe{fairyc_c_array_%[1]s_make(choice)};
 let fairyc_c_array_%[1]s_envelope saved=fairyc_c_array_%[1]s_envelope{.packet=original};
 let ref fairyc_c_array_%[1]s_envelope view=ref saved;
 let c::%[1]s[3] typed=copy_c_array_%[1]s(view.packet.values);
 let inferred=copy_c_array_%[1]s(typed);
 let fairyc_c_array_%[1]s rebuilt=fairyc_c_array_%[1]s{.lead=original.lead,.values=inferred,.tail=original.tail};
 let fairyc_c_array_%[1]s changed=unsafe{fairyc_c_array_%[1]s_change(choice,rebuilt)};
 var fairyc_c_array_%[1]s direct=original;direct.values=changed.values;
 var fairyc_c_array_%[1]s replaced=original;replace_c_array_%[1]s(ref mut replaced,changed);
 let fairyc_c_array_%[1]s_small small=unsafe{fairyc_c_array_%[1]s_small_make(choice)};
 let fairyc_c_array_%[1]s_small small_copy=fairyc_c_array_%[1]s_small{.values=copy_c_array_%[1]s_small(small.values)};
 return unsafe{fairyc_c_array_%[1]s_observe(choice,original,saved,rebuilt,changed,direct,replaced,small_copy)}==42;
}
`

const cScalarArrayCTemplate = `
struct fairyc_c_array_%[1]s {uint8_t lead;%[2]s values[3];uint8_t tail;};
struct fairyc_c_array_%[1]s_envelope {struct fairyc_c_array_%[1]s packet;};
struct fairyc_c_array_%[1]s_small {%[2]s values[1];};
static const %[2]s fairyc_c_array_%[1]s_expected[3]={%[3]s,%[4]s,%[5]s};
struct fairyc_c_array_%[1]s fairyc_c_array_%[1]s_make(int32_t choice){
 struct fairyc_c_array_%[1]s value={0};value.lead=11;value.tail=13;
 for(unsigned i=0;i<3;++i){value.values[i]=fairyc_c_array_%[1]s_expected[(i+(unsigned)choice)%%3];}
 return value;
}
static int fairyc_c_array_%[1]s_valid(int32_t choice,struct fairyc_c_array_%[1]s value){
 if(value.lead!=11||value.tail!=13){return 0;}
 for(unsigned i=0;i<3;++i){if(value.values[i]!=fairyc_c_array_%[1]s_expected[(i+(unsigned)choice)%%3]){return 0;}}
 return 1;
}
struct fairyc_c_array_%[1]s fairyc_c_array_%[1]s_change(int32_t choice,struct fairyc_c_array_%[1]s value){
 if(!fairyc_c_array_%[1]s_valid(choice,value)){struct fairyc_c_array_%[1]s failed={0};return failed;}
 for(unsigned i=0;i<3;++i){value.values[i]=fairyc_c_array_%[1]s_expected[(i+(unsigned)choice+1)%%3];}
 return value;
}
struct fairyc_c_array_%[1]s_small fairyc_c_array_%[1]s_small_make(int32_t choice){
 struct fairyc_c_array_%[1]s_small value={{0}};value.values[0]=fairyc_c_array_%[1]s_expected[(unsigned)choice%%3];return value;
}
int32_t fairyc_c_array_%[1]s_observe(int32_t choice,struct fairyc_c_array_%[1]s original,
 struct fairyc_c_array_%[1]s_envelope saved,struct fairyc_c_array_%[1]s rebuilt,struct fairyc_c_array_%[1]s changed,
 struct fairyc_c_array_%[1]s direct,struct fairyc_c_array_%[1]s replaced,struct fairyc_c_array_%[1]s_small small){
 return fairyc_c_array_%[1]s_valid(choice,original)&&fairyc_c_array_%[1]s_valid(choice,saved.packet)&&
  fairyc_c_array_%[1]s_valid(choice,rebuilt)&&fairyc_c_array_%[1]s_valid(choice+1,changed)&&
  fairyc_c_array_%[1]s_valid(choice+1,direct)&&fairyc_c_array_%[1]s_valid(choice+1,replaced)&&
  small.values[0]==fairyc_c_array_%[1]s_expected[(unsigned)choice%%3]?42:0;
}
`
