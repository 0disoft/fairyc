package main

func writeArrayMIR(path string) error {
	seeds := []scalarMIRSeed{
		{"array-copy", "fn i32 main(){var i32[3] a=[1,2,3];let b=a;a[0]=9;assert(b[0]==1 && a[0]==9);return b[2];}", true},
		{"array-return", "fn i64[2] make(){return [4000000000,7];}fn i64[2] id(i64[2] a){return a;}fn i32 main(){let a=id(make());assert(a[0]==4000000000 && a.len==2);return 7;}", true},
		{"array-bool", "fn i32 main(){var bool[2] a=[false,true];a[0]=true;assert(a[0] && a[1]);return 1;}", true},
		{"array-unsigned", "fn i32 main(){let u8[2] a=[0,255];let u32[1] b=[4294967295];let usize[1] c=[18446744073709551615];assert(a[1]==255 && b[0]==4294967295 && c[0]==18446744073709551615);return 0;}", true},
		{"array-repeat-once", "fn i32 main(){var i32 n=0;let i32[64] a=[{n+=1;n};64];assert(n==1 && a[0]==1 && a[63]==1 && a.len==64);return n;}", true},
		{"array-initializer-order", "fn i32 main(){var i32 n=1;let a=[n,{n=3;n},n];assert(a[0]==1 && a[1]==3 && a[2]==3);return 0;}", true},
		{"array-store-order", "fn i32 main(){var i32[2] a=[5,1];var usize i=0;a[{i+=1;0}]+={a[0]=2;a[1]=9;3};assert(i==1 && a[0]==8 && a[1]==9);return 0;}", true},
		{"array-read-index-effects", "fn i32 main(){var i32[2] a=[1,2];return a[{a[0]=9;0}];}", true},
		{"array-rhs-root", "fn i32 main(){var i32[2] a=[1,2];a[0]={a=[3,4];9};assert(a[0]==9 && a[1]==4);return 0;}", true},
		{"array-rhs-return", "fn i32 main(){var i32[1] a=[1];a[0]={return 6;9};return a[0];}", true},
		{"array-initializer-return", "fn i32 main(){let i32[2] a=[1,{return 5;2}];return 9;}", true},
		{"array-frozen-arguments", "fn i32 first(i32[2] a,i32[2] b){return a[0]*10+b[0];}fn i32 main(){var a=[1,2];return first(a,{a=[3,4];a});}", true},
		{"array-loop", "fn i32 main(){var i32[3] a=[0;3];var usize i=0;while(i<3){a[i]+=2;i+=1;}return a[0]+a[1]+a[2];}", true},
		{"array-bounds-read", "fn i32 main(){let i32[1] a=[1];let usize i=1;return a[i];}", true},
		{"array-bounds-large", "fn i32 main(){let i32[1] a=[1];let usize i=18446744073709551615;return a[i];}", true},
		{"array-overflow", "fn i32 main(){var i64[1] a=[9223372036854775807];a[0]+=1;return 0;}", true},
		{"defer-trace-array-bounds-store", "fn i32 defer_trace(i32 x){discard x;return 1;}fn void emit(i32 x){discard defer_trace(x);}fn i32 main(){var i32[1] a=[1];defer emit(66);a[1]={discard defer_trace(65);9};return 0;}", true},
	}
	return writeMIRSeeds(path, seeds, "fairyc.array-mir-oracle/v1")
}
