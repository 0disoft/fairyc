module main;
struct Span { i64 x; i32 y; bool ready; u8 byte; u32 word; usize size; }
struct Point { Span position; i64 sibling; }
fn void add(ref mut i64 value, i64 amount) { *value = *value + amount; }
fn i64 read(ref i64 value) { return *value; }
fn ref i64 shared(ref i64 value) { return value; }
fn ref mut i64 exclusive(ref mut i64 value) { return value; }
fn Option<ref i64> optional(ref i64 value) { return .some(value); }
fn Result<ref mut i64,u32> result(ref mut i64 value) { return .ok(value); }
fn i64 copied(Point[2] data) {
    let ref i64 field = ref data[1].position.x;
    return *field;
}
fn i32 main() {
    let Point origin = Point { .position = Span { .x = 4000000000, .y = 2,
        .ready = true, .byte = 3, .word = 4, .size = 5 }, .sibling = 7 };
    let Point[2] fixed = [origin; 2];
    let ref i64 first = ref fixed[0].position.x;
    let ref Span whole = ref ((fixed)[1]).position;
    assert(*first == whole.x && copied(fixed) == 4000000000);
    var Point[2] points = fixed;
    let ref i64 a = ref points[0].position.x;
    let ref i64 b = ref points[1].position.x;
    assert(read(a) == read(b));
    add(ref mut points[0].position.x, 20);
    let ref i64 returned = shared(ref points[0].position.x);
    let ref i64 returned_copy = returned;
    assert(*returned_copy == 4000000020);
    switch (optional(ref points[0].position.x)) {
        case .some(value): { assert(*value == 4000000020); }
        case .none: { assert(false); }
    };
    var usize calls = 0;
    var usize index = 1;
    let ref mut i64 selected = exclusive(ref mut ((points)[{calls += 1; index}]).position.x);
    index = 0;
    add(selected, -10);
    assert(*selected == 3999999990 && calls == 1);
    let ref mut Span nested = ref mut points[1].position;
    nested.y = 9;
    let ref mut Span alias = nested;
    alias.x += 1;
    switch (result(ref mut points[0].position.x)) {
        case .ok(value): { add(value, 2); }
        case .err(_): { assert(false); }
    };
    let ref mut i32 y = ref mut points[0].position.y; *y = 11;
    let ref mut bool ready = ref mut points[0].position.ready; *ready = false;
    let ref mut u8 byte = ref mut points[0].position.byte; *byte = 6;
    let ref mut u32 word = ref mut points[0].position.word; *word = 8;
    let ref mut usize size = ref mut points[0].position.size; *size = 10;
    let ref i32 cy = ref points[0].position.y; assert(*cy == 11);
    let ref bool cr = ref points[0].position.ready; assert(!*cr);
    let ref u8 cb = ref points[0].position.byte; assert(*cb == 6);
    let ref u32 cw = ref points[0].position.word; assert(*cw == 8);
    let ref usize cs = ref points[0].position.size; assert(*cs == 10);
    add(ref mut points[points.len - 1].sibling, 3);
    assert(points[0].position.x == 4000000022 && points[1].position.x == 3999999991);
    assert(points[1].position.y == 9 && points[1].sibling == 10);
    assert(fixed[0].position.x == 4000000000 && fixed[0].position.ready);
    return 0;
}
