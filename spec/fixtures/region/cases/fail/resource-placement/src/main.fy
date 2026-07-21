module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 invalid(Region scratch, Token token) {
    let ref Token stored = scratch.place(token);
    return stored.value;
}
