module main;
fn i64 add_fee(i64 amount, i64 fee) { return amount + fee; }
fn i64 total() {
    var i64 amount = add_fee(4000000000, 100);
    amount += 25;
    return amount;
}
fn bool above_limit(i64 amount) { return 4000000000 < amount; }
