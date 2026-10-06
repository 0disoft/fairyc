"""Nested tagged source fixtures with fixed native outcomes; no compiler logic."""
from verify_results import SCALARS, VALUES

POSITIVES=[]
for t in SCALARS:
    value=VALUES[t]
    tests=[
        ('option-option',f'Option<Option<{t}>>',f'switch(r){{case .some(inner):{{switch(inner){{case .some(n):{{assert(n=={value});7}}case .none:{{2}}}}}}case .none:{{3}}}}',f'assert(read(.some(.some({value})))==7);assert(read(.some(.none))==2);assert(read(.none)==3);'),
        ('option-result',f'Option<Result<{t},bool>>',f'switch(r){{case .some(inner):{{switch(inner){{case .ok(n):{{assert(n=={value});7}}case .err(e):{{assert(e);2}}}}}}case .none:{{3}}}}',f'assert(read(.some(.ok({value})))==7);assert(read(.some(.err(true)))==2);assert(read(.none)==3);'),
        ('result-option',f'Result<Option<{t}>,bool>',f'switch(r){{case .ok(inner):{{switch(inner){{case .some(n):{{assert(n=={value});7}}case .none:{{2}}}}}}case .err(e):{{assert(e);3}}}}',f'assert(read(.ok(.some({value})))==7);assert(read(.ok(.none))==2);assert(read(.err(true))==3);'),
        ('result-result',f'Result<Result<{t},bool>,bool>',f'switch(r){{case .ok(inner):{{switch(inner){{case .ok(n):{{assert(n=={value});7}}case .err(e):{{assert(e);2}}}}}}case .err(e):{{assert(e);3}}}}',f'assert(read(.ok(.ok({value})))==7);assert(read(.ok(.err(true)))==2);assert(read(.err(true))==3);'),
        ('result-nested-error',f'Result<i32,Option<{t}>>',f'switch(r){{case .ok(n):{{n}}case .err(inner):{{switch(inner){{case .some(n):{{assert(n=={value});7}}case .none:{{2}}}}}}}}',f'assert(read(.ok(7))==7);assert(read(.err(.some({value})))==7);assert(read(.err(.none))==2);'),
    ]
    for name,typ,body,checks in tests:
        POSITIVES.append((name+'-'+t,f'fn i32 read({typ} r){{return {body};}}fn i32 main(){{{checks}return 7;}}',7))
POSITIVES += [
    ('propagate-nested-success','fn Result<i32,bool> unwrap(Result<Result<i32,bool>,bool> r){let inner=r?;return .ok(inner?);}fn i32 main(){assert(unwrap(.ok(.ok(7))).is_ok);assert(!unwrap(.ok(.err(true))).is_ok);assert(!unwrap(.err(true)).is_ok);return 7;}',7),
    ('propagate-nested-error','fn Result<bool,Option<i64>> unwrap(Result<i64,Option<i64>> r){let n=r?;return .ok(n>0);}fn i32 main(){assert(unwrap(.ok(7)).is_ok);let r=unwrap(.err(.some(4000000000)));return switch(r){case .ok(_):{0}case .err(inner):{switch(inner){case .some(n):{assert(n==4000000000);7}case .none:{0}}}};}',7),
    ('captured-result-discard','fn i32 read(Option<Result<i32,bool>> r){return switch(r){case .some(inner):{discard inner;7}case .none:{7}};}fn i32 main(){assert(read(.some(.err(true)))==7);return read(.none);}',7),
    ('nested-copy-order','fn i32 read(Option<Result<i32,bool>> r){return switch(r){case .some(inner):{switch(inner){case .ok(n):{n}case .err(_):{0}}}case .none:{0}};}fn i32 main(){var Option<Result<i32,bool>> r=.some(.ok(7));let saved=r;r=.some(.err(true));assert(read(r)==0);return read(saved);}',7),
    ('captured-result-before-break','fn i32 main(){while(true){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{discard inner;break;}case .none:{break;}};}return 7;}',7),
    ('both-nested-options','fn i32 read(Result<Option<i64>,Option<bool>> r){return switch(r){case .ok(o):{switch(o){case .some(n):{assert(n==4000000000);7}case .none:{2}}}case .err(o):{switch(o){case .some(e):{assert(e);3}case .none:{4}}}};}fn i32 main(){assert(read(.ok(.some(4000000000)))==7);assert(read(.ok(.none))==2);assert(read(.err(.some(true)))==3);assert(read(.err(.none))==4);return 7;}',7),
    ('both-nested-results','fn i32 read(Result<Result<i64,bool>,Result<bool,i32>> r){return switch(r){case .ok(inner):{switch(inner){case .ok(n):{assert(n==4000000000);7}case .err(e):{assert(e);2}}}case .err(inner):{switch(inner){case .ok(e):{assert(e);3}case .err(n):{n}}}};}fn i32 main(){assert(read(.ok(.ok(4000000000)))==7);assert(read(.ok(.err(true)))==2);assert(read(.err(.ok(true)))==3);assert(read(.err(.err(4)))==4);return 7;}',7),
    ('capture-inner-loop-handled','fn i32 main(){while(true){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{while(true){break;}discard inner;break;}case .none:{break;}};}return 7;}',7),
]
deep='Option<'*12+'i32'+'>'*12
POSITIVES.append(('bounded-depth',f'fn i32 main(){{let {deep} r=.none;discard r;return 7;}}',7))
NEGATIVES=[
    ('wrong-nested-payload','fn Result<Option<i64>,bool> f(){return .ok(.some(true));}','type mismatch'),
    ('wrong-nested-identity','fn Option<Result<i32,bool>> f(Option<Result<i64,bool>> r){return r;}','type mismatch'),
    ('nested-ordered-error','fn Result<Option<i32>,bool> f(Result<bool,Option<i32>> r){return r;}','type mismatch'),
    ('nested-incomplete','fn i32 f(Result<Option<i32>,bool> r){return switch(r){case .ok(o):{switch(o){case .some(n):{n}}}case .err(_):{0}};}','non-exhaustive Option switch'),
    ('nested-duplicate','fn i32 f(Option<Result<i32,bool>> r){return switch(r){case .none:{0}case .some(inner):{switch(inner){case .ok(n):{n}case .ok(n):{n}case .err(_):{0}}}};}','duplicate switch case'),
    ('capture-pending','fn i32 f(Option<Result<i32,bool>> r){return switch(r){case .some(inner):{7}case .none:{0}};}','Result value must be handled'),
    ('capture-before-break','fn i32 main(){while(true){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{break;}case .none:{break;}};}return 0;}','Result value must be handled'),
    ('capture-before-continue','fn i32 main(){while(true){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{continue;}case .none:{break;}};}return 0;}','Result value must be handled'),
    ('capture-branch-break','fn i32 f(bool b){while(b){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{if(b){break;}discard inner;}case .none:{break;}};}return 0;}','Result value must be handled'),
    ('capture-branch-continue','fn i32 f(bool b){while(b){let Option<Result<i32,bool>> r=.some(.ok(1));switch(r){case .some(inner):{if(b){continue;}discard inner;}case .none:{break;}};}return 0;}','Result value must be handled'),
    ('nested-propagate-error','fn Result<i32,Option<i32>> f(Result<i32,Option<i64>> r){return .ok(r?);}','identical error type'),
    ('nested-limit','fn '+('Option<'*14+'i32'+'>'*14)+' f(){return .none;}','error:'),
    ('nested-void','fn Result<Option<void>,bool> f(){return .ok(.none);}','error:'),
]
TRAPS=[]
