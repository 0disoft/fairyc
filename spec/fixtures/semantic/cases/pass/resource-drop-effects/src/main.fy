module main;

@effects(filesystem)
resource Token {
    i32 value;

    cleanup {
        let i32 marker = touch();
    }
}

@effects(filesystem)
fn i32 touch() {
    return 1;
}

@effects(filesystem)
fn i32 main() {
    let Token token = Token { .value = 42 };
    return 0;
}
