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
fn bool write_number_to_stdout(i32 value, usize expected_length) {
    return switch (io::write_i32(value)) {
        case .ok(written) : { written == expected_length }
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
fn bool echo_stdin() {
    var u8[8] buffer = [0, 0, 0, 0, 0, 0, 0, 0];
    return switch (io::read_stdin(ref mut buffer)) {
        case .ok(read_count) : {
            if (read_count <= buffer.len) {
                let ref u8[] bytes = ref buffer;
                switch (io::write_bytes(bytes[..read_count])) {
                    case .ok(written) : { written == read_count }
                    case .err(_) : { false }
                }
            } else {
                false
            }
        }
        case .err(_) : { false }
    };
}

@effects(console)
fn i32 main() {
    let bool stdout_text_ok = write_text_to_stdout("Fairy C says hi\n");
    let bool stdout_bytes_ok = write_bytes_to_stdout(b"bytes on stdout\n");
    let bool stdout_number_ok = write_number_to_stdout(-2147483647 - 1, 11);
    let bool stdout_number_newline_ok = write_text_to_stdout("\n");
    let bool stderr_text_ok = write_text_to_stderr("Fairy C warns too\n");
    let bool stderr_bytes_ok = write_bytes_to_stderr(b"bytes on stderr\n");
    let bool stdin_ok = echo_stdin();
    return if (stdout_text_ok && stdout_bytes_ok && stdout_number_ok && stdout_number_newline_ok && stderr_text_ok && stderr_bytes_ok && stdin_ok) { 0 } else { 1 };
}
