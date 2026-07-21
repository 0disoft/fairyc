module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 run() {
    let Token first = Token { .value = 1 };
    let Token second = Token { .value = 2 };
    return second.value;
}
