module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Option<Token> wrapped = .none;
    return 42;
}
