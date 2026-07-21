module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    var Token token = Token { .value = 1 };
    token = Token { .value = 42 };
    return token.value;
}
