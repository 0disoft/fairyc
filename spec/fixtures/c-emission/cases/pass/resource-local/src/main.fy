module main;

resource Pair {
    i32 left;
    i32 right;

    cleanup {}
}

fn i32 main() {
    let Pair pair = Pair { .left = 1, .right = 42 };
    return pair.right;
}
