module main;
fn T identity<T>(T value) { return value; }
fn T forward<T>(T value) { let T local = identity<T>(value); return local; }
fn i32 main() {
    if (identity<bool>(true)) { return forward<i32>(identity<i32>(42)); }
    return 0;
}
