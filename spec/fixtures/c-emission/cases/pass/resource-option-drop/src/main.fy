module main;

resource Token {
    i32 value;
    cleanup {}
}

fn i32 main() {
    let Option<Token> present = .some(Token { .value = 17 });
    let Option<Token> absent = .none;
    return 42;
}
