module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn divide(divisor: i32) -> i32 {
    let token: Token = Token { value: 12 };
    token.value / divisor
}
