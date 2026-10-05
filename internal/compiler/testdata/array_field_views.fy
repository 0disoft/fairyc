module main;
struct Packet { u8[4] bytes; i64[3] totals; bool[2] flags; i32[2] codes; u32[2] words; usize[2] sizes; usize length; }
struct Envelope { Packet packet; }
fn void fill(ref mut u8[] bytes) { for (let ref mut u8 byte in bytes) { *byte += 1; } }
fn i64 sum(ref i64[] data) { var i64 total = 0; for (let i64 x in data) { total += x; } return total; }
fn ref u8[] expose(ref Packet p) { return ref p.bytes[1..]; }
fn ref mut u8[] writable(ref mut Packet p) { return ref mut p.bytes[..2]; }
fn Option<ref u8[]> tagged(ref Packet p) { return .some(ref p.bytes[1..]); }
fn void edit(ref mut Envelope e) {
    e.packet.bytes[0] = 9;
    e.packet.totals[1] += 10;
    fill(ref mut e.packet.bytes[1..3]);
}
fn i32 main() {
    var Packet p = Packet { .bytes = [1, 2, 3, 4], .totals = [4000000000, 2, 3], .flags = [true, false], .codes = [1, 2], .words = [1, 2], .sizes = [1, 2], .length = 4 };
    let Packet original = p;
    var usize calls = 0;
    p.bytes[{calls += 1; 0}] += 10;
    assert(calls == 1 && p.bytes[0] == 11 && original.bytes[0] == 1);
    let ref u8[] shared = ref p.bytes;
    let ref u8[] middle = ref p.bytes[1..3];
    assert(shared[0] == 11 && middle[0] == 2 && p.length == 4);
    let ref i64[] totals = ref p.totals;
    assert(sum(totals) == 4000000005 && sum(ref p.totals[1..]) == 5);
    let ref mut u8[] selected = ref mut ((p).bytes[{calls += 1; 1}..{calls += 1; 3}]);
    selected[0] = 20; assert(selected[0] == 20 && calls == 3);
    fill(ref mut p.bytes[..2]); assert(p.bytes[0] == 12 && p.bytes[1] == 21);
    let ref mut i64 number = ref mut p.totals[0]; *number += 1;
    let ref i64 read = ref p.totals[0]; assert(*read == 4000000001);
    p.flags[1] = true; p.codes[0] += 3; p.words[0] += 4; p.sizes[0] += 5;
    let ref mut bool flag = ref mut p.flags[0]; *flag = false;
    assert(!p.flags[0] && p.flags[1] && p.codes[0] == 4 && p.words[0] == 5 && p.sizes[0] == 6);
    let ref u8[] returned = expose(ref p); let ref u8[] alias = returned; assert(alias[0] == 21);
    switch(tagged(ref p)) { case .some(value): { assert(value[0] == 21); } case .none: { assert(false); } };
    let ref mut u8[] writable_view = writable(ref mut p); writable_view[0] += 2;
    let ref u8[] empty = ref p.bytes[4..]; assert(empty.is_empty);
    var Envelope e = Envelope { .packet = p }; edit(ref mut e);
    assert(e.packet.bytes[0] == 9 && e.packet.bytes[1] == 22 && p.bytes[1] == 21);
    assert(e.packet.totals[1] == 12 && p.totals[1] == 2);
    assert(original.bytes[0] == 1 && original.totals[0] == 4000000000);
    return 0;
}
