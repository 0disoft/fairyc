"""Tagged struct field fixtures; array storage remains a separate unit-14 part."""
POSITIVES=[
    ('option-field','struct Box{Option<i64> value;}fn i32 main(){var b=Box{.value=.some(4000000000)};let saved=b;b.value=.none;assert(switch(b.value){case .some(_):{false}case .none:{true}});return switch(saved.value){case .some(n):{assert(n==4000000000);7}case .none:{0}};}',7),
    ('result-field','struct Box{Result<i32,bool> value;i32 marker;}fn Box id(Box b){return b;}fn i32 main(){var b=Box{.marker=1,.value=.ok(7)};let saved=id(b);b.value=.err(true);assert(!b.value.is_ok);return switch(saved.value){case .ok(n):{n}case .err(_):{0}};}',7),
    ('enum-field','enum Status{Ready;Done;}struct Box{Status status;}fn i32 main(){var b=Box{.status=.Ready};let saved=b;b.status=.Done;assert(switch(b.status){case .Ready:{false}case .Done:{true}});return switch(saved.status){case .Ready:{7}case .Done:{0}};}',7),
    ('payload-enum-field','enum E{Data{i64 value;};Empty;}struct Box{E item;}fn i32 main(){let b=Box{.item=.Data{.value=4000000000}};return switch(b.item){case .Data{.value=n}:{assert(n==4000000000);7}case .Empty:{0}};}',7),
    ('nested-field','struct Inner{Option<i64> item;}struct Outer{Inner inner;bool ready;}fn i32 main(){var b=Outer{.ready=true,.inner=Inner{.item=.some(4000000000)}};var inner=b.inner;inner.item=.some(4000000002);b.inner=inner;return switch(b.inner.item){case .some(n):{assert(n==4000000002 && b.ready);7}case .none:{0}};}',7),
    ('field-rhs-sibling','struct Box{Result<i32,bool> value;i32 marker;}fn i32 main(){var b=Box{.value=.ok(1),.marker=2};b.value={b.marker=9;.ok(7)};assert(b.marker==9);return switch(b.value){case .ok(n):{n}case .err(_):{0}};}',7),
    ('struct-payload-field','struct Token{i64 value;}struct Problem{u32 code;}struct Box{Result<Token,Problem> item;}fn i32 main(){let b=Box{.item=.err(Problem{.code=8})};return switch(b.item){case .ok(_):{0}case .err(p):{assert(p.code==8);7}};}',7),
    ('field-propagation','struct Box{Result<i64,bool> item;}fn Result<i32,bool> read(Box b){let n=b.item?;assert(n==4000000000);return .ok(7);}fn i32 main(){assert(read(Box{.item=.ok(4000000000)}).is_ok);assert(!read(Box{.item=.err(true)}).is_ok);return 7;}',7),
]
deep='Option<'*12+'i32'+'>'*12
POSITIVES.append(('deep-tagged-field','struct Box{'+deep+' value;}fn i32 main(){let b=Box{.value=.none};discard b;return 7;}',7))
NEGATIVES=[
    ('nested-tagged-store','struct Inner{Option<i32> item;}struct Outer{Inner inner;}fn i32 main(){var b=Outer{.inner=Inner{.item=.some(1)}};b.inner.item=.none;return 0;}','assignment target is immutable'),
    ('nested-sibling-store','struct Inner{i32 item;}struct Outer{Inner inner;Option<i32> tag;}fn i32 main(){var b=Outer{.inner=Inner{.item=1},.tag=.none};b.inner.item=2;return 0;}','assignment target is immutable'),
    ('field-option-type','struct Box{Option<i64> item;}fn i32 main(){let b=Box{.item=.some(true)};return 0;}','type mismatch'),
    ('field-enum-identity','enum A{One;}enum B{One;}struct Box{A item;}fn i32 main(){let B other=.One;let b=Box{.item=other};return 0;}','type mismatch'),
    ('field-result-order','struct Box{Result<i32,bool> item;}fn i32 main(){let Result<bool,i32> other=.ok(true);let b=Box{.item=other};return 0;}','type mismatch'),
    ('field-result-unhandled','struct Box{Result<i32,bool> item;}fn i32 main(){let b=Box{.item=.ok(7)};b.item;return 0;}','Result value must be handled'),
    ('option-field-cycle','struct Node{Option<Node> next;}fn i32 main(){return 0;}','recursive value type'),
    ('result-field-cycle','struct Node{Result<i32,Node> next;}fn i32 main(){return 0;}','recursive value type'),
    ('indirect-field-cycle','struct A{Option<B> next;}struct B{A parent;}fn i32 main(){return 0;}','recursive value type'),
    ('missing-tagged-field','struct Box{Option<i32> item;i32 marker;}fn i32 main(){let b=Box{.item=.none};return 0;}','missing struct fields'),
]
TRAPS=[('field-payload-overflow','struct Box{Option<i32> value;}fn i32 main(){let b=Box{.value=.some(2147483647+1)};return switch(b.value){case .some(n):{n}case .none:{0}};}',1)]
