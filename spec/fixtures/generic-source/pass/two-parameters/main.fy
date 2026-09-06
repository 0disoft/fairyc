module main;
fn A first<A, B>(A value, B unused) { return value; }
fn i32 main() { return first<i32, bool>(42, true); }
