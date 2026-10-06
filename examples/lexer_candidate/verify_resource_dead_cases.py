"""Unit-18 ownership liveness; later units gate joins and loop edges explicitly."""
POSITIVES=[
 ('move-once','resource R{cleanup{}}fn R relay(R r){let R next=r;return next;}',{}),
 ('replace-dead','resource R{cleanup{}}fn void take(R r){}fn R main(){var R r=R{};take(r);r=R{};return r;}',{}),
 ('self-reinstall','resource R{cleanup{}}fn R main(){var R r=R{};r=r;return r;}',{}),
 ('copy-field-repeated','resource R{i32 n;cleanup{}}fn i32 read(R r){let i32 n=r.n;return n+r.n;}',{'n':(1,0)}),
 ('distinct-donors','resource R{cleanup{}}fn void take(R a,R b){}fn void main(){let R a=R{};let R b=R{};take(a,b);}',{}),
 ('shadow','resource R{cleanup{}}fn void take(R r){}fn R main(){let R r=R{};{let R r=R{};take(r);};return r;}',{}),
 ('unreachable','resource R{cleanup{}}fn R relay(R r){return r;r;}',{}),
 ('propagate-once','resource R{cleanup{}}fn Result<R,i32> relay(Result<R,i32> x){let R r=x?;return .ok(r);}',{}),
 ('captured-once','resource R{cleanup{}}fn Option<R> relay(Option<R> x){return switch(x){case .some(r):{.some(r)}case .none:{.none}};}',{}),
]
NEGATIVES=[
 ('second-argument','resource R{cleanup{}}fn void take(R a,R b){}fn void main(){let R r=R{};take(r,r);}','E1101'),
 ('second-binding','resource R{cleanup{}}fn void main(){let R r=R{};let R next=r;let R again=r;}','E1101'),
 ('second-return','resource R{cleanup{}}fn void take(R r){}fn R main(){let R r=R{};take(r);return r;}','E1101'),
 ('field-after-move','resource R{i32 n;cleanup{}}fn void take(R r){}fn i32 main(){let R r=R{.n=7};take(r);return r.n;}','E1101'),
 ('rhs-after-move','resource R{cleanup{}}fn void take(R r){}fn void main(){var R r=R{};take(r);r=r;}','E1101'),
 ('record-after-move','resource R{cleanup{}}resource Box{R child;cleanup{}}fn Box bad(R r){let R next=r;return Box{.child=r};}','E1101'),
 ('is-ok-after-move','resource R{cleanup{}}fn bool bad(Result<R,i32> x){let bool ok=x.is_ok;return x.is_ok;}','E1101'),
 ('propagated-owner','resource R{cleanup{}}fn Result<R,i32> bad(Result<R,i32> x){let R r=x?;return x;}','E1101'),
 ('moved-capture','resource R{cleanup{}}fn void take(R r){}fn Option<R> bad(Option<R> x){return switch(x){case .some(r):{take(r);.some(r)}case .none:{.none}};}','E1101'),
]
