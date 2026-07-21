module nested;

resource Token {
    i32 value;

    cleanup {}
}

fn Option<Option<Token>> wrap(Token value) {
    return .some(.some(value));
}
