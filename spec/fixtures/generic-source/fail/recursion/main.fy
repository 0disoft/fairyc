module main;
fn T recursive<T>(T value) { return recursive<T>(value); }
fn i32 main() { return recursive<i32>(42); }
