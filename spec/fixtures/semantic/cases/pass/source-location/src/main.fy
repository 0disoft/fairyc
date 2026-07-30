module main;

fn ref str locate() {
    return source_location();
}

fn i32 main() {
    let ref str location = locate();
    return if (location.len > 0) { 42 } else { 1 };
}
