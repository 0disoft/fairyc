package main

// Test-only independent semantic/MIR/interpreter expectations.
func writeEnumMIR(path string) error {
	seeds := []scalarMIRSeed{
		{"enum-value", "enum E{A;B;}fn i32 main(){let E e=.B;return switch(e){case .A:{3}case .B:{7}};}", true},
		{"enum-return", "enum E{A;B;}fn E id(E e){return e;}fn E make(){return .A;}fn i32 main(){return switch(id(make())){case .B:{2}case .A:{4}};}", true},
		{"enum-assignment", "enum E{A;B;}fn i32 main(){var E e=.A;let original=e;e=.B;return switch(original){case .A:{switch(e){case .A:{1}case .B:{8}}}case .B:{2}};}", true},
		{"enum-join", "enum E{A;B;}fn E choose(bool yes){return if(yes){.A}else{.B};}fn i32 main(){let E e=switch(choose(false)){case .A:{.B}case .B:{.A}};return switch(e){case .B:{9}case .A:{5}};}", true},
		{"enum-effects", "enum E{A;B;}fn i32 main(){var i32 n=1;var E e=.A;let i32 value=switch({n+=1;e}){case .B:{n+=100;8}case .A:{e=.B;n+=10;3}};return value+n;}", true},
		{"enum-early-return", "enum E{A;B;}fn i32 f(E e){switch(e){case .A:{return 11;}case .B:{return 12;}};return 99;}fn i32 main(){return f(.B);}", true},
		{"enum-loop", "enum E{A;B;}fn i32 main(){var E e=.A;var i32 n=0;while(n<3){e=switch(e){case .A:{.B}case .B:{.A}};n+=1;}return switch(e){case .A:{1}case .B:{6}};}", true},
		{"enum-separate", "enum E{A;B;}enum F{A;B;}struct P{i32 x;}fn i32 main(){let E e=.A;let F f=.B;let p=P{.x=2};return p.x+switch(e){case .A:{3}case .B:{4}}+switch(f){case .A:{5}case .B:{6}};}", true},
		{"enum-call-freeze", "enum E{A;B;}fn i32 f(E a,E b){return switch(a){case .A:{10}case .B:{20}}+switch(b){case .A:{1}case .B:{2}};}fn i32 main(){var E e=.A;return f(e,{e=.B;e});}", true},
		{"enum-unsigned-join", "enum E{A;B;}fn i32 main(){let E e=.B;let u8 n=switch(e){case .A:{1}case .B:{255}};assert(n==255);return 0;}", true},
		{"enum-defer", "enum E{A;B;}fn void check(bool ok){assert(ok);}fn i32 main(){let E e=.B;switch(e){case .A:{}case .B:{defer check(switch(e){case .A:{false}case .B:{true}});}};return 0;}", true},
	}
	return writeMIRSeeds(path, seeds, "fairyc.enum-mir-oracle/v1")
}
