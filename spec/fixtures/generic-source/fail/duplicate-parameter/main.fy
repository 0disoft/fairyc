module main;
fn T identity<T, T>(T value) { return value; }
fn i32 main() { return identity<i32, i32>(42); }
