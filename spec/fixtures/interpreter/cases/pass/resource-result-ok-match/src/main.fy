module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Result<Token, i32> wrapped = .ok(Token { .value = 42 });
    return switch (wrapped) {
        case .ok(token) : { token.value }
        case .err(error) : { -error }
    };
}
