module arithmetic;
import diagnostics as diagnostics;

pub struct Stack { i64[128] values; u32[128] operators; usize[128] offsets; usize value_count; usize operator_count; }
pub fn Stack empty() {
    return Stack { .values = [0; 128], .operators = [0; 128], .offsets = [0; 128], .value_count = 0, .operator_count = 0 };
}
pub fn bool push_value(ref mut Stack stack, i64 value) {
    let usize slot = stack.value_count;
    if (slot == 128) { return false; }
    stack.values[slot] = value;
    stack.value_count = slot + 1;
    return true;
}
pub fn bool push_operator(ref mut Stack stack, u32 kind, usize offset) {
    let usize slot = stack.operator_count;
    if (slot == 128) { return false; }
    stack.operators[slot] = kind;
    stack.offsets[slot] = offset;
    stack.operator_count = slot + 1;
    return true;
}
pub fn u32 top(ref Stack stack) {
    let usize count = stack.operator_count;
    return if (count == 0) { 0 } else { stack.operators[count - 1] };
}
fn u32 precedence(u32 kind) { return if (kind == 6) { 2 } else if (kind == 3) { 1 } else { 0 }; }
fn Result<bool, diagnostics::Problem> reduce(ref mut Stack stack) {
    let usize operators = stack.operator_count;
    let usize values = stack.value_count;
    if (operators == 0 || values < 2) { return .err(diagnostics::problem(8, 0)); }
    let usize operator_slot = operators - 1;
    let u32 kind = stack.operators[operator_slot];
    let usize offset = stack.offsets[operator_slot];
    let usize right_slot = values - 1;
    let usize left_slot = right_slot - 1;
    let i64 left = stack.values[left_slot];
    let i64 right = stack.values[right_slot];
    var i64 result = 0;
    if (kind == 3) {
        if (right > 9223372036854775807 - left) { return .err(diagnostics::problem(10, offset)); }
        result = left + right;
    } else if (kind == 6) {
        if (right != 0 && left > 9223372036854775807 / right) { return .err(diagnostics::problem(10, offset)); }
        result = left * right;
    } else { return .err(diagnostics::problem(8, offset)); }
    stack.values[left_slot] = result;
    stack.value_count = right_slot;
    stack.operator_count = operator_slot;
    return .ok(true);
}

/// Reduce earlier operators of equal or higher precedence, stopping at '('.
/// Incoming 0 finishes the current group. All operations remain checked i64.
pub fn Result<bool, diagnostics::Problem> reduce_for(ref mut Stack stack, u32 incoming) {
    while (stack.operator_count > 0) {
        let u32 previous = top(stack);
        if (previous == 7 || precedence(previous) < precedence(incoming)) { break; }
        discard switch(reduce(stack)) { case .ok(value): { value } case .err(code): { return .err(code); } };
    }
    return .ok(true);
}
