module main;

fn i32 main() {
    let ref str text = "λx";
    let ref u8[] bytes = b"\x00ABC";
    return if (text.len == 3 && bytes.len == 4 && !text.is_empty && !bytes.is_empty && "".is_empty && b"".is_empty) { 42 } else { 1 };
}
