module main;

fn i32 observe(i32 value) { return value; }

resource Token {
    i32 value;

    cleanup {
        let i32 observed = observe(this.value);
    }
}

fn i32 main() {
    let Token token = Token { .value = 65 };
    return 42;
}
