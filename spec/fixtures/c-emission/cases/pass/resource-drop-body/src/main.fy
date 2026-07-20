module main;

fn observe(value: i32) -> i32 { value }

resource Token {
    value: i32;

    drop fn(self: &mut Self) {
        let observed: i32 = observe(self.value);
    }
}

fn main() -> i32 {
    let token: Token = Token { value: 65 };
    42
}
