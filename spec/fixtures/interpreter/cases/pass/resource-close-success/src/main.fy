module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 finish(Token token) {
    return token.value;
}

fn Result<i32, Token> close(Token token, bool fail) {
    return if (fail) {
        .err(token)
    } else {
        .ok(finish(token))
    };
}

fn i32 main() {
    let Token token = Token { .value = 42 };
    return switch (close(token, false)) {
        case .ok(value) : { value }
        case .err(returned) : { returned.value }
    };
}
