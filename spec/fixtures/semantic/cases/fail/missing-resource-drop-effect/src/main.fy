module main;

@effects(filesystem)
resource Token {
    i32 value;

    cleanup {
        let i32 marker = this.value;
    }
}

fn i32 main() {
    let Token token = Token { .value = 42 };
    return 0;
}
