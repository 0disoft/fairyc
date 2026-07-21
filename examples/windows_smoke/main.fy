module examples::windows_smoke::main;

fn i32 main() {
    let ref str empty = "";
    let ref str text = "Fairy λ";
    return if (unsafe { text as ptr u8 } != null) { 42 } else { 1 };
}
