module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    var token: Token = Token { value: 42 };
    token = token;
    token.value
}
