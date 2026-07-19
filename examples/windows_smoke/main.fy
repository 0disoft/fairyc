module examples::windows_smoke::main;

fn main() -> i32 {
    let empty: &str = "";
    let text: &str = "Fairy λ";
    if unsafe { text as *u8 } != null { 42 } else { 1 }
}
