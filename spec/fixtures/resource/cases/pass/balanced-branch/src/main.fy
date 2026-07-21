module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 consume(Token token) {
    return token.value;
}

fn i32 run(bool flag) {
    let Token token = Token { .value = 1 };
    let i32 result = if (flag) { consume(token) } else { consume(token) };
    return result;
}
