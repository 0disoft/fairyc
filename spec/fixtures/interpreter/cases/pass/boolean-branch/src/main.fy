module main;

fn i32 choose(bool flag) {
    return if (flag) { 42 } else { 0 };
}

fn i32 main() {
    return choose(true);
}
