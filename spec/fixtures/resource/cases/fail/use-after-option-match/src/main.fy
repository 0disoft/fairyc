module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Option<Token> wrapped = .some(Token { .value = 42 });
    let i32 value = switch (wrapped) {
        case .some(token) : { token.value }
        case .none : { 0 }
    };
    wrapped;
    return value;
}
