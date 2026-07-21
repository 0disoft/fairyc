module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 consume(Token token) {
    return token.value;
}

fn i32 invalid() {
    let Token token = Token { .value = 1 };
    let i32 first = consume(token);
    return consume(token);
}
