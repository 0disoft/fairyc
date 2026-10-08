module hir_validate_control;
import hir_model as h;
import std::bytes as bytes;
import std::vec as v;

pub fn h::Problem check(ref bytes::Buffer source, ref v::Vec<h::Binding> bindings, ref v::Vec<h::Expression> expressions, ref v::Vec<h::Statement> statements, ref v::Vec<h::Block> blocks, ref v::Vec<usize> edges){
    let usize ns=bytes::len(source);
    let usize nb=v::len<h::Binding>(bindings);
    let usize ne=v::len<h::Expression>(expressions);
    let usize nt=v::len<h::Statement>(statements);
    let usize nk=v::len<h::Block>(blocks);
    let usize na=v::len<usize>(edges);
    var usize i=0;
    i = 0;
    while (i < nt) {
        let h::Statement s = *v::at<h::Statement>(statements,i);
        if (s.start > s.end || s.end > ns || s.expression > ne || s.target > ne || s.binding > nb || s.body > nk || s.initial > nt || s.post > nt || s.loop_target > nt || s.scope == 0) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        if (h::statement_code(s.kind) == h::statement_code(.Break) || h::statement_code(s.kind) == h::statement_code(.Continue)) {
            if (s.loop_target == 0 || s.loop_target > i) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
            let h::Statement target = *v::at<h::Statement>(statements,s.loop_target - 1);
            if (h::statement_code(target.kind) != h::statement_code(.While) && h::statement_code(target.kind) != h::statement_code(.For)) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        }
        if (h::statement_code(s.kind) == h::statement_code(.Binding) && (s.binding == 0 || s.expression == 0)) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        if (h::statement_code(s.kind) == h::statement_code(.Assignment) && (s.target == 0 || s.expression == 0 || h::operator_code(s.operator) == h::operator_code(.None))) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        if ((h::statement_code(s.kind) == h::statement_code(.While) || h::statement_code(s.kind) == h::statement_code(.For)) && s.body == 0) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        if ((h::statement_code(s.kind) == h::statement_code(.Assert) || h::statement_code(s.kind) == h::statement_code(.While)) && (s.expression == 0 || h::type_code(v::at<h::Expression>(expressions,s.expression - 1).type) != h::type_code(.Bool))) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        if (h::statement_code(s.kind) == h::statement_code(.For) && s.expression != 0 && h::type_code(v::at<h::Expression>(expressions,s.expression - 1).type) != h::type_code(.Bool)) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        if (h::statement_code(s.kind) == h::statement_code(.Defer) && (s.expression == 0 || h::expression_code(v::at<h::Expression>(expressions,s.expression - 1).kind) != h::expression_code(.Call) || h::type_code(v::at<h::Expression>(expressions,s.expression - 1).type) != h::type_code(.Void))) { return h::Problem { .code = 4, .start = s.start, .end = s.end }; }
        i += 1;
    }
    i = 0;
    while (i < nk) {
        let h::Block k = *v::at<h::Block>(blocks,i);
        if (k.start > k.end || k.end > ns || k.owner == 0 || k.owner > nb || k.scope == 0 || k.parent_scope == 0 || k.edge_start > na || k.edge_count > na - k.edge_start || k.tail > ne || !h::function_kind(v::at<h::Binding>(bindings,k.owner - 1).kind)) { return h::Problem { .code = 4, .start = k.start, .end = k.end }; }
        var usize j = 0; while (j < k.edge_count) { let usize id = *v::at<usize>(edges,k.edge_start + j); if (id == 0 || id > nt) { return h::Problem { .code = 4, .start = k.start, .end = k.end }; } j += 1; } i += 1;
    }
    return h::Problem{.code=0,.start=0,.end=0};
}
