module main;

@must_use
fn i32 status() { return 7; }

fn i32 main() {
    discard status();
    let stored = status();
    return stored;
}
