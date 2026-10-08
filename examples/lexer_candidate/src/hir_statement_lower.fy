module hir_statement_lower;
import hir_model as h;
import hir_state as state;
import hir_byte_loop as byte_loop;
import type_check as typing;
import name_resolution as names;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import hir_lower as lowering;
import hir_expression_lower as expr_lower;

pub fn usize statement(ref typing::Typed input, ref v::Vec<h::Binding> bindings, ref v::Vec<state::Link> links, ref mut state::Context c, ref mut v::Vec<h::Expression> expressions, ref mut v::Vec<h::Statement> statements, ref mut v::Vec<h::Block> blocks, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(input));
    if (id > syntax::statement_count(parsed)) { return state::fail(c,4,0,0); }
    let syntax::Statement source = syntax::statement_at(parsed,id);
    if (depth >= 64) { return state::fail(c,2,source.start,source.end); }
    let h::StatementKind kind = if (source.kind == 2) { .Binding } else if (source.kind == 3) { .Return } else if (source.kind == 4) { .Assignment } else if (source.kind == 8) { .While } else if (source.kind == 14) { .For } else if (source.kind == 9) { .Break } else if (source.kind == 10) { .Continue } else if (source.kind == 11) { .Assert } else if (source.kind == 12) { .Discard } else if (source.kind == 13) { .Defer } else if(source.kind==16){.Burn}else { .Expression };
    let h::Operator operator = if (source.kind == 4) { state::operator(syntax::assignment_operator(parsed,id)) } else { .None };
    var h::Statement s = h::Statement { .kind, .start = source.start, .end = source.end, .scope = state::scope(c), .expression = 0, .target = 0, .binding = 0, .operator, .body = 0, .initial = 0, .post = 0, .loop_target = 0 };
    let usize result = state::push_statement(statements,s,c); if (result == 0) { return 0; }
    if(source.kind==18){
        let usize parent_scope=state::scope(c);if(!state::enter(c)){return 0;}
        let usize view=expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth+1);
        let built=byte_loop::start(source,syntax::statement_at(parsed,source.first).start,view,links,c,expressions,statements,payload);if(built.loop==0){return 0;}
        let usize old_loop=c.loop_target;c.loop_target=built.loop;
        let usize body=lowering::block(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.first,depth+1);c.loop_target=old_loop;
        let usize wrapper=byte_loop::finish(source,built,body,parent_scope,links,c,expressions,statements,blocks,edges);
        c.scope_count-=1;if(wrapper==0){return 0;}s.expression=wrapper;
    }else if (source.kind == 2) {
         s.binding = state::entry(links,source.entry);
        if (s.binding == 0) { return state::fail(c,4,s.start,s.end); }
        let h::Binding b = *v::at<h::Binding>(bindings,s.binding - 1);
        if ((h::binding_code(b.kind) != h::binding_code(.Let) && h::binding_code(b.kind) != h::binding_code(.Var)) || b.owner != c.owner || b.scope != s.scope || source.expression == 0) { return state::fail(c,4,s.start,s.end); }
        s.expression = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
        if (s.expression != 0 && h::type_code(v::at<h::Expression>(expressions,s.expression - 1).type) != h::type_code(b.type)) { return state::fail(c,4,s.start,s.end); }
    } else if (source.kind == 3) {
         s.expression = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    } else if (source.kind == 4) {

        if (h::operator_code(s.operator) == h::operator_code(.None)) { return state::fail(c,4,s.start,s.end); }
        s.target = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.target,depth + 1);
        s.expression = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    } else if (source.kind == 6 || source.kind == 11 || source.kind == 12 || source.kind == 13 || source.kind==16) {

        s.expression = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    } else if (source.kind == 9 || source.kind == 10) {
         s.loop_target = c.loop_target;
        if (s.loop_target == 0) { return state::fail(c,4,s.start,s.end); }
    } else if (source.kind == 8 || source.kind == 14) {

        if (source.kind == 14) {
            if (!state::enter(c)) { return 0; } s.scope = state::scope(c);
            let syntax::Statement header = syntax::statement_at(parsed,source.second);
            if (header.kind != 15) { return state::fail(c,4,s.start,s.end); }
            s.initial = statement(input,bindings,links,c,expressions,statements,blocks,edges,payload,header.first,depth + 1);
            s.expression = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
            s.post = statement(input,bindings,links,c,expressions,statements,blocks,edges,payload,header.second,depth + 1);
        } else { s.expression = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1); }
        let usize parent_loop = c.loop_target; c.loop_target = result;
        s.body = lowering::block(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.first,depth + 1);
        c.loop_target = parent_loop;
        if (source.kind == 14) { c.scope_count -= 1; }
    } else { return state::fail(c,4,s.start,s.end); }
    if (c.problem.code != 0) { return 0; }
    let ref mut h::Statement stored = v::at_mut<h::Statement>(statements,result - 1);
    stored.start = s.start; stored.end = s.end; stored.scope = s.scope;
    stored.expression = s.expression; stored.target = s.target; stored.binding = s.binding;
    stored.body = s.body; stored.initial = s.initial;
    stored.post = s.post; stored.loop_target = s.loop_target;
    return result;
}
