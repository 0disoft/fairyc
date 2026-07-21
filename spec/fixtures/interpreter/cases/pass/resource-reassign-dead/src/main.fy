module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 consume(Token token) {
    return token.value;
}

fn i32 main() {
    var Token token = Token { .value = 1 };
    let i32 ignored = consume(token);
    token = Token { .value = 42 };
    return token.value;
}
