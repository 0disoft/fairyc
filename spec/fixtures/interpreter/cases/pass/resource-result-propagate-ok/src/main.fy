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
    return switch (read(.ok(Token { .value = 42 }))) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
}
