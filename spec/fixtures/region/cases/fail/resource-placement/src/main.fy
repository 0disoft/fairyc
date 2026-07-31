module main;

resource Token {
    i32 value;

    cleanup {}
}

fn i32 invalid(Region scratch, Token token) {
    discard scratch.place(token);
    return 0;
}
