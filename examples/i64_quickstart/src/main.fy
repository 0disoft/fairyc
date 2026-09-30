module main;

import std::bytes as bytes;
import std::io as io;

@effects(console)
fn Result<i32, bytes::BufferError> render() {
    var bytes::Buffer buffer = bytes::with_capacity(128)?;
    let usize a = bytes::append_i64(ref mut buffer, -9223372036854775808)?;
    let usize b = bytes::append(ref mut buffer, b",")?;
    let usize c = bytes::append_i64(ref mut buffer, 9223372036854775807)?;
    let usize d = bytes::append(ref mut buffer, b",")?;
    let usize e = bytes::append_i64(ref mut buffer, 0)?;
    let usize f = bytes::append(ref mut buffer, b",")?;
    let usize g = bytes::append_i64(ref mut buffer, -5000000000)?;
    let usize h = bytes::append(ref mut buffer, b"\n")?;
    let ref u8[] encoded = bytes::view(ref buffer);
    let bool written = switch (io::write_bytes(encoded)) {
        case .ok(count): { count == encoded.len }
        case .err(_): { false }
    };
    return .ok(if (written) { 0 } else { 2 });
}

@effects(console)
fn i32 main() {
    return switch (render()) {
        case .ok(code): { code }
        case .err(_): { 1 }
    };
}
