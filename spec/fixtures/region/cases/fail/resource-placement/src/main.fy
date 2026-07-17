module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn invalid(scratch: Region, token: Token) -> i32 {
    let stored: &Token = scratch.place(token);
    stored.value
}
