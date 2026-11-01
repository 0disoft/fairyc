package main

import (
	"fmt"
	"strings"
)

type workload struct {
	Name, Source, Support, Trap string
	Foreign                     bool
}

func workloads() []workload {
	var growth strings.Builder
	growth.WriteString("module main;")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&growth, "fn i32 work_%d(i32 x){return (x*3+1)/2;}", i)
	}
	growth.WriteString("fn i32 main(){var i32 total=0;")
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&growth, "total+=work_%d(28);", i)
	}
	growth.WriteString("assert(total==5376);return 42;}")
	return []workload{
		{Name: "scalar-control", Source: `module main;fn i32 main(){var i32 sum=0;for(var i32 i=0;i<200;i+=1){sum+=i;}assert(sum==19900);return 42;}`},
		{Name: "inert-record", Source: `module main;struct Pair{i32 first;i32 second;}fn i32 sum(Pair p){return p.first+p.second;}fn i32 main(){return sum(Pair{.first=20,.second=22});}`},
		{Name: "generic-interface-closure", Source: `module main;@interface fn T transform<T>(T value){}@implements(transform,i32) fn i32 twice(i32 value){return value*2;}fn T algorithm<T>(T value){return transform<T>(value);}fn i32 main(){let i32 base=40;let add=fn [i32 base] i32(i32 x){return base+x;};assert(add(2)==42);return algorithm<i32>(21);}`},
		{Name: "reverse-resource-cleanup", Foreign: true, Source: `module main;unsafe extern C fn i32 study_record(i32 value);unsafe extern C fn i32 study_finish();resource Token{i32 value;cleanup{discard unsafe{study_record(this.value)};}}fn i32 scope(){let Token first=Token{.value=1};let Token second=Token{.value=2};return 0;}fn i32 main(){discard scope();return unsafe{study_finish()};}`,
			Support: `static int32_t study_position=0;static int32_t study_ok=1;int32_t study_record(int32_t value){if(study_position>=2||value!=2-study_position)study_ok=0;++study_position;return value;}int32_t study_finish(void){return study_position==2&&study_ok?42:1;}`},
		{Name: "checked-overflow", Trap: "integer-overflow", Source: `module main;fn i32 add(i32 x){return x+1;}fn i32 main(){return add(2147483647);}`},
		{Name: "aggregate-c-crossing", Foreign: true, Source: `module main;@repr(C) struct Payload{i64 wide;i32 value;}unsafe extern C fn Payload study_roundtrip(Payload input);fn i32 main(){let Payload result=unsafe{study_roundtrip(Payload{.wide=2147483648,.value=20})};assert(result.wide==2147483649);return result.value;}`,
			Support: `typedef struct{int64_t wide;int32_t value;} study_payload;study_payload study_roundtrip(study_payload input){input.wide+=1;input.value+=22;return input;}`},
		{Name: "128-function-emission", Source: growth.String()},
	}
}
