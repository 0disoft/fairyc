module main;

import std::io as io;

@effects(console)
fn bool write_text_to_stdout(ref str value) {
    return switch (io::write(value)) {
        case .ok(written) : { written == value.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_bytes_to_stdout(ref u8[] value) {
    return switch (io::write_bytes(value)) {
        case .ok(written) : { written == value.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_text_to_stderr(ref str value) {
    return switch (io::write_stderr(value)) {
        case .ok(written) : { written == value.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn bool write_bytes_to_stderr(ref u8[] value) {
    return switch (io::write_stderr_bytes(value)) {
        case .ok(written) : { written == value.len }
        case .err(_) : { false }
    };
}

@effects(console)
fn i32 main() {
    let bool stdout_text_ok = write_text_to_stdout("Fairy C says hi\n");
    let bool stdout_bytes_ok = write_bytes_to_stdout(b"bytes on stdout\n");
    let bool stderr_text_ok = write_text_to_stderr("Fairy C warns too\n");
    let bool stderr_bytes_ok = write_bytes_to_stderr(b"bytes on stderr\n");
    return if (stdout_text_ok && stdout_bytes_ok && stderr_text_ok && stderr_bytes_ok) { 0 } else { 1 };
}
