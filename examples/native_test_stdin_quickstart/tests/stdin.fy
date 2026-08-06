module tests::stdin;

import std::io as io;

@test
@effects(console)
fn void caller_input_is_forwarded() {
    var u8[8] input = [9, 9, 9, 9, 9, 9, 9, 9];
    let bool input_matches = switch (io::read_stdin(ref mut input)) {
        case .ok(read) : {
            read == 7 && input[0] == 70 && input[1] == 97 && input[2] == 105 &&
                input[3] == 114 && input[4] == 121 && input[5] == 0 &&
                input[6] == 10 && input[7] == 9
        }
        case .err(_) : { false }
    };
    assert(input_matches);

    let bool eof_matches = switch (io::read_stdin(ref mut input)) {
        case .ok(read) : { read == 0 && input[7] == 9 }
        case .err(_) : { false }
    };
    assert(eof_matches);
}
