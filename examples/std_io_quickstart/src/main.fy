module main;

import std::io as io;

@effects(console)
fn i32 main() {
    let ref str message = "Fairy C says hi\n";
    let ref str warning = "Fairy C warns too\n";
    return switch (io::write(message)) {
        case .ok(written) : {
            if (written == message.len) {
                switch (io::write_stderr(warning)) {
                    case .ok(error_written) : {
                        if (error_written == warning.len) { 0 } else { 4 }
                    }
                    case .err(error) : {
                        if (error.status != 0 && error.written <= warning.len) { 5 } else { 6 }
                    }
                }
            } else {
                1
            }
        }
        case .err(error) : {
            if (error.status != 0 && error.written <= message.len) { 2 } else { 3 }
        }
    };
}
