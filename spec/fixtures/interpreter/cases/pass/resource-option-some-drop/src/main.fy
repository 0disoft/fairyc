module main;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let wrapped: Option<Token> = Option::Some(Token { value: 42 });
    42
}
