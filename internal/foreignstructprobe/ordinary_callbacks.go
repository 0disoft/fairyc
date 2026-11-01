package foreignstructprobe

import (
	"fmt"
	"strings"
)

// GenerateOrdinaryCallbacks uses genuine synchronous C-to-Fairy function pointers,
// with an independent C observer for values that Fairy does not operate on.
func GenerateOrdinaryCallbacks(posix bool) Unit {
	var f, c strings.Builder
	f.WriteString("module main;\n")
	c.WriteString("#include <stdint.h>\n#include <stddef.h>\n#include <limits.h>\n#include <string.h>\n")
	if posix {
		c.WriteString("#include <sys/types.h>\n")
	}
	types := cArrayProbeTypes(posix)
	var calls []string
	for _, typ := range types {
		name := "c_" + typ.name
		ordinaryScalarCallback(&f, &c, name, "c::"+typ.name, typ.c, []string{typ.first, typ.middle, typ.last}, &calls)
	}
	for _, typ := range []struct{ name, c string }{{"u8", "uint8_t"}, {"i32", "int32_t"}, {"i64", "int64_t"}, {"u32", "uint32_t"}, {"usize", "uintptr_t"}, {"i8", "int8_t"}, {"i16", "int16_t"}, {"u16", "uint16_t"}, {"u64", "uint64_t"}, {"isize", "intptr_t"}, {"f32", "float"}, {"f64", "double"}} {
		values := []string{"0", "1", "42"}
		switch typ.name {
		case "i8", "i16", "i32":
			values = []string{"-100", "1", "100"}
		case "i64", "isize":
			values = []string{"-INT64_C(4294967311)", "1", "INT64_C(4294967311)"}
		case "u64", "usize":
			values = []string{"0", "1", "UINT64_C(9223372036854775825)"}
		}
		ordinaryScalarCallback(&f, &c, typ.name, typ.name, typ.c, values, &calls)
	}
	f.WriteString("unsafe extern C fn i32 ordinary_bump(ptr mut i32 data);\n")
	c.WriteString("int32_t ordinary_bump(int32_t *data){return ++*data;}\n")
	for n := 0; n <= 8; n++ {
		var signature, parameters, args, terms []string
		for i := 0; i < n; i++ {
			signature = append(signature, "i32")
			parameters = append(parameters, fmt.Sprintf("i32 a%d", i))
			args = append(args, fmt.Sprintf("%d", i+1))
			terms = append(terms, fmt.Sprintf("a%d*%d", i, i+1))
		}
		result := "i32"
		cresult := "int32_t"
		body := "return " + strings.Join(terms, "+") + ";"
		if n == 0 {
			result = "void"
			cresult = "void"
			body = ""
		}
		fmt.Fprintf(&f, "unsafe extern C fn i32 ordinary_arity_%[1]d(unsafe extern C fn %[2]s(%[3]s) cb,ptr mut i32 data);\nunsafe fn %[2]s ordinary_handle_%[1]d(%[4]s){let i32 count=ordinary_bump(data);%[5]s}\n", n, result, strings.Join(append(signature, "ptr mut i32"), ","), strings.Join(append(parameters, "ptr mut i32 data"), ","), body)
		cparams := strings.Repeat("int32_t, ", n) + "int32_t *"
		invoke := fmt.Sprintf("cb(%s)", strings.Join(append(args, "data"), ","))
		check := invoke + ";"
		if n > 0 {
			sum := n * (n + 1) * (2*n + 1) / 6
			check = fmt.Sprintf("if(%s!=%d)return 0;", invoke, sum)
		}
		fmt.Fprintf(&c, "int32_t ordinary_arity_%d(%s (*cb)(%s),int32_t *data){if(!cb||!data)return 0;for(int i=0;i<3;i++){%s}return *data==3?42:0;}\n", n, cresult, cparams, check)
		calls = append(calls, fmt.Sprintf("{var i32 count=0;unsafe{ordinary_arity_%d(ordinary_handle_%d,ref mut count as ptr mut i32)}==42&&count==3}", n, n))
	}

	f.WriteString(`
unsafe extern C fn i32 ordinary_mixed(unsafe extern C fn c::double(c::double,c::float,c::int,c::double,i64,c::float,c::int,c::double,ptr mut i32) cb,ptr mut i32 data);
unsafe extern C fn c::double ordinary_mixed_helper(c::double a,c::float b,c::int n,c::double d,i64 e,c::float f,c::int g,c::double h,ptr mut i32 data);
unsafe fn c::double ordinary_mixed_handler(c::double a,c::float b,c::int n,c::double d,i64 e,c::float f,c::int g,c::double h,ptr mut i32 data){return ordinary_mixed_helper(a,b,n,d,e,f,g,h,data);}
`)
	c.WriteString(`
double ordinary_mixed_helper(double a,float b,int n,double d,int64_t e,float f,int g,double h,int32_t *data){if(a!=1.5||b!=2.5f||n!=3||d!=4.5||e!=INT64_C(4294967311)||f!=6.5f||g!=7||h!=8.5)return -1;(*data)++;return 42.25;}
int32_t ordinary_mixed(double (*cb)(double,float,int,double,int64_t,float,int,double,int32_t *),int32_t *data){if(!cb||!data)return 0;for(int i=0;i<3;i++){if(cb(1.5,2.5f,3,4.5,INT64_C(4294967311),6.5f,7,8.5,data)!=42.25)return 0;}return *data==3?42:0;}
`)
	calls = append(calls, "{var i32 count=0;unsafe{ordinary_mixed(ordinary_mixed_handler,ref mut count as ptr mut i32)}==42&&count==3}")
	// Exact pointer parameters/results, nested output slots, and const typed user data.
	f.WriteString(`
unsafe extern C fn i32 ordinary_pointers(unsafe extern C fn ptr mut c::int(ptr c::int,ptr mut c::int,ptr mut ptr mut c::int,ptr c::int) cb,ptr c::int data);
unsafe extern C fn ptr mut c::int ordinary_pointer_helper(ptr c::int a,ptr mut c::int b,ptr mut ptr mut c::int out,ptr c::int data);
unsafe fn ptr mut c::int ordinary_pointer_handler(ptr c::int a,ptr mut c::int b,ptr mut ptr mut c::int out,ptr c::int data){return ordinary_pointer_helper(a,b,out,data);}
unsafe extern C fn ptr c::int ordinary_pointer_context();
`)
	c.WriteString(`
static int ordinary_context=11;
const int *ordinary_pointer_context(void){return &ordinary_context;}
int *ordinary_pointer_helper(const int *a,int *b,int **out,const int *data){if(!a||!b||!out||data!=&ordinary_context||*a!=11)return NULL;*b+=*data;*out=b;return b;}
int32_t ordinary_pointers(int *(*cb)(const int *,int *,int **,const int *),const int *data){int v=20;int *out=NULL;if(!cb||data!=&ordinary_context)return 0;int *r=cb(data,&v,&out,data);return r==&v&&out==&v&&v==31?42:0;}
`)
	calls = append(calls, "unsafe{ordinary_pointers(ordinary_pointer_handler,ordinary_pointer_context())}==42")
	fmt.Fprintf(&f, "fn i32 main(){return if(%s){42}else{1};}\n", strings.Join(calls, "&&"))
	return Unit{FairySource: f.String(), CompanionC: c.String(), ExpectedExit: 42, RequiredCFragments: []string{"= &fy_cb_", "typedef void (*fy_c_callback_", "const int * fy_a_3"}}
}

func ordinaryScalarCallback(f, c *strings.Builder, name, typ, ctype string, values []string, calls *[]string) {
	fmt.Fprintf(f, "unsafe extern C fn i32 ordinary_%[1]s(unsafe extern C fn %[2]s(%[2]s,ptr mut void) cb,ptr mut void data);\nunsafe fn %[2]s ordinary_id_%[1]s(%[2]s value,ptr mut void data){return value;}\n", name, typ)
	// Compare object representations, observing signed zero and finite floats exactly.
	if ctype == "float" || ctype == "double" {
		values = []string{"-0.0", "1.5", "42.25"}
	}
	fmt.Fprintf(c, "int32_t ordinary_%[1]s(%[2]s (*cb)(%[2]s,void *),void *data){if(!cb||data)return 0;%[2]s values[3]={(%[2]s)(%[3]s),(%[2]s)(%[4]s),(%[2]s)(%[5]s)};for(int i=0;i<3;i++){%[2]s result=cb(values[i],data);if(memcmp(&result,&values[i],sizeof(result))!=0)return 0;}return 42;}\n", name, ctype, values[0], values[1], values[2])
	*calls = append(*calls, fmt.Sprintf("unsafe{ordinary_%s(ordinary_id_%s,null)}==42", name, name))
}
