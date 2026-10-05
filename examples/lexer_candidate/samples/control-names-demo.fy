module main;
fn i64 count(i64 limit) {
    var i64 total = 0;
    for (var i64 index = 0; index < limit; index += 1) {
        if (index % 2 == 0) { total += index; } else { continue; }
    }
    while (total > limit) { total -= 1; }
    return total;
}
