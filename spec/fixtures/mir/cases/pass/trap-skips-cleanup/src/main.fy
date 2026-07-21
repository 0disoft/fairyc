module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 divide(i32 divisor) {
    let Token token = Token { .value = 12 };
    return token.value / divisor;
}
