module main;

fn ref str title() { return "Fairy C 한글"; }
fn u8 last(ref u8[] data) { return data[data.len - 1]; }

fn i32 main() {
    let ref str text = title();
    let ref u8[] binary = b"\0\xff";
    assert(text.len == 14 && !text.is_empty);
    assert(binary.len == 2 && last(binary) == 255);
    assert("".is_empty && b"".is_empty);
    return 0;
}
