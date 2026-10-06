"""Unit-16 source fixtures; factories are ordinary typed calls, never constructors."""
POSITIVES=[
 ('empty','resource R{cleanup{}}fn R make(){return R{};}fn R main(){return make();}',{}),
 ('large','resource R{i64 n;cleanup{}}fn R from_amount(i64 n){return R{.n=n};}fn R main(){return from_amount(4000000000);}',{'n':(1,0)}),
 ('written-order','resource R{i64 x;bool y;cleanup{}}fn bool second(){return true;}fn i64 first(){return 9;}fn R make(){return R{.y=second(),.x=first()};}',{'x':(1,0),'y':(1,0)}),
 ('shorthand','resource R{i64 n;cleanup{}}fn R make(i64 n){return R{.n};}',{'n':(1,0)}),
 ('inferred-binding','resource R{i32 n;cleanup{}}fn R make(){return R{.n=7};}fn i32 main(){let r=make();return r.n;}',{'n':(1,0)}),
 ('nested','resource R{cleanup{}}resource Box{R child;cleanup{}}fn R leaf(){return R{};}fn Box make(){return Box{.child=leaf()};}',{'child':(0,1)}),
 ('option','resource R{i32 n;cleanup{}}fn Option<R> make(bool ok){return if(ok){.some(R{.n=7})}else{.none};}',{'n':(1,0)}),
 ('fallible','resource R{i64 n;cleanup{}}pub fn Result<R,i32> checked(bool ok){if(!ok){return .err(3);}return .ok(R{.n=4000000000});}',{'n':(1,0)}),
 ('factory-propagation','resource R{cleanup{}}fn Result<R,i32> make(){return .ok(R{});}fn Result<R,i32> forward(){let R r=make()?;return .ok(r);}',{}),
 ('enum-array-fields','enum E{One;Two;}resource R{E kind;i32[2] values;cleanup{}}fn R make(){return R{.values=[1,2],.kind=.One};}',{'kind':(1,0),'values':(1,0)}),
 ('nested-tag-fields','resource R{cleanup{}}resource Box{Option<R> child;Result<R,i32> outcome;cleanup{}}fn Box make(){return Box{.child=.some(R{}),.outcome=.ok(R{})};}',{'child':(0,1),'outcome':(0,1)}),
 ('public-factory','pub resource R{cleanup{}}pub fn R create(){return R{};}fn R main(){return create();}',{}),
]
NEGATIVES=[
 ('missing-field','resource R{i32 n;bool ready;cleanup{}}fn R make(){return R{.n=1};}','missing struct fields'),
 ('unknown-field','resource R{i32 n;cleanup{}}fn R make(){return R{.n=1,.extra=2};}','unknown field'),
 ('duplicate-field','resource R{i32 n;cleanup{}}fn R make(){return R{.n=1,.n=2};}','duplicate field'),
 ('wrong-field-type','resource R{i64 n;cleanup{}}fn R make(){return R{.n=true};}','type mismatch'),
 ('wrong-resource-field','resource A{cleanup{}}resource B{cleanup{}}resource Box{A child;cleanup{}}fn Box make(){return Box{.child=B{}};}','type mismatch'),
 ('wrong-factory-result','resource A{cleanup{}}resource B{cleanup{}}fn A make(){return B{};}','type mismatch'),
 ('wrong-factory-argument','resource R{cleanup{}}fn R make(i32 n){return R{};}fn R main(){return make(true);}','type mismatch'),
 ('wrong-factory-arity','resource R{cleanup{}}fn R make(i32 n){return R{};}fn R main(){return make();}','call argument count mismatch'),
 ('not-constructor-call','resource R{cleanup{}}fn R main(){return R();}','expression is not callable'),
 ('wrong-error-payload','resource R{cleanup{}}fn Result<R,i32> make(){return .err(true);}','type mismatch'),
 ('nonempty-cleanup','resource R{cleanup{assert(true);}}fn R make(){return R{};}','unsupported'),
 ('resource-array','resource R{cleanup{}}fn i32 main(){let a=[R{}];return 0;}','type mismatch'),
]
