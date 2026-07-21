module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 main() {
    let Option<Option<Token>> wrapped = .some(.some(Token { .value = 42 }));
    return 42;
}
