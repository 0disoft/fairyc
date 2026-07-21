module main;

fn usize text_length(ref str text) {
    return text.len;
}

fn usize byte_length(ref u8[] bytes) {
    return bytes.len;
}

fn i32 main() {
    return if (text_length("λx") == 3 && byte_length(b"\x00ABC") == 4) { 42 } else { 1 };
}
