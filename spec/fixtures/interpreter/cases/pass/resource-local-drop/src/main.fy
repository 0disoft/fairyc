module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Token token = Token { .value = 42 };
    return token.value;
}
