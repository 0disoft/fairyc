module main;
struct Packet { u8[4] bytes; i64[2] totals; bool[2] flags; usize length; }
struct Envelope { Packet packet; i32[2] codes; u32[2] words; usize[2] sizes; }
fn Packet copied(Packet p) {
    return p;
}
fn void replace(ref mut Packet p) { p.bytes = [8, 9, 10, 11]; p.totals = [5000000000, 6]; }
fn i32 main() {
    var Packet p = Packet { .bytes = [1, 2, 3, 4], .totals = [4000000000, 5], .flags = [true, false], .length = 4 };
    let Packet original = p;
    let Packet returned = copied(p);
    var Packet copy = p;
    copy.bytes = [7; 4];
    assert(p.bytes[0] == 1 && original.bytes[0] == 1 && returned.bytes[0] == 1 && copy.bytes[0] == 7);
    replace(ref mut p);
    assert(p.bytes[0] == 8 && p.totals[0] == 5000000000 && original.totals[0] == 4000000000);
    let u8[4] extracted = p.bytes;
    p.bytes = [0; 4]; assert(extracted[0] == 8 && p.bytes[0] == 0);
    var Envelope envelope = Envelope { .packet = p, .codes = [-1, 2], .words = [1, 2], .sizes = [3, 4] };
    envelope.packet.bytes = [6; 4]; envelope.codes = [8, 9];
    assert(envelope.packet.bytes[3] == 6 && p.bytes[3] == 0 && envelope.codes[0] == 8);
    assert(envelope.words[1] == 2 && envelope.sizes[1] == 4);
    let Option<Packet> saved = .some(original);
    switch(saved) { case .some(value): { assert(value.bytes[1] == 2); } case .none: { assert(false); } };
    var Packet[2] packets = [original; 2];
    packets[0] = p; assert(packets[0].bytes[0] == 0 && packets[1].bytes[0] == 1);
    assert(p.flags[0] && !p.flags[1] && p.bytes.len == 4);
    return 0;
}
