module examples::windows_output::main;

import examples::windows_output::windows_console as console;

@effects(console)
fn i32 main() {
    let ref str text = "Fairy C says hi\n";
    let ref u8[] bytes = b"\x00A";
    return if (console::stdout_contract_edge_states()) {
        switch (console::stdout_write_text(text)) {
            case .ok(written) : { if (written == text.len &&
                console::byte_view_ready(bytes)) { 42 } else { 1 } }
            case .err(_) : { 2 }
        }
    } else {
        3
    };
}
