package main

// Independent executable expectations come only from the Go compiler and
// interpreter. Candidate temporary IDs and operation counts are not goldens.
func writeNominalMIR(path string) error {
	seeds := []scalarMIRSeed{
		{"empty-construct", "struct E{}fn i32 main(){let e=E{};return 1;}", true},
		{"empty-return", "struct E{}fn E make(){return E{};}fn E id(E e){return e;}fn i32 accept(E e){return 3;}fn i32 main(){return accept(id(make()));}", true},
		{"empty-nested", "struct E{}struct B{E e;i32 n;}fn B id(B b){return b;}fn i32 main(){var b=id(B{.e=E{},.n=4});b.n=5;return b.n;}", true},
		{"empty-field-rhs-effects", "struct E{}struct B{E e;i32 n;}fn i32 main(){var b=B{.e=E{},.n=1};b.e={b.n=9;E{}};return b.n;}", true},
		{"empty-field-rhs-return", "struct E{}struct B{E e;}fn i32 main(){var b=B{.e=E{}};b.e={return 7;E{}};return 0;}", true},
		{"empty-nested-field", "struct E{}struct B{E e;i32 n;}struct C{B b;}fn i32 main(){var c=C{.b=B{.e=E{},.n=1}};c.b.e={c.b.n=8;E{}};return c.b.n;}", true},
		{"forward-layout", "struct B{A a;i32 n;}struct A{i32 x;}fn i32 main(){return B{.a=A{.x=7},.n=2}.a.x;}", true},
		{"construct-read", "struct A{i32 x;}fn i32 main(){return A{.x=17}.x;}", true},
		{"reordered-frozen", "struct P{i32 x;i32 y;}fn i32 main(){var i32 n=1;let p=P{.y=n,.x={n=3;n}};return p.x*10+p.y;}", true},
		{"initializer-return", "struct P{i32 x;i32 y;}fn i32 main(){let p=P{.y=1,.x={return 9;2}};return p.x;}", true},
		{"forward-type", "fn A id(A x){return x;}fn i32 main(){return id(A{.x=7}).x;}struct A{i32 x;}", true},
		{"nested-call-copy", "struct A{i64 x;bool ok;}struct B{A a;i32 z;}fn B id(B b){return b;}fn i32 main(){var b=B{.z=2,.a=A{.ok=true,.x=4000000000}};let c=id(b);b.a.x+=3;assert(c.a.x==4000000000);assert(b.a.x==4000000003);assert(c.a.ok);return c.z;}", true},
		{"whole-assignment", "struct A{i32 x;}fn i32 main(){var a=A{.x=1};a=A{.x=8};return a.x;}", true},
		{"nested-assignment", "struct A{i32 x;}struct B{A a;i32 z;}fn i32 main(){var b=B{.a=A{.x=1},.z=3};b.a=A{.x=8};return b.a.x+b.z;}", true},
		{"sibling-rhs", "struct A{i32 x;i32 y;}fn i32 main(){var a=A{.x=5,.y=1};a.x={a.y=9;3};return a.x*10+a.y;}", true},
		{"compound-frozen", "struct A{i32 x;i32 y;}fn i32 main(){var a=A{.x=5,.y=1};a.x+={a.x=2;a.y=9;3};return a.x*10+a.y;}", true},
		{"nested-rhs-owner", "struct A{i32 x;i32 y;}struct B{A a;i32 z;}fn i32 main(){var b=B{.a=A{.x=5,.y=1},.z=2};b.a.x+={b.a=A{.x=2,.y=9};b.z=7;3};return b.a.x*100+b.a.y*10+b.z;}", true},
		{"frozen-call-args", "struct A{i32 x;}fn i32 sum(A a,A b){return a.x*10+b.x;}fn i32 main(){var a=A{.x=1};return sum(a,{a=A{.x=3};a});}", true},
		{"struct-branch", "struct A{i32 x;}fn A choose(bool yes){return if(yes){A{.x=2}}else{A{.x=9}};}fn i32 main(){return choose(false).x;}", true},
		{"loop-update", "struct A{i32 x;}fn i32 main(){var a=A{.x=0};for(var i32 n=0;n<3;n+=1){a.x+=2;}return a.x;}", true},
		{"branch-update", "struct A{i32 x;}fn i32 f(bool c){var a=A{.x=1};if(c){a.x=3;}else{a.x=5;}return a.x;}fn i32 main(){return f(true)+f(false);}", true},
		{"distinct-identities", "struct A{i32 x;}struct B{i32 x;}fn i32 main(){let a=A{.x=1};let b=B{.x=2};return a.x+b.x;}", true},
		{"shorthand", "pub struct A{i32 x;bool b;}fn i32 main(){let i32 x=4;let bool b=true;let a=A{.b,.x};assert(a.b);return a.x;}", true},
		{"field-overflow", "struct A{i64 x;}fn i32 main(){var a=A{.x=9223372036854775807};a.x+=1;return 0;}", true},
		{"field-division", "struct A{i32 x;}fn i32 main(){var a=A{.x=7};a.x/=0;return 0;}", true},
		{"rhs-return", "struct A{i32 x;}fn i32 main(){var a=A{.x=1};a.x={return 6;3};return a.x;}", true},
		{"defer-trace-nominal", "struct A{i32 x;}fn i32 defer_trace(i32 x){discard x;return 1;}fn void emit(i32 x){discard defer_trace(x);}fn i32 main(){var a=A{.x=65};defer emit(a.x);a.x=66;defer emit(a.x);return 0;}", true},
	}
	return writeMIRSeeds(path, seeds, "fairyc.nominal-mir-oracle/v1")
}
