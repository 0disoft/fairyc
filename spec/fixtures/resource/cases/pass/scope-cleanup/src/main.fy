module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 read(ref Token token) {
    return token.value;
}

fn i32 run() {
    let Token token = Token { .value = 1 };
    return read(ref token);
}
