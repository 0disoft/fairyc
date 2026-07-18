module nested;

resource Token {
    value: i32;

    drop fn(self: &mut Self) {}
}

fn wrap(value: Token) -> Option<Option<Token>> {
    Option::Some(Option::Some(value))
}
