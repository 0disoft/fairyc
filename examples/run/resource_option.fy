module examples::run::resource_option;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Option<Token> wrapped = .some(Token { .value = 42 });
    return switch (wrapped) {
        case .some(token) : { token.value }
        case .none : { 0 }
    };
}
