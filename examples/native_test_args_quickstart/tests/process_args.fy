module tests::process_args;

import std::process as process;

@test
fn void program_arguments_are_forwarded() {
    assert(process::arg_count() == 4);

    var u8[8] flag = [0, 0, 0, 0, 0, 0, 0, 0];
    let bool flag_matches = switch (process::arg(1, ref mut flag)) {
        case .ok(read) : {
            read == 8 && flag[0] == 45 && flag[1] == 45 && flag[2] == 108 && flag[3] == 111 &&
                flag[4] == 99 && flag[5] == 107 && flag[6] == 101 && flag[7] == 100
        }
        case .err(_) : { false }
    };
    assert(flag_matches);

    var u8[1] empty = [7];
    let bool empty_matches = switch (process::arg(2, ref mut empty)) {
        case .ok(read) : { read == 0 && empty[0] == 7 }
        case .err(_) : { false }
    };
    assert(empty_matches);

    var u8[8] utf8 = [0, 0, 0, 0, 0, 0, 0, 0];
    let bool utf8_matches = switch (process::arg(3, ref mut utf8)) {
        case .ok(read) : {
            read == 8 && utf8[0] == 237 && utf8[1] == 149 && utf8[2] == 156 &&
                utf8[3] == 234 && utf8[4] == 184 && utf8[5] == 128 &&
                utf8[6] == 206 && utf8[7] == 187
        }
        case .err(_) : { false }
    };
    assert(utf8_matches);
}
