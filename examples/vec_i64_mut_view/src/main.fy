module main;
import std::vec as vec;

fn void append(ref mut vec::Vec<i64> values, i64 value) {
    assert(switch (vec::push<i64>(values, value)) { case .ok(_) : { true } case .err(_) : { false } });
}
fn void reverse(ref mut i64[] values) {
    var usize left = 0;
    var usize right = values.len;
    while (left < right) {
        right -= 1;
        let i64 saved = values[left];
        values[left] = values[right];
        values[right] = saved;
        left += 1;
    }
}
fn void read(ref i64[] values) { assert(values.len > 0); }
fn ref mut i64[] middle(ref mut vec::Vec<i64> values) { return vec::view_mut_i64(values)[1..4]; }
fn Option<ref mut i64[]> tagged(ref mut vec::Vec<i64> values) { return .some(middle(values)); }
fn i32 main() {
    var vec::Vec<i64> values = vec::empty<i64>();
    let ref mut i64[] empty = vec::view_mut_i64(ref mut values);
    assert(empty.is_empty && empty[..].len == 0 && empty[0..0].is_empty);
    append(ref mut values, 9223372036854775807);
    append(ref mut values, 4000000000);
    append(ref mut values, -7);
    append(ref mut values, -9223372036854775808);
    append(ref mut values, 9);

    let ref mut i64[] all = vec::view_mut_i64(ref mut values);
    let ref mut i64[] child = all[1..4];
    reverse(child);
    read(child);
    assert(all[0] == 9223372036854775807 && all[1] == -9223372036854775808);
    assert(all[2] == -7 && all[3] == 4000000000 && all[4] == 9);
    let ref mut i64[] nested = all[1..][1..2];
    nested[0] -= 1;
    assert(all[2] == -8);
    let ref mut i64[] returned = middle(ref mut values);
    returned[1] = 42;
    let Option<ref mut i64[]> result = tagged(ref mut values);
    switch (result) { case .some(view) : { view[1] += 1; } case .none : { assert(false); } };
    assert(vec::view_i64(ref values)[2] == 43);

    let usize previous_capacity = vec::capacity<i64>(ref values);
    assert(switch (vec::reserve<i64>(ref mut values, previous_capacity + 16)) {
        case .ok(capacity) : { capacity >= previous_capacity + 16 }
        case .err(_) : { false }
    });
    let ref mut i64[] grown = vec::view_mut_i64(ref mut values);
    grown[4] = 123;
    assert(grown[1] == -9223372036854775808 && grown[4] == 123);
    append(ref mut values, 99);
    assert(switch (vec::pop<i64>(ref mut values)) { case .some(value) : { value == 99 } case .none : { false } });
    assert(vec::truncate<i64>(ref mut values, 3) == 2);
    assert(vec::clear<i64>(ref mut values) == 3);
    assert(vec::view_mut_i64(ref mut values)[..].is_empty);
    values = vec::empty<i64>();
    append(ref mut values, 55);
    let ref mut i64[] last = vec::view_mut_i64(ref mut values);
    last[0] = 56;
    let vec::Vec<i64> moved = values;
    assert(vec::view_i64(ref moved)[0] == 56);
    return 0;
}
