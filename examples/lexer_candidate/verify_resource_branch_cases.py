"""Unit-19 branch equality, terminal paths, capture scope and short circuit."""
POSITIVES=[
 ('unchanged','resource R{i32 n;cleanup{}}fn R f(bool b,R r){if(b){let i32 n=r.n;}return r;}',{'n':(1,0)}),
 ('both-dead','resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R r){if(b){take(r);}else{take(r);}}',{}),
 ('early-return','resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R r){if(b){take(r);return R{};}return r;}',{}),
 ('both-return','resource R{cleanup{}}fn R f(bool b,R r){if(b){return r;}else{return r;}}',{}),
 ('restore-live','resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R input){var R r=input;if(b){take(r);r=R{};}return r;}',{}),
 ('dead-restored-both','resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R input){var R r=input;take(r);if(b){r=R{};}else{r=R{};}return r;}',{}),
 ('branch-local','resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R r){if(b){let R local=R{};take(local);}return r;}',{}),
 ('short-circuit-read','resource R{i32 n;cleanup{}}fn bool f(bool b,R r){return b || r.n==0;}',{'n':(1,0)}),
 ('switch-return','resource R{cleanup{}}fn R f(Option<R> x){return switch(x){case .some(r):{r}case .none:{R{}}};}',{}),
]
NEGATIVES=[
 ('one-dead-unused','resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R r){if(b){take(r);}}','E1104'),
 ('one-dead-use','resource R{cleanup{}}fn void take(R r){}fn R f(bool b,R r){if(b){take(r);}return r;}','E1104'),
 ('dead-revive-one','resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R input){var R r=input;take(r);if(b){r=R{};}}','E1104'),
 ('different-donors','resource R{cleanup{}}fn void take(R r){}fn void f(bool b,R x,R y){if(b){take(x);}else{take(y);}}','E1104'),
 ('switch-outer','resource R{cleanup{}}fn void take(R r){}fn void f(Option<i32> tag,R r){switch(tag){case .some(n):{take(r);}case .none:{}};}','E1104'),
 ('nested-branch','resource R{cleanup{}}fn void take(R r){}fn void f(bool a,bool b,R r){if(a){if(b){take(r);}}}','E1104'),
 ('short-circuit-and','resource R{cleanup{}}fn bool f(bool b,Result<R,i32> x){return b && x.is_ok;}','E1104'),
 ('short-circuit-or','resource R{cleanup{}}fn bool f(bool b,Result<R,i32> x){return b || x.is_ok;}','E1104'),
]
