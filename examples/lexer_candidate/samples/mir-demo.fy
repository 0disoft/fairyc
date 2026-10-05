module demo;

fn i64 add(i64 left, i64 right) { return left + right; }

pub fn i32 main() {
    var i64 total = 4000000000;
    total += add(1, 2);
    assert(total == 4000000003);
    return 0;
}
