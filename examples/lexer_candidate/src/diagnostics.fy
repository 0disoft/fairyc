module diagnostics;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

pub resource Entry {
    u32 code; usize start; usize end; bool exit_repair; bytes::Buffer token;
    cleanup { }
}

pub struct Info { u32 code; usize start; usize end; bool exit_repair; }
pub fn Info info(ref Entry entry) {
    return Info { .code = entry.code, .start = entry.start, .end = entry.end, .exit_repair = entry.exit_repair };
}
pub fn ref u8[] token_view(ref Entry entry) { return bytes::view(ref entry.token); }

// Status 5 is reference-budget exhaustion, distinct from allocation/storage failure.
pub fn i32 add_limited(ref mut v::Vec<Entry> entries, usize count, usize limit, u32 code, usize start, usize end, bool exit_repair, ref u8[] token) {
    if (count >= limit) { return 5; }
    return add(entries, code, start, end, exit_repair, token);
}

// Bound sorting and retained diagnostic storage independently of input length.
pub fn i32 add(ref mut v::Vec<Entry> entries, u32 code, usize start, usize end, bool exit_repair, ref u8[] token) {
    var bool allocated = true;
    let bytes::Buffer owned = switch (bytes::copy(token)) {
        case .ok(value) : { value }
        case .err(_) : { allocated = false; bytes::empty() }
    };
    if (!allocated) { return 3; }
    let Entry entry = Entry { .code = code, .start = start, .end = end, .exit_repair = exit_repair, .token = owned };
    // Push returns the exact new length without converting a mutable borrow.
    // One overflow record is retained only until the failed scan drops its owner.
    return switch (v::push<Entry>(entries, entry)) {
        case .ok(length) : { if (length > 256 && !(code == 10 && length == 257)) { 4 } else { 0 } }
        case .err(_) : { 3 }
    };
}

fn bool less(ref Entry a, ref Entry b, usize ai, usize bi) {
    if (a.start != b.start) { return a.start < b.start; }
    if (a.end != b.end) { return a.end < b.end; }
    if (a.code != b.code) { return a.code < b.code; }
    if (a.exit_repair != b.exit_repair) { return a.exit_repair; }
    let ref u8[] at = bytes::view(ref a.token);
    let ref u8[] bt = bytes::view(ref b.token);
    var usize i = 0;
    while (i < at.len && i < bt.len) {
        if (at[i] != bt[i]) { return at[i] < bt[i]; }
        i += 1;
    }
    if (at.len != bt.len) { return at.len < bt.len; }
    return ai < bi;
}

@effects(console)
fn void number(usize value) {
    assert(switch (io::write_usize(value)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (io::write(" ")) { case .ok(_) : { true } case .err(_) : { false } });
}
@effects(console)
fn void text(ref u8[] value) {
    number(value.len);
    var usize i = 0;
    while (i < value.len) { number(value[i] as usize); i += 1; }
}

// Stable total-order selection needs no mutable vector view or copied resource.
@effects(console)
pub fn void report(ref v::Vec<Entry> entries) {
    let usize count = v::len<Entry>(entries);
    number(count);
    var usize emitted = 0;
    var usize previous = 0;
    while (emitted < count) {
        var usize best = count;
        var usize i = 0;
        while (i < count) {
            let ref Entry entry = v::at<Entry>(entries, i);
            if ((emitted == 0 || less(v::at<Entry>(entries, previous), entry, previous, i)) &&
                (best == count || less(entry, v::at<Entry>(entries, best), i, best))) { best = i; }
            i += 1;
        }
        assert(best < count);
        let ref Entry entry = v::at<Entry>(entries, best);
        if (entry.code == 1) { text(b"E0001"); }
        else if (entry.code == 4) { text(b"E0004"); }
        else if (entry.code == 5) { text(b"E0005"); }
        else if (entry.code == 10) { text(b"E0010"); }
        else { assert(entry.code == 6); text(b"E0006"); }
        if (entry.exit_repair) { text(b"exit"); } else { text(b"none"); }
        text(bytes::view(ref entry.token)); number(entry.start); number(entry.end);
        previous = best; emitted += 1;
    }
}
