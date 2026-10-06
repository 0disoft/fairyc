"""Unit-17 whole-root transfer fixtures; flow rejection follows in 18..20."""
POSITIVES=[
 ('binding','resource R{cleanup{}}fn R relay(R x){let R y=x;return y;}',{}),
 ('assignment','resource R{cleanup{}}fn R relay(R x){var R y=R{};y=x;return y;}',{}),
 ('argument','resource R{cleanup{}}fn void take(R x){}fn void main(){let R r=R{};take(r);}',{}),
 ('return','resource R{cleanup{}}fn R relay(R x){return x;}',{}),
 ('parenthesized','resource R{cleanup{}}fn R relay(R x){return (x);}',{}),
 ('block-tail','resource R{cleanup{}}fn R relay(R x){return {x};}',{}),
 ('copy-field','resource R{i32 n;cleanup{}}fn R relay(R x){let i32 n=x.n;return x;}',{'n':(1,0)}),
 ('nested-field-read','resource R{i32 n;cleanup{}}resource Box{R child;cleanup{}}fn i32 read(Box b){return b.child.n;}',{'n':(1,0),'child':(0,1)}),
 ('resource-field-install','resource R{cleanup{}}resource Box{R child;cleanup{}}fn Box make(R r){return Box{.child=r};}',{'child':(0,1)}),
 ('option-root','resource R{cleanup{}}fn Option<R> relay(Option<R> x){return x;}',{}),
 ('result-root','resource R{cleanup{}}fn Result<R,i32> relay(Result<R,i32> x){return x;}',{}),
 ('capture','resource R{cleanup{}}fn Option<R> relay(Option<R> x){return switch(x){case .some(r):{.some(r)}case .none:{.none}};}',{}),
 ('propagation','resource R{cleanup{}}fn Result<R,i32> relay(Result<R,i32> x){let R r=x?;return .ok(r);}',{}),
 ('is-ok-consumes','resource R{cleanup{}}fn bool check(Result<R,i32> x){return x.is_ok;}',{}),
]
NEGATIVES=[
 ('field-return','resource R{cleanup{}}resource Box{R child;cleanup{}}fn R bad(Box b){return b.child;}','E1103'),
 ('field-binding','resource R{cleanup{}}resource Box{R child;cleanup{}}fn void bad(Box b){let R r=b.child;}','E1103'),
 ('field-argument','resource R{cleanup{}}resource Box{R child;cleanup{}}fn void take(R x){}fn void bad(Box b){take(b.child);}','E1103'),
 ('field-construction','resource R{cleanup{}}resource Box{R child;cleanup{}}fn Box bad(Box b){return Box{.child=b.child};}','E1103'),
 ('tagged-field','resource R{cleanup{}}resource Box{Option<R> child;cleanup{}}fn Option<R> bad(Box b){return b.child;}','E1103'),
]
