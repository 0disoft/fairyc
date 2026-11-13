package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestCandidateNestedOptionPipeline(t *testing.T) {
	var source, observer, checks strings.Builder
	source.WriteString(`module root;
fn Option<Option<T>> nested<T>(T value,i32 mode){if(mode==0){return .none;}if(mode==1){return .some(.none);}return .some(.some(value));}
fn Option<Option<T>> relay_nested<T>(Option<Option<T>> value){let Option<Option<T>> copied=value;return copied;}
fn T choose_nested<T>(Option<Option<T>> value,T fallback){return switch(value){case .none:{fallback}case .some(inner):{switch(inner){case .none:{fallback}case .some(actual):{actual}}}};}
fn i32 state_nested<T>(Option<Option<T>> value){return switch(value){case .none:{0}case .some(inner):{switch(inner){case .none:{1}case .some(_):{2}}}};}
fn T transport_nested<T>(T value,T fallback,i32 mode,bool copy){let Option<Option<T>> original=nested<T>(value,mode);let Option<Option<T>> copied=relay_nested<T>(original);assert(state_nested<T>(original)==mode && state_nested<T>(copied)==mode);return if(copy){choose_nested<T>(copied,fallback)}else{choose_nested<T>(original,fallback)};}
`)
	observer.WriteString("#include <stdint.h>\n#include <string.h>\n")
	for _, v := range []struct{ fy, c, value, fallback string }{
		{"i8", "int8_t", "INT8_MIN", "INT8_MAX"}, {"i16", "int16_t", "INT16_MIN", "INT16_MAX"},
		{"isize", "intptr_t", "(-INTPTR_MAX-1)", "INTPTR_MAX"}, {"u16", "uint16_t", "UINT16_MAX", "0"}, {"u64", "uint64_t", "UINT64_MAX", "0"},
		{"f32", "float", "-0.0f", "1.25f"}, {"f64", "double", "-0.0", "-3.5"},
	} {
		for _, copy := range []bool{false, true} {
			name := fmt.Sprintf("probe_%s_%t", v.fy, copy)
			fmt.Fprintf(&source, "pub fn %s %s(%s value,%s fallback,i32 mode){return transport_nested<%s>(value,fallback,mode,%t);}", v.fy, name, v.fy, v.fy, v.fy, copy)
			fmt.Fprintf(&observer, "%s fairyc_%s(%s,%s,int32_t);\n", v.c, name, v.c, v.c)
			fmt.Fprintf(&checks, "for(int32_t mode=0;mode<3;mode++){%s value=%s,fallback=%s,expected=mode==2?value:fallback; %s actual=fairyc_%s(value,fallback,mode);if(memcmp(&actual,&expected,sizeof actual))return 1;}\n", v.c, v.value, v.fallback, v.c, name)
			if v.fy == "f32" || v.fy == "f64" {
				fmt.Fprintf(&checks, "{%s values[]={0.0,-0.0,1.25};for(int i=0;i<3;i++)for(int32_t mode=0;mode<3;mode++){%s value=values[i],fallback=values[(i+1)%%3],expected=mode==2?value:fallback; %s actual=fairyc_%s(value,fallback,mode);if(memcmp(&actual,&expected,sizeof actual))return 2;}}\n", v.c, v.c, v.c, name)
			}
		}
	}
	source.WriteString("unsafe extern C fn i32 observe();fn i32 main(){return unsafe{observe()};}")
	observer.WriteString("int32_t observe(void){" + checks.String() + "return 42;}\n")
	runCandidateTaggedTransportPipeline(t, source.String(), observer.String())
}
