module demo;

fn i64 choose(i64 a, i64 b, bool first) {
    return if (first) { a } else { b };
}

pub fn i64 total(i64 seed) {
    var i64 sum = seed;
    for (var i32 index = 0; index < 3; index += 1) {
        sum += choose(4000000000, 1, index == 0);
    }
    return sum;
}
