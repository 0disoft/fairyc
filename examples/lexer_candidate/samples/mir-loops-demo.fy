module main;

fn i32 main() {
    var i64 total = 4000000000;
    for (var i64 i = 0; i < 4; i += 1) {
        if (i == 2) { continue; }
        total += i;
    }
    var i32 count = 0;
    while (count < 3) {
        count += 1;
        if (count == 2) { break; }
    }
    assert(total == 4000000004);
    return count;
}
