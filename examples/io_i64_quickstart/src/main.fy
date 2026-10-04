module main;
import std::io as io;

@effects(console)
fn bool show(i64 value, usize length) {
    if (!switch (io::write_i64(value)) {
        case .ok(written): { written == length }
        case .err(_): { false }
    }) { return false; }
    if (!switch (io::write("\n")) { case .ok(written): { written == 1 } case .err(_): { false } }) { return false; }
    if (!switch (io::write_stderr_i64(value)) {
        case .ok(written): { written == length }
        case .err(_): { false }
    }) { return false; }
    return switch (io::write_stderr("\n")) { case .ok(written): { written == 1 } case .err(_): { false } };
}

@effects(console)
fn i32 main() {
    if (!show(0, 1)) { return 2; }
    if (!show(1, 1)) { return 2; }
    if (!show(-1, 2)) { return 2; }
    if (!show(9, 1)) { return 2; }
    if (!show(-9, 2)) { return 2; }
    if (!show(10, 2)) { return 2; }
    if (!show(-10, 3)) { return 2; }
    if (!show(2147483647, 10)) { return 2; }
    if (!show(-2147483648, 11)) { return 2; }
    if (!show(2147483648, 10)) { return 2; }
    if (!show(-2147483649, 11)) { return 2; }
    if (!show(4000000000, 10)) { return 2; }
    if (!show(-4000000000, 11)) { return 2; }
    if (!show(9223372036854775807, 19)) { return 2; }
    if (!show(-9223372036854775808, 20)) { return 2; }
    return 0;
}
