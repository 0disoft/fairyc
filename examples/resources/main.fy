module examples::resources::main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 consume(Token token) {
    return token.value;
}

fn i32 run() {
    let Token token = Token { .value = 42 };
    return consume(token);
}
