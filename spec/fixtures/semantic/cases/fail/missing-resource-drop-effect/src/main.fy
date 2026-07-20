module main;

@effects(filesystem)
resource Token {
    value: i32;

    drop fn(self: &mut Self) {
        let marker: i32 = self.value;
    }
}

fn main() -> i32 {
    let token: Token = Token { value: 42 };
    0
}
