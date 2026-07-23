module main;

import std::io as io;

@effects(console)
fn i32 main() {
    let ref str message = "Fairy C says hi\n";
    return switch (io::write(message)) {
        case .ok(written) : { if (written == message.len) { 42 } else { 1 } }
        case .err(error) : {
            if (error.status != 0 && error.written <= message.len) { 2 } else { 3 }
        }
    };
}
