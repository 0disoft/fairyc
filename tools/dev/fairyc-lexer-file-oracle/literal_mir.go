package main

func writeLiteralMIR(path string) error {
	seeds := []scalarMIRSeed{
		{"literal-text", `fn i32 main(){let ref str s="한글\n\0\u{1f600}";assert(s.len==12 && !s.is_empty);return 0;}`, true},
		{"literal-empty", `fn i32 main(){assert("".is_empty && b"".is_empty && "".len==0 && b"".len==0);return 0;}`, true},
		{"literal-bytes", `fn i32 main(){let ref u8[] b=b"\xff\0\t\n\r\\\"";assert(b.len==7 && b[0]==255 && b[1]==0 && b[2]==9 && b[3]==10 && b[4]==13 && b[5]==92 && b[6]==34);return 7;}`, true},
		{"literal-utf8-bytes", `fn i32 main(){let b=b"한글😀";assert(b.len==10 && b[0]==237 && b[9]==128);return 0;}`, true},
		{"literal-return", `fn ref str make(){return "한글";}fn usize size(ref str text){return text.len;}fn i32 main(){assert(size(make())==6);return 3;}`, true},
		{"literal-byte-return", `fn ref u8[] make(){return b"\0\xff";}fn u8 read(ref u8[] b,usize i){return b[i];}fn i32 main(){assert(read(make(),1)==255);return 0;}`, true},
		{"literal-copy-reassign", `fn i32 main(){var b=b"AB";let saved=b;b=b"";assert(b.is_empty && saved[1]==66);return 0;}`, true},
		{"literal-index-owner", `fn i32 main(){var b=b"A";let u8 x=b[{b=b"";0}];assert(x==65 && b.is_empty);return 0;}`, true},
		{"literal-index-return", `fn i32 main(){let b=b"A";discard b[{return 5;0}];return 9;}`, true},
		{"literal-loop", `fn i32 main(){let b=b"ABC";var usize i=0;var i32 sum=0;while(i<b.len){if(b[i]>=65){sum+=1;}i+=1;}return sum;}`, true},
		{"literal-array-mixed", `fn i32 main(){var i32[1] a=[0];let b=b"A";if(b[0]==65){a[0]=7;}return a[0];}`, true},
		{"literal-branch", `fn i32 main(){let b=if(true){b"\xff"}else{b""};assert(b[0]==255);return 0;}`, true},
		{"literal-bounds", `fn i32 main(){discard b"A"[1];return 0;}`, true},
		{"literal-empty-bounds", `fn i32 main(){discard b""[0];return 0;}`, true},
		{"literal-large-bounds", `fn i32 main(){discard b"A"[18446744073709551615];return 0;}`, true},
		{"defer-trace-literal-bounds", `fn i32 defer_trace(i32 x){discard x;return 1;}fn void emit(i32 x){discard defer_trace(x);}fn i32 main(){defer emit(66);discard b""[{discard defer_trace(65);0}];return 0;}`, true},
	}
	return writeMIRSeeds(path, seeds, "fairyc.literal-mir-oracle/v1")
}
