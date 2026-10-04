module main;
import records as records;
import std::vec as vec;
import std::io as io;

@effects(console)
fn bool log_change(i64 before, i64 after) {
    return io::write("before=").is_ok && io::write_i64(before).is_ok
        && io::write(" after=").is_ok && io::write_i64(after).is_ok
        && io::write("\n").is_ok;
}

@effects(console)
fn i32 main() {
    var vec::Vec<records::Record> owner = vec::empty<records::Record>();
    let i64[3] inputs = [-5, 4000000000, 7];
    var usize offset = 0;
    for (let i64 value in inputs) {
        if (!vec::push<records::Record>(ref mut owner, records::Record {
            .value = value, .position = records::Span { .start = offset, .end = offset + 1 }
        }).is_ok) { return 1; }
        offset += 1;
    }
    let ref mut records::Record[] items = vec::view_mut<records::Record>(ref mut owner);
    var usize index = 0;
    var Option<records::Record> last = .none;
    for (let records::Record before in items) {
        // Each iteration copies the current record before any body writes.
        items[index].value = records::clamp(before.value);
        var records::Record preview = before;
        preview.position.start = 99;
        assert(before.position.start == index && items[index].position.start == index);
        if (!log_change(before.value, items[index].value)) { return 2; }
        last = .some(before);
        index += 1;
    }
    assert(index == 3);
    assert(vec::clear<records::Record>(ref mut owner) == 3);
    // This is a retained value copy and survives clearing the original owner.
    switch (last) {
        case .some(record): { assert(record.value == 7 && record.position.start == 2); }
        case .none: { return 3; }
    };
    return 0;
}
