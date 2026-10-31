package foreignstructprobe

import (
	"fmt"
	"strings"
)

// The independently compiled C provider owns element values and checks every
// member after Fairy copies/reconstructs/replaces the enclosing record.
func recordArraySources() (string, string) {
	var fairy, c strings.Builder
	fairy.WriteString("\n@repr(C) struct fairyc_record_leaf{c::double value;}\n@repr(c::uint) enum FairycRecordState{Zero;One;}\n")
	c.WriteString("\nstruct fairyc_record_leaf{double value;};\n")
	template := strings.NewReplacer("c_array_", "record_array_", "c::%[1]s", "fairyc_record_element_%[1]s").Replace(cScalarArrayFairyTemplate)
	var calls []string
	for _, typ := range []struct{ name, fairy, c, values, equal string }{
		{"small", "u32 code;", "uint32_t code;", "{0},{UINT32_MAX},{17}", "return a.code==b.code;"},
		{"max", "u32 code;", "uint32_t code;", "{0},{UINT32_MAX},{17}", "return a.code==b.code;"},
		{"mixed", "fairyc_record_leaf nested;c::long code;f64 ratio;", "struct fairyc_record_leaf nested;long code;double ratio;", "{{-DBL_MAX},LONG_MIN,-DBL_MAX},{{0.0},0,0.0},{{DBL_MAX},LONG_MAX,DBL_MAX}", "return a.nested.value==b.nested.value&&a.code==b.code&&a.ratio==b.ratio;"},
		{"arrays", "c::int[2] codes;f32[2] samples;FairycRecordState[2] states;fairyc_record_leaf[2] nested;", "int codes[2];float samples[2];unsigned int states[2];struct fairyc_record_leaf nested[2];", "{{INT_MIN,INT_MAX},{-FLT_MAX,FLT_MAX},{0,UINT_MAX},{{-DBL_MAX},{DBL_MAX}}},{{0,17},{0.0f,1.0f},{1,0},{{0.0},{1.0}}},{{INT_MAX,INT_MIN},{FLT_MAX,-FLT_MAX},{UINT_MAX,1},{{DBL_MAX},{-DBL_MAX}}}", "for(unsigned i=0;i<2;++i){if(a.codes[i]!=b.codes[i]||a.samples[i]!=b.samples[i]||a.states[i]!=b.states[i]||a.nested[i].value!=b.nested[i].value){return 0;}}return 1;"},
	} {
		fmt.Fprintf(&fairy, "\n@repr(C) struct fairyc_record_element_%s{%s}\n", typ.name, typ.fairy)
		fairyTemplate, cTemplate := template, recordArrayCTemplate
		if typ.name == "max" {
			fairyTemplate = strings.ReplaceAll(fairyTemplate, "[3]", "[4096]")
			cTemplate = strings.NewReplacer("values[3]", "values[4096]", "i<3", "i<4096").Replace(cTemplate)
		}
		fmt.Fprintf(&fairy, fairyTemplate, typ.name)
		fmt.Fprintf(&c, cTemplate, typ.name, typ.c, typ.values, typ.equal)
		for choice := 0; choice < 3; choice++ {
			calls = append(calls, fmt.Sprintf("check_record_array_%s(%d)", typ.name, choice))
		}
	}
	fmt.Fprintf(&fairy, "fn bool check_record_arrays(){return %s;}\n", strings.Join(calls, "&&"))
	return fairy.String(), c.String()
}

const recordArrayCTemplate = `
struct fairyc_record_element_%[1]s{%[2]s};
struct fairyc_record_array_%[1]s{uint8_t lead;struct fairyc_record_element_%[1]s values[3];uint8_t tail;};
struct fairyc_record_array_%[1]s_envelope{struct fairyc_record_array_%[1]s packet;};
struct fairyc_record_array_%[1]s_small{struct fairyc_record_element_%[1]s values[1];};
static const struct fairyc_record_element_%[1]s fairyc_record_array_%[1]s_expected[3]={%[3]s};
static int fairyc_record_array_%[1]s_equal(struct fairyc_record_element_%[1]s a,struct fairyc_record_element_%[1]s b){%[4]s}
struct fairyc_record_array_%[1]s fairyc_record_array_%[1]s_make(int32_t choice){
 struct fairyc_record_array_%[1]s value={0};value.lead=11;value.tail=13;
 for(unsigned i=0;i<3;++i){value.values[i]=fairyc_record_array_%[1]s_expected[(i+(unsigned)choice)%%3];}
 return value;
}
static int fairyc_record_array_%[1]s_valid(int32_t choice,struct fairyc_record_array_%[1]s value){
 if(value.lead!=11||value.tail!=13){return 0;}
 for(unsigned i=0;i<3;++i){if(!fairyc_record_array_%[1]s_equal(value.values[i],fairyc_record_array_%[1]s_expected[(i+(unsigned)choice)%%3])){return 0;}}
 return 1;
}
struct fairyc_record_array_%[1]s fairyc_record_array_%[1]s_change(int32_t choice,struct fairyc_record_array_%[1]s value){
 if(!fairyc_record_array_%[1]s_valid(choice,value)){struct fairyc_record_array_%[1]s failed={0};return failed;}
 for(unsigned i=0;i<3;++i){value.values[i]=fairyc_record_array_%[1]s_expected[(i+(unsigned)choice+1)%%3];}
 return value;
}
struct fairyc_record_array_%[1]s_small fairyc_record_array_%[1]s_small_make(int32_t choice){
 struct fairyc_record_array_%[1]s_small value={0};value.values[0]=fairyc_record_array_%[1]s_expected[(unsigned)choice%%3];return value;
}
int32_t fairyc_record_array_%[1]s_observe(int32_t choice,struct fairyc_record_array_%[1]s original,
 struct fairyc_record_array_%[1]s_envelope saved,struct fairyc_record_array_%[1]s rebuilt,struct fairyc_record_array_%[1]s changed,
 struct fairyc_record_array_%[1]s direct,struct fairyc_record_array_%[1]s replaced,struct fairyc_record_array_%[1]s_small small){
 return fairyc_record_array_%[1]s_valid(choice,original)&&fairyc_record_array_%[1]s_valid(choice,saved.packet)&&
  fairyc_record_array_%[1]s_valid(choice,rebuilt)&&fairyc_record_array_%[1]s_valid(choice+1,changed)&&
  fairyc_record_array_%[1]s_valid(choice+1,direct)&&fairyc_record_array_%[1]s_valid(choice+1,replaced)&&
  fairyc_record_array_%[1]s_equal(small.values[0],fairyc_record_array_%[1]s_expected[(unsigned)choice%%3])?42:0;
}
`
