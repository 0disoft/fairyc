module tests::shipping;

import main as app;

@test
fn void applies_free_shipping_at_the_threshold() {
    assert(app::shipping_fee(0) == 3000);
    assert(app::shipping_fee(29999) == 3000);
    assert(app::shipping_fee(30000) == 0);
    assert(app::shipping_fee(30001) == 0);
}
