module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Result<i32, Token> wrapped = .err(Token { .value = 42 });
    return switch (wrapped) {
        case .ok(value) : { value }
        case .err(token) : { token.value }
    };
}
