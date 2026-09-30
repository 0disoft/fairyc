module main;
struct Total { i64 value; }
fn i64 sum(i64 a, i64 b) { return a + b; }
fn void bump(ref mut i64 v) { *v = *v + 1; }
fn Option<i64> optional(i64 x) { return .some(x); }
fn Result<i64, i64> result(i64 x) { return .ok(x); }
fn i32 main() {
    var i64 n = 5000000000;
    bump(ref mut n);
    assert(n == 5000000001);
    assert(sum(n, -1) == 5000000000);
    let i64 lo = -9223372036854775808;
    let i64 hi = 9223372036854775807;
    let i64 root = 3037000499;
    assert(hi * 1 == hi && lo * 1 == lo && lo * 0 == 0);
    assert(root * root == 9223372030926249001 && root * -root == -9223372030926249001);
    assert(-root * -root == 9223372030926249001);
    assert(lo < hi && lo != hi && hi >= lo && hi > lo);
    assert(lo / 1 == lo && lo % 10 == -8 && hi / -1 == -9223372036854775807);
    assert(lo >> 63 == -1 && lo << 0 == lo && (lo >> 1) << 1 == lo);
    let i64 negative = -1;
    assert(negative << 63 == lo && ~lo == hi && (lo | hi) == negative && (lo & hi) == 0);
    let i32 narrow = -2147483648;
    let u32 wide = 4294967295;
    let u8 byte = 255;
    assert(narrow as i64 == -2147483648 && wide as i64 == 4294967295 && byte as i64 == 255);
    var Total total = Total { .value = n };
    total.value += 2;
    var i64[2] values = [lo, hi];
    values[0] = total.value;
    assert(values[0] == 5000000003 && values[1] == hi);
    let i64 option = switch (optional(n)) { case .some(v): { v } case .none: { 0 } };
    let i64 checked = switch (result(n)) { case .ok(v): { v } case .err(e): { e } };
    assert(option == n && checked == n);
    return 0;
}
