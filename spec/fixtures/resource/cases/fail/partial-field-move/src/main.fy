module main;

resource Token {
    i32 value;

    cleanup {}
}

resource Pair {
    Token left;
    Token right;

    cleanup {}
}

fn i32 invalid() {
    let Pair pair = Pair {
        .left = Token { .value = 1 },
        .right = Token { .value = 2 },
    };
    let Token extracted = pair.left;
    return extracted.value;
}
