module main;

import lib as lib;

fn i32 main() {
    let lib::Pair pair = lib::Pair { .value = 40 };
    let lib::Token token = lib::token(2);
    return pair.value + 2;
}
