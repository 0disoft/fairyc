module output;
import std::io as io;
import std::bytes as bytes;
import std::text as utf8;

@effects(console)
pub fn i32 error(ref str path, ref str message) {
    let bool a = switch (io::write_stderr(path)) { case .ok(_) : { true } case .err(_) : { false } };
    let bool b = switch (io::write_stderr(": error: ")) { case .ok(_) : { true } case .err(_) : { false } };
    let bool c = switch (io::write_stderr(message)) { case .ok(_) : { true } case .err(_) : { false } };
    let bool d = switch (io::write_stderr("\n")) { case .ok(_) : { true } case .err(_) : { false } };
    return 1;
}
@effects(console)
pub fn i32 usage(ref str message) {
    let i32 code = error("task-manager-cli", message);
    return 2;
}
@effects(console)
pub fn i32 file_error(ref str path, ref str operation, u32 status, u32 os_code) {
    let bool a = switch (io::write_stderr(path)) { case .ok(_) : { true } case .err(_) : { false } };
    let bool b = switch (io::write_stderr(": error: ")) { case .ok(_) : { true } case .err(_) : { false } };
    let bool c = switch (io::write_stderr(operation)) { case .ok(_) : { true } case .err(_) : { false } };
    let bool d = switch (io::write_stderr(" (status=")) { case .ok(_) : { true } case .err(_) : { false } };
    let bool e = switch (io::write_stderr_u32(status)) { case .ok(_) : { true } case .err(_) : { false } };
    let bool f = switch (io::write_stderr(" os_code=")) { case .ok(_) : { true } case .err(_) : { false } };
    let bool g = switch (io::write_stderr_u32(os_code)) { case .ok(_) : { true } case .err(_) : { false } };
    let bool h = switch (io::write_stderr(")\n")) { case .ok(_) : { true } case .err(_) : { false } };
    return 1;
}
@effects(console)
pub fn i32 text(ref str line) {
    let bool written = switch (io::write(line)) { case .ok(n) : { n == line.len } case .err(_) : { false } };
    if (written) { return 0; }
    let i32 code = error("task-manager-cli", "stdout write failed; mutation may already be saved");
    return 3;
}
@effects(console)
pub fn i32 response(ref bytes::Buffer buffer) {
    return switch (utf8::validate_utf8(bytes::view(buffer))) {
        case .ok(line) : { text(line) }
        case .err(_) : { let i32 code = error("task-manager-cli", "cannot format response"); 3 }
    };
}
