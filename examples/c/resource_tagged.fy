module main;

resource Token {
    i32 value;

    cleanup {}
}

fn Result<i32, i32> read(Result<Token, i32> input) {
    let Token token = input?;
    return .ok(token.value);
}

fn i32 main() {
    let Option<Token> wrapped = .some(Token { .value = 42 });
    let Token token = switch (wrapped) {
        case .some(token) : { token }
        case .none : { Token { .value = 0 } }
    };
    return switch (read(.ok(token))) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
