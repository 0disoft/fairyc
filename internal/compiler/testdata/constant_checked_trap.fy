module main;
struct Span { i64 start; i64 end; }
struct Owner { Span position; }
fn Span bad() {
    var i64 n = 9223372036854775807;
    n += 1;
    return Span { .start = n, .end = 0 };
}
fn i32 main() {
    var Owner s = Owner { .position = Span { .start = 3, .end = 4 } };
    s.position = bad();
    return 0;
}
