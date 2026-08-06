module tests::timeout;

import std::process as process;

@test
fn void hangs() {
    while (process::arg_count() == 1) {}
}
