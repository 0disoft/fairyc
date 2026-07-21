module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    var Token token = Token { .value = 42 };
    token = token;
    return token.value;
}
