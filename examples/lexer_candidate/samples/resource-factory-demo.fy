module main;
resource Amount { i64 value; cleanup {} }
pub fn Result<Amount,i32> from_amount(i64 value) {
    if (value<0) { return .err(1); }
    return .ok(Amount{.value});
}
fn Result<Amount,i32> main() {
    return from_amount(4000000000);
}
