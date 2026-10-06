module tagged_check;
import tagged_types as tagged;
import type_control as control;
import type_state as state;
import name_resolution as names;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Pattern validation and capture facts shared by builtin Option and Result.
// Recursively checking each arm's body remains owned by type_check.
pub struct Pattern { bool first; state::Problem problem; }
pub fn bool first_name(u32 type, ref u8[] name) {
    return bytes::equal(name,if (tagged::result(type)) { b"ok" } else { b"some" });
}
pub fn bool second_name(u32 type, ref u8[] name) {
    return bytes::equal(name,if (tagged::result(type)) { b"err" } else { b"none" });
}
pub fn bool no_value_arm(ref syntax::Parsed p, usize id, u32 type) {
    if (control::assert_arm(p,id)) { return true; }
    if (!tagged::result(type)) { return false; }
    let body = syntax::statement_at(p,id); let usize count = syntax::block_statement_count(p,id);
    return body.expression == 0 && count != 0 && syntax::statement_at(p,syntax::block_statement(p,id,count - 1)).kind == 3;
}
pub fn Pattern pattern(ref names::Resolved resolved, ref mut v::Vec<state::Slot> tables, usize arm_id, u32 type) {
    let ref syntax::Parsed p = names::parsed(resolved);
    let arm = syntax::expression_at(p,arm_id); let pattern = syntax::expression_at(p,arm.left);
    let ref u8[] name = syntax::expression_text(p,arm.left);
    let bool first = first_name(type,name); let bool second = second_name(type,name);
    let bool payload = first || tagged::result(type);
    if (pattern.kind != 16 || pattern.right != 0 || (!first && !second) ||
        payload && pattern.left == 0 || !payload && pattern.left != 0) {
        return Pattern { .first, .problem = state::fail(if (tagged::result(type)) { 38 } else { 34 },pattern.start,pattern.end,0,0) };
    }
    state::set_flags(tables,arm_id - 1,state::small_id(tagged::member(p,type,first)));
    if (payload) {
        let u32 child = tagged::payload_type(type,first);
        state::set_expressions(tables,pattern.left - 1,child);
        state::set_flags(tables,pattern.left - 1,state::small_id(tagged::payload_for(p,type,first)));
        var usize sid = 1;
        while (sid <= names::symbol_count(resolved)) {
            let symbol = names::symbol_at(resolved,sid);
            if (symbol.kind == 8 && symbol.declaration == pattern.left) { state::set_symbols(tables,sid - 1,child); state::set_uses(tables,pattern.left-1,sid); }
            sid += 1;
        }
    }
    return Pattern { .first, .problem = state::ok() };
}
