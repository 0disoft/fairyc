module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Token source = Token { .value = 42 };
    let Option<Token> wrapped = .some(source);
    return switch (wrapped) {
        case .some(token) : { token.value }
        case .none : { 0 }
    };
}
