module c_emit;
import mir_model as m;
import hir_shapes as shapes;
import mir_verify as verify;
import c_text as w;
import c_scalar as scalar;
import c_guard as guard;
import c_flow as flow;
import c_nominal as nominal;
import c_arrays as arrays;
import c_literals as literals;
import std::bytes as bytes;

fn bool signature(ref m::Program p, ref flow::Plan plan, m::Function f, ref mut bytes::Buffer out, ref mut w::State s) {
    if (flow::stops(plan,f.target) && !w::text(out,s,b"_Noreturn ")) { return false; }
    if (!w::type(out,s,if (flow::stops(plan,f.target)) { 7 } else { m::target_at(p,f.target).result }) || !w::text(out,s,b" ") || !w::function(out,s,f.target) || !w::text(out,s,b"(")) { return false; }
    if (f.parameter_count == 0) { if (!w::text(out,s,b"void")) { return false; } }
    var usize i = 0; while (i < f.parameter_count) {
        let usize id = m::edge_at(p,f.parameter_start + i);
        if (i != 0 && !w::text(out,s,b",")) { return false; }
        if (!w::type(out,s,m::local_at(p,id).type) || !w::text(out,s,b" ") || !w::local(out,s,id)) { return false; } i += 1;
    }
    return w::text(out,s,b")");
}
fn bool function(ref m::Program p, ref flow::Plan plan, m::Function f, ref mut bytes::Buffer out, ref mut w::State s) {
    if (!signature(p,plan,f,out,s) || !w::text(out,s,b"{\n")) { return false; }
    var usize id = f.local_start + 1; while (id <= f.local_start + f.local_count) {
        let m::Local l = m::local_at(p,id);
        if (!l.parameter && (!w::text(out,s,b"    ") || !w::type(out,s,l.type) || !w::text(out,s,b" ") || !w::local(out,s,id) || !w::text(out,s,if (l.type >= 256 || l.type == 9 || l.type == 10 || l.type == 11) { b"={0};\n" } else { b"=0;\n" }))) { return false; }
        if (!w::text(out,s,b"    (void)") || !w::local(out,s,id) || !w::text(out,s,b";\n")) { return false; } id += 1;
    }
    var bool[7] divisors = [false;7]; var bool trapped = false;
    id = f.block_start + 1; while (id <= f.block_start + f.block_count) {
        let m::Block b = m::block_at(p,id); if (!flow::live(plan,id)) { id += 1; continue; } trapped = trapped || b.kind == 6;
        var usize op = b.first; while (op != 0) { let m::Operation o = m::operation_at(p,op); if (o.kind == 4 && (o.operator == 6 || o.operator == 7)) { divisors[o.owner_type as usize] = true; } op = o.next; }
        id += 1;
    }
    if (trapped && !w::text(out,s,b"    uint32_t fy_pending_trap=UINT32_C(7);\n")) { return false; }
    id = 1; while (id <= 5) {
        if (divisors[id] && (!w::text(out,s,b"    volatile ") || !w::type(out,s,if (id == 1) { 1 } else if (id == 2) { 2 } else if (id == 3) { 3 } else if (id == 4) { 4 } else { 5 })
            || !w::text(out,s,b" ") || !w::text(out,s,b"fy_div_") || !w::number(out,s,id) || !w::text(out,s,b"=0;\n"))) { return false; } id += 1;
    }
    if (!w::text(out,s,b"    ") || !w::jump(out,s,f.entry)) { return false; }
    id = f.block_start + 1; while (id <= f.block_start + f.block_count) {
        let m::Block b = m::block_at(p,id);
        if (!flow::live(plan,id)) { id += 1; continue; }
        if (!w::text(out,s,b"fy_b_") || !w::number(out,s,id) || !w::text(out,s,b":;\n")) { return false; }
        var usize op = b.first; while (op != 0) { if (!scalar::operation(p,op,out,s)) { return false; } op = m::operation_at(p,op).next; }
        if (b.kind == 1) {
            if (!w::text(out,s,b"    return") || (b.value != 0 && (!w::text(out,s,b" ") || !w::local(out,s,b.value))) || !w::text(out,s,b";\n")) { return false; }
        } else if (b.kind == 2) { if (!w::text(out,s,b"    ") || !w::jump(out,s,b.target)) { return false; } }
        else if (flow::selected(p,b) != 0) {
            if (b.kind == 4 && flow::selected(p,b) == b.alternative && (!w::text(out,s,b"    fy_pending_trap=UINT32_C(")
                || !w::number(out,s,if (flow::failure(p,b) != 0) { flow::failure(p,b) as usize } else { 5 }) || !w::text(out,s,b");\n"))) { return false; }
            if (!w::text(out,s,b"    ") || !w::jump(out,s,flow::selected(p,b))) { return false; }
        }
        else if (b.kind == 3) {
            if (!w::text(out,s,b"    if(") || !w::local(out,s,b.value) || !w::text(out,s,b"){ ") || !w::jump(out,s,b.target) || !w::text(out,s,b"}else{ ") || !w::jump(out,s,b.alternative) || !w::text(out,s,b"}\n")) { return false; }
        } else if (b.kind == 4) { if (!guard::emit(out,s,b)) { return false; } }
        else if (b.kind == 5) {
            if (!w::text(out,s,b"    ")) { return false; }
            if (b.destination != 0 && !flow::stops(plan,b.callee) && (!w::local(out,s,b.destination) || !w::text(out,s,b"="))) { return false; }
            if (!w::function(out,s,b.callee) || !w::text(out,s,b"(")) { return false; }
            var usize j = 0; while (j < b.argument_count) {
                if (j != 0 && !w::text(out,s,b",")) { return false; }
                if (!w::local(out,s,m::edge_at(p,b.argument_start + j))) { return false; } j += 1;
            }
            if (!w::text(out,s,b");\n") || (!flow::stops(plan,b.callee) && (!w::text(out,s,b"    ") || !w::jump(out,s,b.target)))) { return false; }
        } else if (!w::text(out,s,if (b.kind == 6) { b"    fy_runtime_trap(fy_pending_trap);\n" } else { b"    fy_runtime_trap(UINT32_C(7));\n" })) { return false; }
        id += 1;
    }
    return w::text(out,s,b"}\n");
}
// Verified scalar/nominal C11 transport for 64-bit Windows/Linux. No source identifiers
// or paths are interpolated; all C symbols come from private numeric arena IDs.
pub fn u32 render(ref m::Program p, ref mut bytes::Buffer out) {
    let m::Problem problem = verify::check(p); if (problem.code != 0) { return problem.code; }
    var usize entry = 0; var usize id = 1;
    while (id <= m::target_count(p)) {
        let m::Target t = m::target_at(p,id);
        if (bytes::equal(m::target_text(p,id),b"main")) {
            if (entry != 0 || t.result != 1 || t.signature_count != 0) { return 1; } entry = id;
        }
        id += 1;
    }
    if (entry == 0) { return 1; }
    let flow::Plan plan = switch(flow::build(p)) { case .ok(value): { value } case .err(code): { return code; } };
    var w::State s = w::State { .code = 0 };
    if (!w::text(out,ref mut s,if (shapes::has_results(m::nominal(p))) { b"/* fairyc.candidate-result-c/v1; C11, 64-bit transport */\n" } else if (shapes::has_options(m::nominal(p))) { b"/* fairyc.candidate-option-c/v1; C11, 64-bit transport */\n" } else if (shapes::has_payloads(m::nominal(p))) { b"/* fairyc.candidate-enum-payload-c/v1; C11, 64-bit transport */\n" } else if (shapes::has_enums(m::nominal(p))) { b"/* fairyc.candidate-enum-c/v1; C11, 64-bit transport */\n" } else if (m::has_views(p)) { b"/* fairyc.candidate-view-c/v1; C11, 64-bit transport */\n" } else if (m::has_literals(p)) { b"/* fairyc.candidate-literal-c/v1; C11, 64-bit transport */\n" } else if (arrays::present(p)) { b"/* fairyc.candidate-array-c/v1; C11, 64-bit transport */\n" } else if (shapes::count(m::nominal(p)) != 0) { b"/* fairyc.candidate-nominal-c/v1; C11, 64-bit transport */\n" } else { b"/* fairyc.candidate-scalar-c/v1; C11, 64-bit transport */\n" })
        || !w::text(out,ref mut s,b"#include <stdint.h>\n#include <limits.h>\n#include <string.h>\n_Static_assert(CHAR_BIT==8 && UINTPTR_MAX==UINT64_MAX,\"requires 64-bit byte-addressed target\");\n_Static_assert(INT32_MIN==(-INT32_MAX-1) && INT64_MIN==(-INT64_MAX-1),\"requires two's complement signed integers\");\n_Noreturn void fy_runtime_trap(uint32_t category);\n")
        || !literals::definitions(p,out,ref mut s) || !arrays::definitions(p,out,ref mut s) || !nominal::definitions(p,out,ref mut s) || !scalar::helpers(p,ref plan,out,ref mut s)) { return s.code; }
    id = 1; while (id <= m::function_count(p)) { if (!signature(p,ref plan,m::function_at(p,id),out,ref mut s) || !w::text(out,ref mut s,b";\n")) { return s.code; } id += 1; }
    id = 1; while (id <= m::function_count(p)) { if (!function(p,ref plan,m::function_at(p,id),out,ref mut s)) { return s.code; } id += 1; }
    if (!w::text(out,ref mut s,if (flow::stops(ref plan,entry)) { b"int32_t fy_candidate_entry(void){" } else { b"int32_t fy_candidate_entry(void){return " }) || !w::function(out,ref mut s,entry) || !w::text(out,ref mut s,b"();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n")) { return s.code; }
    return 0;
}
