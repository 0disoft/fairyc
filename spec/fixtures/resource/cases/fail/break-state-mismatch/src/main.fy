module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 consume(Token token) {
    return token.value;
}

fn i32 invalid(bool flag) {
    let Token token = Token { .value = 1 };
    while (flag) {
        consume(token);
        break;
    }
    return 0;
}
