module main;

pub fn ref i64[] tail(ref i64[] values) { return values[1..]; }

fn i64 sum(ref i64[] values) {
    var i64 result = 0;
    var usize index = 0;
    while (index < values.len) {
        result += values[index];
        index += 1;
    }
    return result;
}

fn Option<ref i64[]> pair(ref i64[] values) { return .some(values[2..]); }

fn i32 main() {
    var i64[4] data = [9223372036854775807, -9223372036854775808, 4000000000, -3];
    let ref i64[] whole = ref data;
    assert(whole.len == 4);
    assert(whole[0] == 9223372036854775807);
    assert(whole[1] == -9223372036854775808);
    let ref i64[] selected = whole[2..];
    assert(sum(selected) == 3999999997);
    let ref i64[] left = whole[..2];
    let ref i64[] right = whole[2..];
    assert(left[0] == 9223372036854775807 && right[1] == -3);
    let ref i64[] nested = tail(selected);
    assert(nested.len == 1 && nested[0] == -3);
    let ref i64[] empty = whole[whole.len..];
    assert(empty.is_empty && empty.len == 0 && sum(empty) == 0);
    let ref i64[] copy = whole[..];
    assert(copy[2] == 4000000000);
    let Option<ref i64[]> tagged = pair(whole);
    switch (tagged) {
        case .some(value): { assert(sum(value) == 3999999997); }
        case .none: { assert(false); }
    };
    // The last view use has passed, so the original array can be edited again.
    data[2] = 5;
    assert(sum(ref data) == 1);
    return 0;
}
