module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Option<Token> wrapped = .some(Token { .value = 42 });
    return 42;
}
