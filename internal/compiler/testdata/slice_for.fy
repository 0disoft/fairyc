module main;
struct Pos { i64 x; bool ready; }
struct Point { Pos pos; i32 y; }
fn Option<ref Point> first(ref Point[] view) {
    for (let ref Point point in view[1..]) { return .some(point); }
    return .none;
}
fn Option<ref mut i64> first_mut(ref mut i64[] view) {
    for (let ref mut i64 item in view[1..]) { return .some(item); }
    return .none;
}
fn void fill(ref mut i64[] view, i64 value) {
    for (let ref mut i64 item in view[1..]) { *item = value; }
}
fn i32 main() {
    var usize calls = 0;
    var i64[4] data = [1,4000000000,3,4];
    var usize end = 3;
    var i64 sum = 0;
    for (let i64 item in data[{ calls += 1; 1 }..{ calls += 10; end }]) {
        end = 1; sum += item;
    }
    assert(calls == 11 && sum == 4000000003);
    for (let ref mut i64 item in data[1..3]) { *item = *item + 1; }
    assert(data[0] == 1 && data[1] == 4000000001 && data[2] == 4 && data[3] == 4);
    var usize count = 0;
    for (let ref i64 item in data[..]) { assert(*item > 0); count += 1; }
    for (let i64 item in data[4..]) { assert(false); }
    for (let ref mut i64 item in data[2..2]) { assert(false); }
    assert(count == 4);
    let ref mut i64[] view = ref mut data;
    fill(view, 7);
    for (let i64 item in view[1..]) { assert(item == 7); }
    switch (first_mut(view)) { case .some(item): { *item = 8; } case .none: { assert(false); } };
    assert(view[1] == 8);
    let Point origin = Point { .pos = Pos { .x = 4000000000, .ready = true }, .y = 2 };
    var Point[3] points = [origin; 3];
    for (let Point point in points[1..]) { var Point copy = point; copy.pos.x += 1; assert(copy.pos.x == 4000000001); }
    for (let ref mut Point point in points[1..3]) { point.pos.x += 2; point.y = 9; }
    for (let ref Point point in points[1..]) { assert(point.pos.x == 4000000002 && point.y == 9); }
    assert(points[0].pos.x == 4000000000);
    let ref Point[] vertices = ref points;
    switch (first(vertices)) { case .some(point): { assert(point.y == 9); } case .none: { assert(false); } };
    var bool[3] flags = [true,false,true];
    for (let ref mut bool flag in flags[1..]) { *flag = !*flag; }
    for (let bool flag in flags[..1]) { assert(flag); }
    assert(flags[1] && !flags[2]);
    var u8[3] bytes = [1,2,3];
    for (let ref mut u8 byte in bytes[1..]) { *byte = *byte + 1; continue; }
    assert(bytes[1] == 3 && bytes[2] == 4);
    var i32[2] signed = [-2,3];
    for (let ref mut i32 item in signed[..]) { *item = *item + 1; break; }
    assert(signed[0] == -1 && signed[1] == 3);
    var u32[2] words = [1,2];
    for (let ref mut u32 word in words[..]) { *word = *word + 1; }
    var usize[2] sizes = [1,2];
    for (let ref mut usize size in sizes[..]) { *size = *size + 1; }
    assert(words[1] == 3 && sizes[1] == 3);
    return 0;
}
