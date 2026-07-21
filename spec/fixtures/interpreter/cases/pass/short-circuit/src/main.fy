module main;

fn i32 main() {
    return if ((false && (1 / 0 == 0)) || (true || (1 / 0 == 0))) {
        42
    } else {
        0
    };
}
