module main;
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
fn ref mut i64[] tail(ref mut i64[] values) { return values[1..]; }
fn void fill(ref mut i64[] values) { values[0] = -4000000000; }
fn void read(ref i64[] values) { assert(values.len > 0); }
fn Option<ref mut i64[]> tagged(ref mut i64[] values) { return .some(values[1..]); }
fn i32 main() {
    var i64[5] data = [9223372036854775807,4000000000,-7,-9223372036854775808,9];
    let ref mut i64[] view = ref mut data;
    let ref mut i64[] middle = view[1..4];
    reverse(middle);
    assert(view[0] == 9223372036854775807 && view[1] == -9223372036854775808);
    assert(view[2] == -7 && view[3] == 4000000000 && view[4] == 9);
    let ref mut i64[] child = tail(view);
    let ref mut i64[] leaf = child[..1];
    fill(leaf);
    read(child);
    assert(view[1] == -4000000000);
    let Option<ref mut i64[]> result = tagged(view);
    switch (result) { case .some(values): { values[0] += 1; } case .none: { assert(false); } };
    assert(view[1] == -3999999999);
    let ref mut i64[] empty = view[view.len..];
    assert(empty.is_empty && empty.len == 0);
    var usize order = 0;
    let ref mut i64[] ordered = view[{ order = order * 10 + 1; 1 }..{ order = order * 10 + 2; 4 }];
    assert(order == 12);
    reverse(ordered);
    assert(view[1] == 4000000000 && view[3] == -3999999999);
    assert(data[1] == 4000000000 && data[2] == -7 && data[3] == -3999999999);
    data[0] = 0;
    assert(data[0] == 0);
    return 0;
}
