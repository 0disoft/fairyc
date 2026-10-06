"""Struct and diagnostic payload fixtures with independent fixed outcomes."""

PAIR='struct Pair{i64 value;bool ready;}'
DIAG='struct Diag{u32 code;usize start;usize end;}'
POSITIVES=[
    ('option-struct',PAIR+'fn i32 read(Option<Pair> r){return switch(r){case .some(p):{assert(p.value==4000000000 && p.ready);7}case .none:{2}};}fn i32 main(){assert(read(.some(Pair{.ready=true,.value=4000000000}))==7);assert(read(.none)==2);return 7;}',7),
    ('result-token-diagnostic',PAIR+DIAG+'fn i32 read(Result<Pair,Diag> r){return switch(r){case .ok(p):{assert(p.value==4000000000 && p.ready);7}case .err(d):{assert(d.code==8 && d.start==2 && d.end==9);3}};}fn i32 main(){assert(read(.ok(Pair{.value=4000000000,.ready=true}))==7);assert(read(.err(Diag{.end=9,.code=8,.start=2}))==3);return 7;}',7),
    ('struct-result-success-copy',PAIR+'fn Result<Pair,bool> identity(Result<Pair,bool> r){return r;}fn i32 main(){var p=Pair{.value=4000000000,.ready=true};let saved=identity(.ok(p));p.value=9;return switch(saved){case .ok(x):{assert(x.value==4000000000);7}case .err(_):{0}};}',7),
    ('struct-result-error-copy',DIAG+'fn Result<i32,Diag> identity(Result<i32,Diag> r){return r;}fn i32 main(){var d=Diag{.code=8,.start=2,.end=9};let saved=identity(.err(d));d.code=1;return switch(saved){case .ok(_):{0}case .err(x):{assert(x.code==8 && x.end==9);7}};}',7),
    ('struct-propagation',PAIR+DIAG+'fn Result<Pair,Diag> forward(Result<i64,Diag> r){let n=r?;return .ok(Pair{.value=n,.ready=true});}fn i32 main(){assert(forward(.ok(7)).is_ok);let r=forward(.err(Diag{.code=8,.start=2,.end=9}));return switch(r){case .ok(_):{0}case .err(d):{assert(d.code==8 && d.start==2 && d.end==9);7}};}',7),
    ('struct-extraction-propagation',PAIR+'fn Result<i32,bool> f(Result<Pair,bool> r){let p=r?;assert(p.value==4000000000 && p.ready);return .ok(7);}fn i32 main(){assert(f(.ok(Pair{.value=4000000000,.ready=true})).is_ok);assert(!f(.err(true)).is_ok);return 7;}',7),
    ('option-result-struct',PAIR+'fn i32 read(Option<Result<Pair,bool>> r){return switch(r){case .some(inner):{switch(inner){case .ok(p):{assert(p.value==4000000000);7}case .err(_):{2}}}case .none:{3}};}fn i32 main(){assert(read(.some(.ok(Pair{.value=4000000000,.ready=true})))==7);assert(read(.some(.err(true)))==2);assert(read(.none)==3);return 7;}',7),
    ('result-option-struct',PAIR+'fn i32 read(Result<Option<Pair>,bool> r){return switch(r){case .ok(o):{switch(o){case .some(p):{assert(p.value==4000000000);7}case .none:{2}}}case .err(_):{3}};}fn i32 main(){assert(read(.ok(.some(Pair{.value=4000000000,.ready=true})))==7);assert(read(.ok(.none))==2);assert(read(.err(true))==3);return 7;}',7),
    ('empty-struct-payload','struct Unit{}fn i32 read(Result<Unit,Unit> r){return switch(r){case .ok(_):{7}case .err(_):{2}};}fn i32 main(){assert(read(.ok(Unit{}))==7);assert(read(.err(Unit{}))==2);return 7;}',7),
    ('nested-struct-array','struct Data{i64 number;i32[3] values;}struct Packet{Data data;bool ready;}fn i32 main(){let Option<Packet> r=.some(Packet{.ready=true,.data=Data{.number=4000000000,.values=[1,2,3]}});return switch(r){case .some(p):{assert(p.data.number==4000000000 && p.ready && p.data.values[2]==3);7}case .none:{0}};}',7),
    ('struct-assignment',PAIR+DIAG+'fn i32 main(){var Result<Pair,Diag> r=.ok(Pair{.value=4000000000,.ready=true});let saved=r;r=.err(Diag{.code=8,.start=2,.end=9});assert(!r.is_ok);return switch(saved){case .ok(p):{assert(p.value==4000000000);7}case .err(_):{0}};}',7),
    ('captured-struct-mutation',PAIR+'fn i32 main(){let Option<Pair> r=.some(Pair{.value=4000000000,.ready=true});return switch(r){case .some(p):{var copy=p;copy.value+=2;assert(p.value==4000000000 && copy.value==4000000002);7}case .none:{0}};}',7),
    ('reverse-nominal-order','struct Diag{u32 code;}struct Pair{i64 value;}fn i32 main(){let Result<Pair,Diag> r=.ok(Pair{.value=4000000000});return switch(r){case .ok(p):{assert(p.value==4000000000);7}case .err(_):{0}};}',7),
    ('struct-field-order-effects','struct Pair{i32 x;i32 y;}fn i32 main(){var i32 count=0;let Option<Pair> r=.some(Pair{.y={count+=1;count},.x={count+=1;count}});return switch(r){case .some(p):{assert(p.x==2 && p.y==1 && count==2);7}case .none:{0}};}',7),
]
NEGATIVES=[
    ('same-shape-different-declaration','struct A{i32 n;}struct B{i32 n;}fn Option<A> f(){return .some(B{.n=7});}','type mismatch'),
    ('different-diagnostic-identity','struct A{u32 code;}struct B{u32 code;}fn Result<i32,A> f(Result<i32,B> r){return .ok(r?);}','identical error type'),
    ('wrong-struct-field',PAIR+'fn Option<Pair> f(){return .some(Pair{.value=true,.ready=true});}','type mismatch'),
    ('missing-struct-field',PAIR+'fn Option<Pair> f(){return .some(Pair{.value=7});}','missing struct fields'),
    ('unknown-struct','fn Option<Missing> f(){return .none;}','error:'),
    ('struct-result-unhandled',PAIR+'fn i32 main(){let Result<Pair,bool> r=.ok(Pair{.value=7,.ready=true});return 0;}','Result value must be handled'),
    ('struct-nested-result-unhandled',PAIR+'fn i32 f(Option<Result<Pair,bool>> r){return switch(r){case .some(inner):{7}case .none:{0}};}','Result value must be handled'),
    ('struct-wrong-result-order',PAIR+DIAG+'fn Result<Pair,Diag> f(Result<Diag,Pair> r){return r;}','type mismatch'),
    ('resource-payload-not-yet','resource R{cleanup{}}fn Option<R> f(){return .none;}','error:'),
    ('pointer-bearing-struct-not-yet','struct S{ref str text;}fn Option<S> f(){return .none;}','invalid nominal type'),
    ('recursive-struct','struct S{S child;}fn Option<S> f(){return .none;}','recursive value type'),
    ('nominal-encoding-bound',PAIR+'fn Result<Option<Pair>,Pair> f(){return .err(Pair{.value=7,.ready=true});}','error:'),
]
TRAPS=[('struct-payload-trap','struct Pair{i32 n;}fn i32 main(){let Option<Pair> r=.some(Pair{.n=2147483647+1});return switch(r){case .some(p):{p.n}case .none:{0}};}',1)]
