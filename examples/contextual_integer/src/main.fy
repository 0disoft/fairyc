module main;
const u32 WORD = 2;
const u32 QUOTIENT = (2147483647 + 1) / WORD;
const bool COMPARED = 1 < WORD;
static_assert(QUOTIENT == 1073741824 && COMPARED);
fn usize next(ref mut i32 count) { *count = *count + 1; return 2; }
fn usize ordered_next(ref mut i32 count) { assert(*count == 2); *count = *count + 1; return 2; }
fn u32 take(u32 value) { return value; }
fn usize buffer_size(usize rows) { return 1048576 / rows; }
fn i32 main() {
    var usize rows = 2;
    assert(buffer_size(rows) == 524288);
    let inferred = ((1048576)) / rows;
    assert(inferred == 524288);
    assert(1 < rows && 2 == rows && 3 != rows);
    let u32 wide = 2147483647 + 1;
    let i64 signed = -5000000000 / 2;
    let usize maximum = 18446744073709551615 / 1;
    assert(wide == 2147483648 && signed == -2500000000);
    assert(maximum == 18446744073709551615);
    assert(take(2147483647 + 1) == wide);
    let u32[2] values = [2147483647 + 1; 2];
    assert(values[1] == wide);
    var i32 calls = 0;
    let result = 8 / next(ref mut calls);
    assert(calls == 1 && result == 4);
    let usize ordered = { calls += 1; 8 } / ordered_next(ref mut calls);
    assert(calls == 3 && ordered == 4);
    var u8 limit = 254;
    let u8 sum = 1 + limit;
    assert(sum == 255);
    var u32 amount = 1;
    let u32 shifted = 1 << amount;
    assert(shifted == 2);
    let u32 mask = ~0;
    assert(mask == 4294967295);
    return 0;
}
