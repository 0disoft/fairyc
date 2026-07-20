module examples::windows_output::main;

import examples::windows_output::windows_console as console;

@effects(console)
fn main() -> i32 {
    let text: &str = "Fairy C says hi\n";
    let bytes: &[u8] = b"\x00A";
    if console::stdout_contract_edge_states() &&
        console::stdout_write_text(text) &&
        console::byte_view_ready(bytes) { 42 } else { 1 }
}
