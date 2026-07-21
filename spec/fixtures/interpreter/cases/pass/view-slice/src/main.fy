module main;

fn i32 main() {
    let ref u8[] bytes = b"fairy"[1..4];
    return if (bytes.len == 3 && bytes[0] == 97 && bytes[2] == 114) { 42 } else { 1 };
}

fn i32 empty_at_end() {
    let ref u8[] bytes = b"fairy"[5..5];
    return if (bytes.is_empty) { 42 } else { 1 };
}

fn i32 start_after_end() {
    let ref u8[] bytes = b"fairy"[4..2];
    return 42;
}

fn i32 end_after_length() {
    let ref u8[] bytes = b"fairy"[0..6];
    return 42;
}

fn i32 omitted_start() {
    let ref u8[] bytes = b"fairy"[..2];
    return if (bytes.len == 2 && bytes[1] == 97) { 42 } else { 1 };
}

fn i32 omitted_end() {
    let ref u8[] bytes = b"fairy"[2..];
    return if (bytes.len == 3 && bytes[0] == 105) { 42 } else { 1 };
}

fn i32 omitted_both() {
    let ref u8[] bytes = b"fairy"[..];
    return if (bytes.len == 5 && bytes[4] == 121) { 42 } else { 1 };
}
