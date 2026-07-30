module lib;

pub struct Pair {
    i32 value;
}

pub resource Token {
    i32 value;
    cleanup {}
}

pub fn Token token(i32 value) {
    return Token { .value = value };
}
