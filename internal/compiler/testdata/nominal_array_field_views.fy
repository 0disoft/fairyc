module main;
struct Page { Token[3] tokens; usize used; }
struct Envelope { Page page; }
struct Meta { u32 kind; }
struct Token { i64 value; Meta meta; }
fn void fill(ref mut Token[] data) { for (let ref mut Token t in data) { t.value += 1; t.meta.kind += 1; } }
fn i64 sum(ref Page p) { var i64 total = 0; for (let Token t in p.tokens) { total += t.value; } return total; }
fn ref Token first(ref Page p) { for (let ref Token t in p.tokens[1..]) { return t; } return ref p.tokens[0]; }
fn ref Token[] expose(ref Page p) { return ref p.tokens[1..]; }
fn ref mut Token[] writable(ref mut Page p) { return ref mut p.tokens[..2]; }
fn Option<ref Token[]> tagged(ref Page p) { return .some(ref p.tokens[1..]); }
fn void edit(ref mut Envelope e) { e.page.tokens[0].value += 10; fill(ref mut e.page.tokens[1..]); }
fn void early(ref mut Page p) { for (let ref mut Token t in p.tokens) { t.value += 1; return; } }
fn Result<i32,u32> fail() { return .err(7); }
fn Result<i32,u32> propagate(ref mut Page p) { for (let ref mut Token t in p.tokens) { t.value += 1; let i32 unused = fail()?; } return .ok(0); }
fn i32 main() {
    var Page p = Page { .tokens = [Token { .value = 4000000000, .meta = Meta { .kind = 1 } }; 3], .used = 3 };
    let Page original = p;
    let Token[3] copied = p.tokens;
    var usize calls = 0;
    p.tokens[{calls += 1; 0}].value += 1;
    p.tokens[1] = Token { .value = 2, .meta = Meta { .kind = 2 } };
    p.tokens[1].meta.kind += 1;
    assert(calls == 1 && p.tokens[0].value == 4000000001 && p.tokens[1].meta.kind == 3);
    assert(sum(ref p) == 8000000003);
    let ref Token[] all = ref p.tokens;
    let ref Token[] tail = ref p.tokens[1..];
    assert(all[0].value == 4000000001 && tail[0].value == 2 && p.used == 3);
    let ref mut Token slot = ref mut p.tokens[1]; slot.value += 3;
    let ref Token read = ref p.tokens[1]; assert(read.value == 5);
    let ref mut i64 leaf = ref mut p.tokens[1].value; *leaf += 1;
    let ref mut Token[] selected = ref mut ((p).tokens[{calls += 1; 1}..{calls += 10; 3}]);
    selected[0].value += 1; assert(selected[0].value == 7 && calls == 12);
    fill(ref mut p.tokens[..2]);
    let ref Token[] returned = expose(ref p); let ref Token[] alias = returned; assert(alias[0].value == 8);
    switch(tagged(ref p)) { case .some(value): { assert(value[0].value == 8); } case .none: { assert(false); } };
    let ref mut Token[] write = writable(ref mut p); write[0].meta.kind += 1;
    var usize end = 3; var i64 total = 0;
    for (let Token t in p.tokens[{calls += 1; 1}..{calls += 10; end}]) { end = 1; total += t.value; }
    assert(calls == 23 && total == 4000000008);
    for (let ref mut Token t in p.tokens[1..]) { t.value += 1; continue; }
    for (let ref mut Token t in ((p).tokens)) { t.value += 1; break; }
    var usize count = 0;
    for (let ref Token t in p.tokens[..]) { assert(t.value > 0); count += 1; }
    for (let Token t in p.tokens[3..]) { assert(false); }
    for (let ref mut Token t in p.tokens[2..2]) { assert(false); }
    let ref Token[] empty = ref p.tokens[3..]; assert(empty.is_empty && count == 3);
    let ref Token item = first(ref p); assert(item.value == 9);
    early(ref mut p);
    switch(propagate(ref mut p)) { case .err(code): { assert(code == 7); } case .ok(value): { assert(false); } };
    assert(p.tokens[0].value == 4000000005);
    var Envelope e = Envelope { .page = p }; edit(ref mut e);
    assert(e.page.tokens[0].value == 4000000015 && e.page.tokens[1].value == 10 && p.tokens[1].value == 9);
    assert(original.tokens[0].value == 4000000000 && copied[1].value == 4000000000);
    return 0;
}
