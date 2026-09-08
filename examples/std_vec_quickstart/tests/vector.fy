module tests::vector;

import main as app;

@test
fn void sum_after_pop_and_clear() {
    assert(app::total() == 42);
}
