module main;
struct Point { i64 x; i32 y; }
fn i64 sum(ref i64[] values) {
    var i64 total = 0;
    for (let i64 value in values) { total += value; }
    return total;
}
fn void adjust(ref mut Point[] values) {
    for (let ref mut Point value in values) { value.x += 10; }
}
fn i64 copied(i64[4] values) { return sum(ref values[1..3]); }
fn ref i64[] relay(ref i64[] values) { return values; }
fn Option<ref i64[]> wrapped(ref i64[] values) { return .some(values); }
fn i32 main() {
    let i64[4] fixed = [4000000000, 2, 3, 4];
    let ref i64[] middle = ref fixed[1..3];
    assert(middle.len == 2 && sum(middle) == 5 && copied(fixed) == 5);
    assert(sum(ref (fixed)[..2]) == 4000000002);
    assert(sum(ref fixed[2..]) == 7 && sum(ref fixed[..]) == 4000000009);
    let ref i64[] empty = ref fixed[4..4]; assert(empty.is_empty);
    var i64[4] data = fixed;
    var usize calls = 0;
    let ref mut i64[] selected = ref mut (data[{calls += 1; 1}..{assert(calls == 1); calls += 1; 3}]);
    selected[0] += 20; selected[1] += 30;
    assert(selected[0] == 22 && calls == 2);
    assert(data[0] == 4000000000 && data[1] == 22 && data[2] == 33 && data[3] == 4);
    let ref i64[] returned = relay(ref data[1..3]);
    let ref i64[] alias = returned; assert(sum(alias) == 55);
    switch (wrapped(ref data[1..3])) {
        case .some(value): { assert(sum(value) == 55); }
        case .none: { assert(false); }
    };
    var Point[4] points = [Point { .x = 4000000000, .y = 7 }; 4];
    let ref Point[] two = ref ((points)[1..3]);
    assert(two.len == 2 && two[0].x == 4000000000);
    adjust(ref mut points[1..3]);
    assert(points[0].x == 4000000000 && points[1].x == 4000000010);
    assert(points[2].x == 4000000010 && points[3].x == 4000000000);
    let ref mut Point[] tail = ref mut points[2..]; tail[1].y = 9;
    assert(points[3].y == 9 && points[2].y == 7);
    let bool[2] flags = [true, false]; let ref bool[] flag = ref flags[1..]; assert(!flag[0]);
    let i32[2] ints = [1, 2]; let ref i32[] int = ref ints[..1]; assert(int[0] == 1);
    let u8[2] bytes = [1, 2]; let ref u8[] byte = ref bytes[1..]; assert(byte[0] == 2);
    let u32[2] words = [1, 2]; let ref u32[] word = ref words[..1]; assert(word[0] == 1);
    let usize[2] sizes = [1, 2]; let ref usize[] size = ref sizes[1..]; assert(size[0] == 2);
    return 0;
}
