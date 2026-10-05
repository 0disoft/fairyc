module main;
struct Packet { u8[4] bytes; i64[3] totals; bool[2] flags; i32[2] codes; u32[2] words; usize[2] sizes; }
struct Envelope { Packet packet; }
fn i64 sum(ref Packet p) { var i64 total = 0; for (let i64 x in p.totals) { total += x; } return total; }
fn ref u8 first(ref Packet p) { for (let ref u8 x in p.bytes[1..]) { return x; } return ref p.bytes[0]; }
fn void edit(ref mut Envelope e) { for (let ref mut u8 x in e.packet.bytes[1..3]) { *x += 1; } }
fn void early(ref mut Packet p) { for (let ref mut u8 x in p.bytes) { *x += 1; return; } }
fn Result<i32,u32> fail() { return .err(7); }
fn Result<i32,u32> propagate(ref mut Packet p) { for (let ref mut u8 x in p.bytes) { *x += 1; let i32 unused = fail()?; } return .ok(0); }
fn i32 main() {
    var Packet p = Packet { .bytes = [1,2,3,4], .totals = [4000000000,2,3], .flags = [true,false], .codes = [-2,3], .words = [1,2], .sizes = [1,2] };
    let Packet original = p;
    assert(sum(ref p) == 4000000005);
    var usize calls = 0; var usize end = 3; var i64 total = 0;
    for (let i64 x in p.totals[{calls += 1; 1}..{calls += 10; end}]) { end = 1; total += x; }
    assert(calls == 11 && total == 5);
    for (let ref mut i64 x in p.totals[1..]) { *x += 1; continue; }
    assert(p.totals[1] == 3 && p.totals[2] == 4);
    for (let ref mut u8 x in ((p).bytes)) { *x += 1; break; }
    assert(p.bytes[0] == 2 && p.bytes[1] == 2);
    var usize count = 0;
    for (let ref u8 x in p.bytes[..]) { assert(*x > 0); count += 1; }
    for (let u8 x in p.bytes[4..]) { assert(false); }
    for (let ref mut u8 x in p.bytes[2..2]) { assert(false); }
    assert(count == 4);
    for (let ref mut bool x in p.flags) { *x = !*x; }
    for (let bool x in p.flags[1..]) { assert(x); }
    for (let ref mut i32 x in p.codes) { *x += 1; }
    for (let ref mut u32 x in p.words[..]) { *x += 1; }
    for (let ref mut usize x in p.sizes) { *x += 1; }
    assert(!p.flags[0] && p.flags[1] && p.codes[0] == -1 && p.words[1] == 3 && p.sizes[1] == 3);
    let ref u8 a = first(ref p); assert(*a == 2);
    early(ref mut p); assert(p.bytes[0] == 3);
    switch (propagate(ref mut p)) { case .err(code): { assert(code == 7); } case .ok(value): { assert(false); } };
    assert(p.bytes[0] == 4);
    var Envelope e = Envelope { .packet = p }; edit(ref mut e);
    assert(e.packet.bytes[1] == 3 && e.packet.bytes[2] == 4 && p.bytes[1] == 2);
    for (let u8 e in e.packet.bytes[..1]) { assert(e == 4); }
    assert(original.bytes[0] == 1 && original.totals[1] == 2);
    return 0;
}
