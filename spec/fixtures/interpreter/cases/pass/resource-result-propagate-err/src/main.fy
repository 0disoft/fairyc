module main;

resource Token {
    i32 value;

    cleanup {}
}

fn Result<i32, Token> pass(Result<i32, Token> input) {
    let i32 value = input?;
    return .ok(value);
}

fn i32 main() {
    return switch (pass(.err(Token { .value = 42 }))) {
        case .ok(value) : { value }
        case .err(token) : { token.value }
    };
}
