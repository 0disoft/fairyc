module main;

fn i32 main() {
    let ref str text = "fairy";
    let ref str empty_text = "";
    let ref u8[] bytes = b"fairy";
    let ref u8[] empty_bytes = b"";
    return if (text.len == 5 && bytes.len == 5 && !text.is_empty && empty_text.is_empty && !bytes.is_empty && empty_bytes.is_empty) { 42 } else { 1 };
}
