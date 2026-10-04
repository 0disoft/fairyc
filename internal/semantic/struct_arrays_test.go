package semantic_test

import (
	"testing"
)

func TestStructArrayRejections(t *testing.T) {
	for name, source := range map[string]string{
		"nominal-mismatch": `struct A{i32 x;}struct B{i32 x;}fn void bad(){let A[1] a=[B{.x=1}];}`,
		"resource":         `resource R{i32 x;cleanup{}}fn void bad(R[1] data){}`,
		"nested-resource":  `resource R{i32 x;cleanup{}}struct S{R owner;}fn void bad(S[1] data){}`,
		"pointer":          `struct S{ptr i32 pointer;}fn void bad(S[1] data){}`,
		"borrowed":         `struct S{ref i32 value;}fn void bad(S[1] data){}`,
		"tagged":           `struct S{Option<i32> value;}fn void bad(S[1] data){}`,
		"zero":             `struct S{i32 value;}fn void bad(S[0] data){}`,
		"oversize":         `struct S{i32 value;}fn void bad(S[4097] data){}`,
		"local-escape":     `struct S{i32 value;}fn ref S bad(){var S[1] data=[S{.value=1}];return ref data[0];}`,
		"shared-store":     `struct S{i32 value;}fn void bad(ref S[] data){data[0]=S{.value=1};}`,
		"live-array-write": `struct S{i32 value;}fn void bad(){var S[1] data=[S{.value=1}];for(let ref S item in data){data[0]=S{.value=2};assert(item.value==1);}}`,
		"sibling-loans":    `struct S{i32 value;}fn void bad(){var S[2] data=[S{.value=1};2];let ref mut S a=ref mut data[0];let ref mut S b=ref mut data[1];a.value=b.value;}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + source})
			if len(r.Diagnostics)+len(r.Unsupported) == 0 {
				t.Fatal("unadmitted struct array accepted")
			}
		})
	}
}
