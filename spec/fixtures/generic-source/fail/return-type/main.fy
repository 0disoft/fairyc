module main;
fn T wrong<T>(T value) { return false; }
fn i32 main() { return wrong<i32>(42); }
