module main;
import std::vec as vec;

fn void append(ref mut vec::Vec<u8> values, u8 value) {
    assert(switch (vec::push<u8>(values, value)) { case .ok(_): { true } case .err(_): { false } });
}
fn ref u8[] tail(ref vec::Vec<u8> values) { return vec::view_u8(values)[1..]; }
fn Option<ref u8[]> shared_tagged(ref vec::Vec<u8> values) { return .some(tail(values)); }
fn ref mut u8[] middle(ref mut vec::Vec<u8> values) { return vec::view_mut_u8(values)[1..3]; }
fn Option<ref mut u8[]> tagged(ref mut vec::Vec<u8> values) { return .some(middle(values)); }
fn void read(ref u8[] values) { assert(values.len > 0); }
fn void reverse(ref mut u8[] values) {
    var usize left = 0;
    var usize right = values.len;
    while (left < right) {
        right -= 1;
        let u8 saved = values[left];
        values[left] = values[right];
        values[right] = saved;
        left += 1;
    }
}
fn i32 main() {
    var vec::Vec<u8> values = vec::empty<u8>();
    let ref u8[] empty = vec::view_u8(ref values);
    assert(empty.is_empty && empty[..].len == 0 && empty[0..0].is_empty);
    for (let u8 byte in empty) { assert(false); }
    let ref mut u8[] writable_empty = vec::view_mut_u8(ref mut values);
    assert(writable_empty.is_empty && writable_empty[..].len == 0);
    for (let u8 byte in writable_empty) { assert(false); }
    append(ref mut values, 0);
    let ref mut u8[] one = vec::view_mut_u8(ref mut values);
    for (let u8 byte in one) { assert(byte == 0); one[0] = 1; }
    append(ref mut values, 127);
    append(ref mut values, 128);
    append(ref mut values, 255);
    let ref u8[] shared = vec::view_u8(ref values);
    var i32 sum = 0;
    for (let u8 byte in shared) { sum += byte as i32; }
    assert(sum == 511 && shared[0] == 1 && shared[3] == 255);
    let ref u8[] returned = tail(ref values);
    assert(returned.len == 3 && returned[0] == 127);
    let Option<ref u8[]> shared_result = shared_tagged(ref values);
    switch (shared_result) { case .some(view): { assert(view.len == 3 && view[0] == 127); } case .none: { assert(false); } };
    let ref mut u8[] writable = vec::view_mut_u8(ref mut values);
    let ref mut u8[] child = writable[1..3];
    reverse(child);
    read(child);
    assert(writable[1] == 128 && writable[2] == 127);
    let ref mut u8[] nested = writable[1..][..1];
    nested[0] += 1;
    assert(writable[1] == 129);
    let Option<ref mut u8[]> result = tagged(ref mut values);
    switch (result) { case .some(view): { view[0] = 130; } case .none: { assert(false); } };
    let ref mut u8[] loop_view = vec::view_mut_u8(ref mut values);
    var usize index = 0;
    for (let u8 byte in loop_view) {
        if (index == 0) { loop_view[1] = 42; }
        if (index == 1) { assert(byte == 42); }
        index += 1;
    }
    assert(index == 4);
    let usize capacity = vec::capacity<u8>(ref values);
    assert(switch (vec::reserve<u8>(ref mut values, capacity + 16)) {
        case .ok(new_capacity): { new_capacity >= capacity + 16 }
        case .err(_): { false }
    });
    let ref mut u8[] grown = vec::view_mut_u8(ref mut values);
    grown[2] = 99;
    assert(grown[1] == 42 && grown[2] == 99 && grown[3] == 255);
    append(ref mut values, 7);
    assert(switch (vec::pop<u8>(ref mut values)) { case .some(byte): { byte == 7 } case .none: { false } });
    assert(vec::truncate<u8>(ref mut values, 2) == 2);
    assert(vec::clear<u8>(ref mut values) == 2);
    assert(vec::view_u8(ref values).is_empty);
    values = vec::empty<u8>();
    append(ref mut values, 55);
    let ref mut u8[] last = vec::view_mut_u8(ref mut values);
    last[0] = 56;
    let vec::Vec<u8> moved = values;
    assert(vec::view_u8(ref moved)[0] == 56);
    return 0;
}
