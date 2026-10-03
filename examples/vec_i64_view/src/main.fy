module main;
import std::vec as vec;

fn void append(ref mut vec::Vec<i64> values, i64 value) {
    assert(switch (vec::push<i64>(values, value)) {
        case .ok(_) : { true }
        case .err(_) : { false }
    });
}

fn i64 sum(ref i64[] values) {
    var i64 result = 0;
    let usize count = values.len;
    for (var usize index = 0; index < count; index += 1) {
        result += values[index];
    }
    return result;
}

fn ref i64[] middle(ref vec::Vec<i64> values) {
    return vec::view_i64(values)[1..3];
}

fn Option<ref i64[]> first_two(ref vec::Vec<i64> values) {
    return .some(vec::view_i64(values)[..2]);
}

fn i32 main() {
    var vec::Vec<i64> values = vec::empty<i64>();
    let ref i64[] empty = vec::view_i64(ref values);
    assert(empty.is_empty && empty[..].len == 0 && empty[0..0].is_empty);

    append(ref mut values, -9223372036854775808);
    append(ref mut values, 4000000000);
    append(ref mut values, -3);
    append(ref mut values, 9223372036854775807);
    let ref i64[] all = vec::view_i64(ref values);
    let ref i64[] sibling = vec::view_i64(ref values);
    assert(all.len == 4 && !all.is_empty);
    assert(all[0] == -9223372036854775808 && sibling[3] == 9223372036854775807);
    assert(sum(middle(ref values)) == 3999999997);
    assert(all[1..][..2][1] == -3 && all[4..].is_empty);
    assert(all[..].len == sibling.len);
    let ref i64[] tagged = switch (first_two(ref values)) {
        case .some(view) : { view }
        case .none : { assert(false); vec::view_i64(ref values)[..0] }
    };
    assert(tagged[1] == 4000000000);

    // The last view use precedes growth. Capacity exceeds the old allocation.
    let usize previous_capacity = vec::capacity<i64>(ref values);
    assert(switch (vec::reserve<i64>(ref mut values, previous_capacity + 1024)) {
        case .ok(capacity) : { capacity >= previous_capacity + 1024 }
        case .err(_) : { false }
    });
    assert(vec::view_i64(ref values)[1] == 4000000000);
    append(ref mut values, 42);
    assert(vec::view_i64(ref values)[4] == 42);
    assert(switch (vec::pop<i64>(ref mut values)) {
        case .some(value) : { value == 42 }
        case .none : { false }
    });
    assert(vec::truncate<i64>(ref mut values, 3) == 1);
    assert(vec::clear<i64>(ref mut values) == 3);
    assert(vec::view_i64(ref values)[..].is_empty);
    append(ref mut values, 99);
    let vec::Vec<i64> moved = values;
    assert(vec::view_i64(ref moved)[0] == 99);
    return 0;
}
