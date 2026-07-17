module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn run() -> i32 {
    let first: Token = Token { value: 1 };
    let second: Token = Token { value: 2 };
    second.value
}
