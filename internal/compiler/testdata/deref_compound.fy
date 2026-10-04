module main;
struct Counter { i64 value; }
fn void edit_u8(ref mut u8 item) {
    *(item) += 5; assert(*item == 17);
    *(item) -= 2; assert(*item == 15);
    *(item) *= 3; assert(*item == 45);
    *(item) /= 4; assert(*item == 11);
    *(item) %= 5; assert(*item == 1);
    *(item) <<= 2; assert(*item == 4);
    *(item) >>= 1; assert(*item == 2);
    *(item) |= 8; assert(*item == 10);
    *(item) &= 11; assert(*item == 10);
    *(item) ^= 3; assert(*item == 9);
}
fn void edit_i32(ref mut i32 item) {
    *(item) += 5; assert(*item == 17);
    *(item) -= 2; assert(*item == 15);
    *(item) *= 3; assert(*item == 45);
    *(item) /= 4; assert(*item == 11);
    *(item) %= 5; assert(*item == 1);
    *(item) <<= 2; assert(*item == 4);
    *(item) >>= 1; assert(*item == 2);
    *(item) |= 8; assert(*item == 10);
    *(item) &= 11; assert(*item == 10);
    *(item) ^= 3; assert(*item == 9);
}
fn void edit_i64(ref mut i64 item) {
    *(item) += 5; assert(*item == 17);
    *(item) -= 2; assert(*item == 15);
    *(item) *= 3; assert(*item == 45);
    *(item) /= 4; assert(*item == 11);
    *(item) %= 5; assert(*item == 1);
    *(item) <<= 2; assert(*item == 4);
    *(item) >>= 1; assert(*item == 2);
    *(item) |= 8; assert(*item == 10);
    *(item) &= 11; assert(*item == 10);
    *(item) ^= 3; assert(*item == 9);
}
fn void edit_u32(ref mut u32 item) {
    *(item) += 5; assert(*item == 17);
    *(item) -= 2; assert(*item == 15);
    *(item) *= 3; assert(*item == 45);
    *(item) /= 4; assert(*item == 11);
    *(item) %= 5; assert(*item == 1);
    *(item) <<= 2; assert(*item == 4);
    *(item) >>= 1; assert(*item == 2);
    *(item) |= 8; assert(*item == 10);
    *(item) &= 11; assert(*item == 10);
    *(item) ^= 3; assert(*item == 9);
}
fn void edit_usize(ref mut usize item) {
    *(item) += 5; assert(*item == 17);
    *(item) -= 2; assert(*item == 15);
    *(item) *= 3; assert(*item == 45);
    *(item) /= 4; assert(*item == 11);
    *(item) %= 5; assert(*item == 1);
    *(item) <<= 2; assert(*item == 4);
    *(item) >>= 1; assert(*item == 2);
    *(item) |= 8; assert(*item == 10);
    *(item) &= 11; assert(*item == 10);
    *(item) ^= 3; assert(*item == 9);
}
fn i32 early(ref mut i64 item) { *item += { return 42; 1 }; return 0; }
fn Result<i64,u32> fail() { return .err(9); }
fn Result<i64,u32> propagate(ref mut i64 item) { *item += fail()?; return .ok(*item); }
fn i32 main() {
    var u8[2] data_u8 = [12,12];
    for (let ref mut u8 item in data_u8[1..]) { edit_u8(item); }
    assert(data_u8[0] == 12 && data_u8[1] == 9);
    var i32[2] data_i32 = [12,12];
    for (let ref mut i32 item in data_i32[1..]) { edit_i32(item); }
    assert(data_i32[0] == 12 && data_i32[1] == 9);
    var i64[2] data_i64 = [12,12];
    for (let ref mut i64 item in data_i64[1..]) { edit_i64(item); }
    assert(data_i64[0] == 12 && data_i64[1] == 9);
    var u32[2] data_u32 = [12,12];
    for (let ref mut u32 item in data_u32[1..]) { edit_u32(item); }
    assert(data_u32[0] == 12 && data_u32[1] == 9);
    var usize[2] data_usize = [12,12];
    for (let ref mut usize item in data_usize[1..]) { edit_usize(item); }
    assert(data_usize[0] == 12 && data_usize[1] == 9);
    var i64 x = 10;
    let ref mut i64 snapshot = ref mut x;
    *snapshot += { *snapshot = 100; 2 };
    assert(x == 12);
    var i64 y = 20;
    var ref mut i64 selected = ref mut x;
    var usize calls = 0;
    *selected += { calls += 1; selected = ref mut y; 3 };
    assert(*selected == 20 && calls == 1);
    assert(x == 15 && y == 20);
    assert(early(ref mut x) == 42 && x == 15);
    switch (propagate(ref mut x)) { case .err(code): { assert(code == 9); } case .ok(_): { assert(false); } };
    assert(x == 15);
    var Counter counter = Counter { .value = 4000000000 };
    let ref mut i64 field = ref mut counter.value;
    *field += 2;
    assert(counter.value == 4000000002);
    var i64 neg = -8;
    let ref mut i64 signed = ref mut neg;
    *signed >>= 1;
    assert(neg == -4);
    return 0;
}
