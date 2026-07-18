module examples::run::resource_option;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn main() -> i32 {
    let wrapped: Option<Token> = Option::Some(Token { value: 42 });
    match wrapped {
        Option::Some(token) => token.value,
        Option::None => 0,
    }
}
