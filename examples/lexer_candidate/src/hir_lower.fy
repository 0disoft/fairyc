module hir_lower;
import resource_abilities as abilities;
import hir_model as h;
import unsafe_scopes as unsafe_scopes;
import hir_view_check as views;
import hir_state as state;
import hir_shapes as shapes;
import hir_shape_lower as shape_lower;
import type_check as typing;
import name_resolution as names;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import type_state as typing_ids;
import hir_expression_lower as expr_lower;
import hir_statement_lower as stmt_lower;
import hir_bindings_lower as bindings_lower;

pub fn usize block(ref typing::Typed input, ref v::Vec<h::Binding> bindings, ref v::Vec<state::Link> links, ref mut state::Context c, ref mut v::Vec<h::Expression> expressions, ref mut v::Vec<h::Statement> statements, ref mut v::Vec<h::Block> blocks, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, usize id, usize depth) {
    if (c.problem.code != 0) { return 0; }
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(input));
    if (id == 0 || id > syntax::statement_count(parsed)) { return state::fail(c,4,0,0); }
    let syntax::Statement source = syntax::statement_at(parsed,id);
    if (depth >= 64 || v::len<h::Block>(blocks) >= 8192) { return state::fail(c,2,source.start,source.end); }
    if (source.kind != 1) { return state::fail(c,4,source.start,source.end); }
    let usize parent_scope = state::scope(c); if (!state::enter(c)) { return 0; }
    let usize scope = state::scope(c); let usize count = syntax::block_statement_count(parsed,id);
    let usize start = state::reserve_edges(edges,count,c); if (c.problem.code != 0) { return 0; }
    var usize i = 0;
    while (i < count) {
        let usize child = stmt_lower::statement(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::block_statement(parsed,id,i),depth + 1);
        if (child == 0) { return 0; } state::set_edge(edges,start + i,child); i += 1;
    }
    let usize tail = expr_lower::expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    c.scope_count -= 1;
    if (c.problem.code != 0) { return 0; }
    let h::Block b = h::Block { .unsafe_body=unsafe_scopes::block(parsed,id), .start = source.start, .end = source.end, .owner = c.owner, .scope, .parent_scope, .edge_start = start, .edge_count = count, .tail };
    if (!v::push<h::Block>(blocks,b).is_ok) { return state::fail(c,3,b.start,b.end); }
    return v::len<h::Block>(blocks);
}

// Consumes successful Typed. All construction mutates local SDK containers,
// never mutable container fields of a resource. The return owns source bytes,
// canonical literals and executable semantic links; input tables are dropped.
pub fn Result<h::Program,h::Problem> lower(typing::Typed input, usize module_index) {
    if (typing::problem(ref input).code != 0 || names::problem(typing::resolved(ref input)).code != 0) { return .err(h::Problem { .code = 1, .start = 0, .end = typing::source_view(ref input).len }); }
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(ref input));
    if (syntax::problem_count(parsed) != 0) { return .err(h::Problem { .code = 1, .start = 0, .end = syntax::source_view(parsed).len }); }
    var state::Context c = state::empty();
    var v::Vec<h::Function> functions = v::empty<h::Function>(); var v::Vec<h::Binding> bindings = v::empty<h::Binding>();
    var v::Vec<h::Drop> drops=v::empty<h::Drop>();
    var v::Vec<h::Signature> signatures = v::empty<h::Signature>(); var v::Vec<state::Link> links = v::empty<state::Link>();
    var v::Vec<h::Expression> expressions = v::empty<h::Expression>(); var v::Vec<h::Statement> statements = v::empty<h::Statement>();
    var v::Vec<h::Block> blocks = v::empty<h::Block>(); var v::Vec<usize> edges = v::empty<usize>();
    var bytes::Buffer source = bytes::empty(); var bytes::Buffer payload = bytes::empty();
    if (!bindings_lower::prepare(ref input,module_index,ref mut bindings,ref mut signatures,ref mut links,ref mut c)) { return .err(c.problem); }
    let shapes::Table nominal = switch(shape_lower::collect(ref input)) { case .ok(value): { value } case .err(code): { return .err(h::Problem { .code, .start = 0, .end = syntax::source_view(parsed).len }); } };
    if (!bytes::append(ref mut source,typing::source_view(ref input)).is_ok) { return .err(h::Problem { .code = 3, .start = 0, .end = 0 }); }
    var usize i = 0;
    while (i < syntax::declaration_count(parsed)) {
        let syntax::Declaration d = syntax::declaration_at(parsed,i);
        let bool drop_body=d.kind==6 && state::function(ref links,i+1)!=0;
        var usize receiver=0;
        if (d.kind == 3 || drop_body) {
            c.owner = state::function(ref links,i + 1); c.loop_target = 0;
            if (c.owner == 0 || !state::enter(ref mut c)) { if (c.problem.code == 0) { discard state::fail(ref mut c,4,d.start,d.end); } return .err(c.problem); }
            let usize scope = state::scope(ref c); let usize start = v::len<usize>(ref edges); var usize entry = d.entry_start; var usize count = 0;
            while (!drop_body && entry < d.entry_start + d.entry_count) {
                if (syntax::entry_at(parsed,entry).kind == 1) {
                    let usize binding = state::entry(ref links,entry + 1);
                    if (binding == 0 || v::at<h::Binding>(ref bindings,binding - 1).scope != scope) { discard state::fail(ref mut c,4,d.start,d.end); return .err(c.problem); }
                    if (!state::edge(ref mut edges,binding,ref mut c)) { return .err(c.problem); } count += 1;
                }
                entry += 1;
            }
            if(drop_body){var usize n=1;while(n<=v::len<h::Binding>(ref bindings)){let b=*v::at<h::Binding>(ref bindings,n-1);if(h::binding_code(b.kind)==10 && b.owner==c.owner){receiver=n;}n+=1;}
                if(receiver==0 || v::at<h::Binding>(ref bindings,receiver-1).scope!=scope || !state::edge(ref mut edges,receiver,ref mut c)){return .err(h::Problem{.code=4,.start=d.cleanup_start,.end=d.cleanup_end});}count=1;
            }
            let usize body = block(ref input,ref bindings,ref links,ref mut c,ref mut expressions,ref mut statements,ref mut blocks,ref mut edges,ref mut payload,if(drop_body){d.cleanup_body}else{d.body},0);
            c.scope_count -= 1; if (c.problem.code != 0) { return .err(c.problem); }
            if (!v::push<h::Function>(ref mut functions,h::Function { .binding = c.owner, .start = if(drop_body){d.cleanup_start}else{d.start}, .end = if(drop_body){d.cleanup_end}else{d.end}, .scope, .public = !drop_body && d.public, .body, .parameter_start = start, .parameter_count = count }).is_ok) { return .err(h::Problem { .code = 3, .start = d.start, .end = d.end }); }
        } else if (d.kind != 12 && d.kind != 1 && d.kind != 2 && d.kind != 4 && d.kind != 6 && d.kind != 9 && d.kind != 11) { return .err(h::Problem { .code = 4, .start = d.start, .end = d.end }); }
        if(d.kind==6){let u32 type=98303+typing_ids::small_id(i+1);
            if(!v::push<h::Drop>(ref mut drops,h::Drop{.abilities=abilities::mask(parsed,i),.type,.function=if(drop_body){c.owner}else{0},.receiver,.start=d.cleanup_start,.end=d.cleanup_end,.nonempty=drop_body}).is_ok){return .err(h::Problem{.code=3,.start=d.start,.end=d.end});}
        }
        i += 1;
    }
    let h::Program result = switch(h::assemble_resources(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,module_index,nominal,drops)) { case .ok(p): { p } case .err(code): { return .err(code); } };
    let h::Problem proof = views::check(ref result); if (proof.code != 0) { return .err(proof); } return .ok(result);
}
