module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn consume(token: Token) -> i32 {
    token.value
}

fn main() -> i32 {
    var token: Token = Token { value: 1 };
    let ignored: i32 = consume(token);
    token = Token { value: 42 };
    token.value
}
