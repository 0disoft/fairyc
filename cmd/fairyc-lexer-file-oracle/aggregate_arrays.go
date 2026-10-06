package main

// Test-only fixed expectations for candidate aggregate arrays. The Go compiler
// does not admit rank-two source types; no Go MIR/interpreter parity is claimed.
import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

func writeAggregateArrayCases(path string) error {
	seeds := []struct {
		name, input string
		value       int64
		trap        string
	}{
		{"field-copy", `module main;struct A{i32[2] x;bool keep;}fn i32 main(){var a=A{.keep=true,.x=[1,2]};let b=a;a.x[0]=9;assert(a.x[0]==9 && b.x[0]==1 && a.keep);return 0;}`, 0, ""},
		{"non-square", `module main;fn i32 main(){var u8[2][3] a=[[1,2],[3,4],[5,6]];a[2][1]=9;assert(a.len==3 && a[0].len==2 && a[2][1]==9);return 0;}`, 0, ""},
		{"inferred-repeat", `module main;fn i32 main(){var a=[[1,2];3];a[1]=[7,8];assert(a[0][0]==1 && a[1][0]==7);return 0;}`, 0, ""},
		{"nested-field", `module main;struct A{i64[2][1] a;}struct B{A a;}fn i32 main(){var b=B{.a=A{.a=[[4000000000,7]]}};b.a.a[0][0]+=2;assert(b.a.a[0][0]==4000000002);return 0;}`, 0, ""},
		{"whole-field", `module main;struct A{bool[2] a;}fn A id(A a){return a;}fn i32 main(){var a=A{.a=[false,true]};a.a=[true,false];let b=id(a);assert(b.a[0] && !b.a[1]);return 0;}`, 0, ""},
		{"rank2-return", `module main;fn i32[2][1] id(i32[2][1] a){return a;}fn i32 main(){let a=id([[1,2]]);return a[0][1];}`, 2, ""},
		{"limit64", `module main;fn i32 main(){var bool[64][64] a=[[false;64];64];a[63][63]=true;assert(a[63][63]);return 0;}`, 0, ""},
		{"unused-field", `module main;struct A{u8[2][3] a;}fn i32 main(){return 0;}`, 0, ""},
		{"repeat-once", `module main;fn i32 main(){var i32 n=0;let i32[2][3] a=[[{n+=1;n};2];3];assert(n==1 && a[2][1]==1);return n;}`, 1, ""},
		{"index-order", `module main;fn i32 main(){var a=[[1,2],[3,4]];var i32 n=0;let x=a[{n+=1;0}][{a[0][1]=9;n+=1;1}];assert(n==2 && x==9);return x;}`, 9, ""},
		{"field-index-effects", `module main;struct A{i32[2] a;}fn i32 main(){var a=A{.a=[1,2]};return a.a[{a.a[0]=9;0}];}`, 9, ""},
		{"store-current-root", `module main;struct A{i32[2][2] a;bool keep;}fn i32 main(){var a=A{.a=[[5,1],[2,3]],.keep=true};a.a[0][0]+={a=A{.a=[[2,9],[7,8]],.keep=false};3};assert(a.a[0][0]==8 && a.a[0][1]==9 && a.a[1][0]==7 && !a.keep);return 0;}`, 0, ""},
		{"row-current-root", `module main;fn i32 main(){var a=[[1,2],[3,4]];a[0]={a=[[5,6],[7,8]];[9,10]};assert(a[0][0]==9 && a[1][0]==7);return 0;}`, 0, ""},
		{"frozen-arguments", `module main;fn i32 first(i32[1][1] a,i32[1][1] b){return a[0][0]*10+b[0][0];}fn i32 main(){var a=[[1]];return first(a,{a=[[3]];a});}`, 13, ""},
		{"outer-bounds", `module main;fn i32 main(){var a=[[1]];a[1][{assert(false);0}]={assert(false);9};return 0;}`, 0, "bounds"},
		{"inner-bounds", `module main;fn i32 main(){var a=[[1]];a[0][1]={assert(false);9};return 0;}`, 0, "bounds"},
		{"bounds-large", `module main;fn i32 main(){let a=[[1]];return a[0][18446744073709551615];}`, 0, "bounds"},
		{"field-overflow", `module main;struct A{i64[1] a;}fn i32 main(){var a=A{.a=[9223372036854775807]};a.a[0]+=1;return 0;}`, 0, "integer-overflow"},
		{"index-return", `module main;fn i32 main(){var a=[[1]];a[{return 6;0}][0]=9;return 0;}`, 6, ""},
		{"rhs-return", `module main;fn i32 main(){var a=[[1]];a[0][0]={return 7;9};return 0;}`, 7, ""},
		{"temporary-copy", `module main;fn i32[1][1] make(){return [[9]];}fn i32 main(){return make()[0][0];}`, 9, ""},
	}
	fixtures := make([]map[string]any, 0, len(seeds))
	for _, seed := range seeds {
		outcome := map[string]any{"Kind": "return", "Value": seed.value}
		if seed.trap != "" {
			outcome = map[string]any{"Kind": "trap", "TrapCategory": seed.trap}
		}
		fixtures = append(fixtures, map[string]any{"name": seed.name, "inputHex": hex.EncodeToString([]byte(seed.input)), "admitted": true, "outcome": outcome, "outcomeOrigin": "fixed-fixture-contract"})
	}
	data, err := json.MarshalIndent(map[string]any{"schema": "fairyc.aggregate-array-fixed-oracle/v1", "fixtures": fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
