module main;

resource Token {
    i32 value;

    cleanup {}
}

fn Token unwrap(Option<Token> input) {
    return switch (input) {
        case .some(token) : { token }
        case .none : { Token { .value = 0 } }
    };
}

fn i32 main() {
    let Token token = unwrap(.some(Token { .value = 42 }));
    return token.value;
}
