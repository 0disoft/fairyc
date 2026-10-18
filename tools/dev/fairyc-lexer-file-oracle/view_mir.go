package main

func writeViewMIR(path string) error {
	seeds := []scalarMIRSeed{
		{"view-block-before-borrow", `fn i32 main(){var u8[1] a=[1];let ref u8[] b={a[0]=2;ref a};assert(b[0]==2);return 0;}`, true},
		{"view-call-before-borrow", `fn ref u8[] same(i32 n,ref u8[] b){discard n;return b;}fn i32 main(){var u8[1] a=[1];let b=same({a[0]=2;0},ref a);assert(b[0]==2);return 0;}`, true},
		{"view-borrow", `fn i32 main(){let u8[3] a=[1,2,255];let ref u8[] b=ref a;assert(b.len==3 && b[0]==1 && b[2]==255);return 7;}`, true},
		{"view-length64", `fn i32 main(){let u8[64] a=[9;64];let ref u8[] b=ref a;assert(b[63]==9 && b.len==64);return 0;}`, true},
		{"view-slice", `fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;let s=b[1..3];assert(s.len==2 && s[0]==2 && s[1]==3);return 0;}`, true},
		{"view-omitted", `fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;assert(b[..].len==3 && b[..2].len==2 && b[1..].len==2 && b[3..].is_empty);return 0;}`, true},
		{"view-alias", `fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;let alias=b;let s=alias[1..][..1];assert(s[0]==2 && s.len==1);return 0;}`, true},
		{"view-siblings", `fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;let x=b[..2];let y=b[1..];assert(x[1]==2 && y[0]==2);return 0;}`, true},
		{"view-last-use", `fn i32 main(){var u8[1] a=[1];a[0]=2;let ref u8[] b=ref a;assert(b[0]==2);a[0]=3;return if(a[0]==3){0}else{1};}`, true},
		{"view-call", `fn u8 read(ref u8[] b,usize i){return b[i];}fn i32 main(){let u8[2] a=[4,5];assert(read(ref a,1)==5);return 0;}`, true},
		{"view-return", `fn ref u8[] tail(ref u8[] b){return b[1..];}fn ref u8[] forward(ref u8[] b){return tail(b);}fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;assert(forward(b)[0]==2);return 3;}`, true},
		{"view-array-parameter", `fn i32 sum(u8[2] a){let ref u8[] b=ref a;return if(b[0]==1 && b[1]==2){0}else{1};}fn i32 main(){let u8[2] a=[1,2];return sum(a);}`, true},
		{"view-loop", `fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;var usize i=0;while(i<b.len){assert(b[i]>0);i+=1;}return 0;}`, true},
		{"view-for", `fn i32 main(){let u8[3] a=[1,2,3];let ref u8[] b=ref a;var i32 total=0;for(var usize i=0;i<3;i+=1){if(b[i]>0){total+=1;}}return total;}`, true},
		{"view-static-slice", `fn i32 main(){let b=b"\0\xffA";assert(b[1..2][0]==255 && b[3..].is_empty);return 0;}`, true},
		{"view-frozen-owner", `fn i32 main(){var b=b"ABC";let s=b[{b=b"";0}..];assert(s[2]==67 && b.is_empty);return 0;}`, true},
		{"view-frozen-start", `fn i32 main(){var usize n=0;let s=b"ABC"[n..{n=2;3}];assert(s.len==3 && n==2);return 0;}`, true},
		{"view-start-return", `fn i32 main(){let u8[2] a=[1,2];let ref u8[] b=ref a;let s=b[{return 5;0}..];return 9;}`, true},
		{"view-end-return", `fn i32 main(){let u8[2] a=[1,2];let ref u8[] b=ref a;let s=b[0..{return 6;1}];return 9;}`, true},
		{"view-bounds", `fn i32 main(){let u8[1] a=[1];let ref u8[] b=ref a;discard b[1];return 0;}`, true},
		{"view-reverse", `fn i32 main(){let u8[2] a=[1,2];let ref u8[] b=ref a;discard b[2..1];return 0;}`, true},
		{"view-end-large", `fn i32 main(){let u8[1] a=[1];let ref u8[] b=ref a;discard b[0..18446744073709551615];return 0;}`, true},
		{"view-empty-index", `fn i32 main(){let u8[1] a=[1];let ref u8[] b=ref a;discard b[1..][0];return 0;}`, true},
		{"view-empty-static", `fn i32 main(){assert(b""[..].is_empty);return 0;}`, true},
	}
	return writeMIRSeeds(path, seeds, "fairyc.view-mir-oracle/v1")
}
