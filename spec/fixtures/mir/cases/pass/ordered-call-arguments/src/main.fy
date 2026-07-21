module main;

fn i32 first() {
    return 1;
}

fn i32 second() {
    return 2;
}

fn i32 choose(i32 left, i32 right) {
    return left;
}

fn i32 run() {
    return choose(first(), second());
}
