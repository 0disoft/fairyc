package main

// Independent Go semantic and interpreter outcomes for Fairy-generated code.
func writeMutableMIR(path string) error {
	seeds := []scalarMIRSeed{
		{"mutable-write", `fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;b[1]=9;assert(b[1]==9);assert(a[1]==9);return 0;}`, true},
		{"mutable-compound", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;b[0]+=2;assert(b[0]==3);assert(a[0]==3);return 0;}`, true},
		{"mutable-child", `fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;let s=b[1..];s[0]=8;b[0]=9;assert(a[0]==9 && a[1]==8);return 0;}`, true},
		{"mutable-nested", `fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;let s=b[1..];let t=s[..1];t[0]=8;s[1]=9;b[0]=7;assert(a[0]==7 && a[1]==8 && a[2]==9);return 0;}`, true},
		{"mutable-sequential", `fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let x=b[..1];x[0]=7;let y=b[1..];y[0]=8;assert(a[0]==7 && a[1]==8);return 0;}`, true},
		{"mutable-alias", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;let alias=b;alias[0]=7;b[0]=8;assert(a[0]==8);return 0;}`, true},
		{"mutable-call", `fn void fill(ref mut u8[] b){b[0]=8;}fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;fill(b);b[0]=9;assert(a[0]==9);return 0;}`, true},
		{"mutable-return", `fn ref mut u8[] tail(ref mut u8[] b){return b[1..];}fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let s=tail(b);s[0]=8;b[0]=9;assert(a[0]==9 && a[1]==8);return 0;}`, true},
		{"mutable-loop", `fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;var usize i=0;while(i<b.len){b[i]+=2;i+=1;}assert(a[0]==3 && a[1]==4 && a[2]==5);return 0;}`, true},
		{"mutable-before-borrow", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b={a[0]=2;ref mut a};b[0]=3;assert(a[0]==3);return 0;}`, true},
		{"mutable-length64", `fn i32 main(){var u8[64] a=[0;64];let ref mut u8[] b=ref mut a;b[63]=255;assert(b.len==64 && b[63]==255);assert(a[63]==255);return 0;}`, true},
		{"mutable-bound-read", `fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;let s=b[..b.len];s[0]=7;assert(a[0]==7);return 0;}`, true},
		{"mutable-bound-order", `fn i32 main(){var u8[3] a=[1,2,3];let ref mut u8[] b=ref mut a;var usize n=0;let s=b[n..{n=2;3}];assert(s.len==3);s[0]=7;assert(a[0]==7 && n==2);return 0;}`, true},
		{"mutable-empty", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;let s=b[1..];assert(s.is_empty);b[0]=8;assert(a[0]==8);return 0;}`, true},
		{"mutable-bounds-read", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;discard b[1];return 0;}`, true},
		{"mutable-bounds-write", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;b[1]=8;return 0;}`, true},
		{"mutable-check-before-rhs", `fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;b[1]={assert(false);8};return 0;}`, true},
		{"mutable-reverse", `fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;discard b[2..1];return 0;}`, true},
		{"mutable-compound-overflow", `fn i32 main(){var u8[1] a=[255];let ref mut u8[] b=ref mut a;b[0]+=1;return 0;}`, true},
		{"mutable-bound-return", `fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let s=b[0..{return 6;1}];return 9;}`, true},
	}
	return writeMIRSeeds(path, seeds, "fairyc.mutable-mir-oracle/v1")
}
