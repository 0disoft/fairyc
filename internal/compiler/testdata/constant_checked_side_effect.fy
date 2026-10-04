module main;
import std::io as io;
struct Span { i64 start; i64 end; }
struct Owner { Span position; }
@effects(console)
fn Span bad() {
    discard io::write("before\n");
    var i64 n = 9223372036854775807;
    n += 1;
    return Span { .start = n, .end = 0 };
}
@effects(console)
fn i32 main() {
    var Owner s = Owner { .position = Span { .start = 3, .end = 4 } };
    s.position = bad();
    discard io::write("after\n");
    return 0;
}
