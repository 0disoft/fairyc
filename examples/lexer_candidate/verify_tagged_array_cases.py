"""Composite array cases; Fairy owns typing, layouts and compilation."""
POSITIVES=[
    ('option-array','fn i32 main(){var Option<i64>[2] a=[.some(4000000000),.none];let saved=a;a[0]=.none;return switch(saved[0]){case .some(n):{assert(n==4000000000);7}case .none:{0}};}',7),
    ('result-array','fn i32 main(){var Result<i32,bool>[2] a=[.ok(7),.err(true)];a[1]=.ok(8);return switch(a[0]){case .ok(n):{n}case .err(_):{0}};}',7),
    ('result-repeat','fn i32 main(){let Result<i32,bool>[3] a=[.ok(7);3];assert(a.len==3);return switch(a[2]){case .ok(n):{n}case .err(_):{0}};}',7),
    ('enum-array','enum E{One;Two;}fn E[2] id(E[2] a){return a;}fn i32 main(){var E[2] a=[.One,.Two];let saved=id(a);a[0]=.Two;return switch(saved[0]){case .One:{7}case .Two:{0}};}',7),
    ('payload-enum-array','enum E{Data{i64 value;};Empty;}fn i32 main(){let E[2] a=[.Empty,.Data{.value=4000000000}];return switch(a[1]){case .Data{.value=n}:{assert(n==4000000000);7}case .Empty:{0}};}',7),
    ('nominal-array','struct T{i64 n;}fn i32 main(){var a=[T{.n=4000000000},T{.n=7}];a[0]=T{.n=8};assert(a[0].n==8);return if(a[1].n==7){7}else{0};}',7),
    ('struct-payload-array','struct T{i64 n;}struct P{u32 code;}fn i32 main(){let Result<T,P>[2] a=[.ok(T{.n=4000000000}),.err(P{.code=8})];assert(!a[1].is_ok);return switch(a[0]){case .ok(t):{assert(t.n==4000000000);7}case .err(_):{0}};}',7),
    ('nested-tag-array','fn i32 main(){let Option<Option<i32>>[2] a=[.some(.some(7)),.none];return switch(a[0]){case .some(inner):{switch(inner){case .some(n):{n}case .none:{0}}}case .none:{0}};}',7),
    ('array-field','struct Box{Option<i32>[2] a;i32 marker;}fn i32 main(){var b=Box{.a=[.some(7),.none],.marker=1};let saved=b;b.a=[.none,.none];return switch(saved.a[0]){case .some(n):{n}case .none:{0}};}',7),
    ('rank-two-tagged','fn i32 main(){var Option<i32>[2][2] a=[[.some(7),.none],[.none,.none]];let saved=a;a[0][0]=.none;return switch(saved[0][0]){case .some(n):{n}case .none:{0}};}',7),
    ('maximum-count','fn i32 main(){let Option<i32>[64] a=[.some(7);64];assert(a.len==64);return switch(a[63]){case .some(n):{n}case .none:{0}};}',7),
    ('duplicate-type','fn Option<i32>[2] id(Option<i32>[2] a){return a;}fn i32 main(){let Option<i32>[2] a=[.some(7),.none];let b=id(a);return switch(b[0]){case .some(n):{n}case .none:{0}};}',7),
    ('duplicate-nested-type','fn Option<Option<i32>>[2] id(Option<Option<i32>>[2] a){return a;}fn i32 main(){let Option<Option<i32>>[2] a=[.some(.some(7)),.none];let b=id(a);return switch(b[0]){case .some(inner):{switch(inner){case .some(n):{n}case .none:{0}}}case .none:{0}};}',7),
]
NEGATIVES=[
    ('wrong-option-element','fn i32 main(){let Option<i32>[2] a=[.some(true),.none];return 0;}','type mismatch'),
    ('wrong-result-order','fn i32 main(){let Result<bool,i32> r=.ok(true);let Result<i32,bool>[1] a=[r];return 0;}','type mismatch'),
    ('enum-identity','enum A{One;}enum B{One;}fn i32 main(){let B b=.One;let A[1] a=[b];return 0;}','type mismatch'),
    ('array-length','fn i32 main(){let Option<i32>[2] a=[.none];return 0;}','type mismatch'),
    ('zero-count','fn i32 main(){let Option<i32>[0] a=[.none;0];return 0;}','unsupported'),
    ('count-limit','fn i32 main(){let Option<i32>[65] a=[.none;65];return 0;}','unsupported'),
    ('immutable-array','fn i32 main(){let Option<i32>[1] a=[.none];a[0]=.some(1);return 0;}','assignment target is immutable'),
    ('unhandled-result-index','fn i32 main(){let Result<i32,bool>[1] a=[.ok(7)];a[0];return 0;}','Result value must be handled'),
    ('array-value-cycle','struct Node{Option<Node>[1] children;}fn i32 main(){return 0;}','recursive value type'),
    ('rank-limit','fn i32 main(){let Option<i32>[1][1][1] a=[[[.none]]];return 0;}','unsupported'),
]
TRAPS=[
    ('bounds-first','fn i32 main(){let Option<i32>[1] a=[.some(7)];return switch(a[1]){case .some(n):{n}case .none:{0}};}',6),
    ('payload-overflow','fn i32 main(){let Option<i32>[1] a=[.some(2147483647+1)];return 0;}',1),
]
