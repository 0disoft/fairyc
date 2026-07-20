module main;

@effects(filesystem)
resource Token {
    value: i32;

    drop fn(self: &mut Self) {
        let marker: i32 = touch();
    }
}

@effects(filesystem)
fn touch() -> i32 {
    1
}

@effects(filesystem)
fn main() -> i32 {
    let token: Token = Token { value: 42 };
    0
}
