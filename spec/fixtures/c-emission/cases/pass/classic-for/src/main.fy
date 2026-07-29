module main;

fn i32 main() {
    var i32 total = 0;
    for (var i32 index = 0; index < 6; index = index + 1) {
        if (index < 2) {
            continue;
        }
        total = total + 7;
    }
    return total + 14;
}
