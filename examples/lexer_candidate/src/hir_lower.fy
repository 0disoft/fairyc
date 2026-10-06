module hir_lower;
import hir_model as h;
import hir_view_check as views;
import hir_state as state;
import hir_shapes as shapes;
import hir_shape_lower as shape_lower;
import type_check as typing;
import literal_types as literals;
import name_resolution as names;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn bool prepare(ref typing::Typed input, usize module_index, ref mut v::Vec<h::Binding> bindings, ref mut v::Vec<h::Signature> signatures, ref mut v::Vec<state::Link> links, ref mut state::Context c) {
    let ref names::Resolved resolved = typing::resolved(input);
    let ref syntax::Parsed parsed = names::parsed(resolved);
    if (syntax::source_view(parsed).len > 32768 || syntax::declaration_count(parsed) > 512 || names::symbol_count(resolved) > 1152 || syntax::expression_count(parsed) > 4096 || syntax::entry_count(parsed) > 1024 || syntax::statement_count(parsed) > 8192 || names::reference_count(resolved) > 4096) { discard state::fail(c,2,0,syntax::source_view(parsed).len); return false; }
    let usize length = if (syntax::expression_count(parsed) > 1024) { syntax::expression_count(parsed) } else { 1024 };
    var usize i = 0;
    while (i < length) { if (!v::push<state::Link>(links,state::Link { .use = 0, .entry = 0, .function = 0 }).is_ok) { discard state::fail(c,3,0,0); return false; } i += 1; }
    i = 1;
    while (i <= names::symbol_count(resolved)) {
        let names::Symbol s = names::symbol_at(resolved,i);
        let h::Type type = state::type(typing::symbol_type(input,i));
        if (s.kind < 1 || s.kind > 7 || h::type_code(type) == h::type_code(.Invalid) || h::type_code(type) == h::type_code(.Callable) || s.start > s.end || s.end > syntax::source_view(parsed).len || s.owner > names::symbol_count(resolved)) { discard state::fail(c,4,s.start,s.end); return false; }
        var usize target_module = module_index; var usize declaration = s.declaration;
        if (s.kind == 5) { let signature = names::external_signature(resolved,i); target_module = signature.module_index; declaration = signature.declaration; }
        let h::Binding binding = h::Binding { .kind = state::binding_kind(s.kind), .type, .start = s.start, .end = s.end, .owner = s.owner, .scope = s.scope, .module_index = target_module, .declaration, .signature_start = 0, .signature_count = 0 };
        if (!v::push<h::Binding>(bindings,binding).is_ok) { discard state::fail(c,3,s.start,s.end); return false; }
        if (s.entry != 0) {
            if (s.entry > length || state::entry(links,s.entry) != 0) { discard state::fail(c,4,s.start,s.end); return false; }
            let ref mut state::Link link = v::at_mut<state::Link>(links,s.entry - 1); link.entry = i;
        }
        if (s.kind == 1) {
            if (s.declaration == 0 || s.declaration > syntax::declaration_count(parsed) || state::function(links,s.declaration) != 0 || s.owner != 0 || s.scope != 0) { discard state::fail(c,4,s.start,s.end); return false; }
            let ref mut state::Link link = v::at_mut<state::Link>(links,s.declaration - 1); link.function = i;
        }
        if ((s.kind == 2 || s.kind == 3 || s.kind == 4) && (s.owner == 0 || s.scope == 0 || s.entry == 0 || h::type_code(type) == h::type_code(.Void))) { discard state::fail(c,4,s.start,s.end); return false; }
        i += 1;
    }
    i = 1;
    while (i <= v::len<h::Binding>(bindings)) {
        let names::Symbol s = names::symbol_at(resolved,i); let h::Binding b = *v::at<h::Binding>(bindings,i - 1);
        if (b.owner != 0 && h::binding_code(v::at<h::Binding>(bindings,b.owner - 1).kind) != h::binding_code(.Function)) { discard state::fail(c,4,b.start,b.end); return false; }
        let usize start = v::len<h::Signature>(signatures);
        if (s.kind == 1) {
            let syntax::Declaration function = syntax::declaration_at(parsed,s.declaration - 1); var usize entry = function.entry_start;
            while (entry < function.entry_start + function.entry_count) {
                if (syntax::entry_at(parsed,entry).kind == 1) {
                    let usize id = state::entry(links,entry + 1);
                    if (id == 0 || h::binding_code(v::at<h::Binding>(bindings,id - 1).kind) != h::binding_code(.Parameter) || v::at<h::Binding>(bindings,id - 1).owner != i) { discard state::fail(c,4,function.start,function.end); return false; }
                    if (v::len<h::Signature>(signatures) >= 32768) { discard state::fail(c,2,0,0); return false; }
                    if (!v::push<h::Signature>(signatures,h::Signature { .type = v::at<h::Binding>(bindings,id - 1).type }).is_ok) { discard state::fail(c,3,0,0); return false; }
                }
                entry += 1;
            }
        } else if (s.kind == 5) {
            let signature = names::external_signature(resolved,i); var usize parameter = 0;
            if (signature.parameter_count > 32768 - v::len<h::Signature>(signatures) || signature.declaration == 0) { discard state::fail(c,2,b.start,b.end); return false; }
            while (parameter < signature.parameter_count) {
                let h::Type type = state::type(names::external_parameter(resolved,signature.parameter_start + parameter));
                if (h::type_code(type) == h::type_code(.Invalid) || h::type_code(type) == h::type_code(.Void) || h::type_code(type) == h::type_code(.Callable)) { discard state::fail(c,4,b.start,b.end); return false; }
                if (!v::push<h::Signature>(signatures,h::Signature { .type }).is_ok) { discard state::fail(c,3,b.start,b.end); return false; } parameter += 1;
            }
        }
        let ref mut h::Binding stored = v::at_mut<h::Binding>(bindings,i - 1); stored.signature_start = start; stored.signature_count = v::len<h::Signature>(signatures) - start;
        i += 1;
    }
    i = 0;
    while (i < names::reference_count(resolved)) {
        let names::Reference use = names::reference_at(resolved,i);
        if (use.expression == 0 || use.expression > syntax::expression_count(parsed) || use.symbol == 0 || use.symbol > v::len<h::Binding>(bindings) || state::use(links,use.expression) != 0) { discard state::fail(c,4,use.start,use.end); return false; }
        let ref mut state::Link link = v::at_mut<state::Link>(links,use.expression - 1); link.use = use.symbol; i += 1;
    }
    return true;
}

fn usize expression(ref typing::Typed input, ref v::Vec<h::Binding> bindings, ref v::Vec<state::Link> links, ref mut state::Context c, ref mut v::Vec<h::Expression> expressions, ref mut v::Vec<h::Statement> statements, ref mut v::Vec<h::Block> blocks, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(input));
    if (id > syntax::expression_count(parsed)) { return state::fail(c,4,0,0); }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return state::fail(c,2,node.start,node.end); }
    if (node.kind == 3) { return expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1); }
    var h::Expression e = state::expression(node.start,node.end);
    let h::Type type = state::type(typing::expression_type(input,id));
    var h::ExpressionKind kind = .Integer; var h::Operator operator = .None;
    if (h::type_code(type) == h::type_code(.Invalid)) { return state::fail(c,4,node.start,node.end); }
    if (node.kind == 1 || (node.kind == 4 && typing::signed_minimum(input,id))) {
        if (h::type_code(type) == 9 || h::type_code(type) == 10) {
            kind = .Literal; let ref u8[] cooked = syntax::literal_cooked(parsed,id);
            e.payload_start = bytes::len(payload); let u32 code = literals::decode(payload,cooked);
            if (code != 0) { return state::fail(c,code,node.start,node.end); } e.payload_end = bytes::len(payload);
        } else if (h::type_code(type) == h::type_code(.Bool)) { kind = .Boolean; e.boolean = bytes::equal(syntax::expression_text(parsed,id),b"true"); }
        else {
            kind = .Integer; e.payload_start = bytes::len(payload);
            let usize literal = if (node.kind == 4) { node.left } else { id };
            if (!state::integer(payload,syntax::expression_text(parsed,literal),node.kind == 4,c)) { return 0; }
            e.payload_end = bytes::len(payload);
        }
    } else if (node.kind == 2) {
        kind = .Reference; e.binding = state::use(links,id);
        if (e.binding == 0) { return state::fail(c,4,node.start,node.end); }
        let h::Binding b = *v::at<h::Binding>(bindings,e.binding - 1);
        if (!state::visible(c,b) || h::type_code(b.type) != h::type_code(type) || h::binding_code(b.kind) == h::binding_code(.Function) || h::binding_code(b.kind) == h::binding_code(.External)) { return state::fail(c,4,node.start,node.end); }
    } else if (node.kind == 4 || node.kind == 5) {
        kind = if (node.kind == 4) { .Unary } else { .Binary }; operator = state::operator(syntax::expression_text(parsed,id));
        if (h::operator_code(operator) == h::operator_code(.None)) { return state::fail(c,4,node.start,node.end); }
        e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        if (node.kind == 5) { e.right = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.right,depth + 1); }
    } else if (node.kind == 6) {
        kind = .Call; e.binding = state::use(links,node.left);
        if (e.binding == 0) { return state::fail(c,4,node.start,node.end); }
        let h::Binding callee = *v::at<h::Binding>(bindings,e.binding - 1);
        if ((h::binding_code(callee.kind) != h::binding_code(.Function) && h::binding_code(callee.kind) != h::binding_code(.External)) || h::type_code(callee.type) != h::type_code(type)) { return state::fail(c,4,node.start,node.end); }
        e.edge_count = syntax::call_argument_count(parsed,id);
        if (callee.signature_count != e.edge_count) { return state::fail(c,4,node.start,node.end); }
        e.edge_start = state::reserve_edges(edges,e.edge_count,c); if (c.problem.code != 0) { return 0; }
        var usize i = 0;
        while (i < e.edge_count) {
            let usize argument = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,i),depth + 1);
            if (argument == 0) { return 0; } state::set_edge(edges,e.edge_start + i,argument); i += 1;
        }
    } else if (node.kind == 24 || node.kind == 29) {
        kind = .Array; e.boolean = node.kind == 29;
        e.edge_count = if (e.boolean) { 1 } else { syntax::array_element_count(parsed,id) };
        e.edge_start = state::reserve_edges(edges,e.edge_count,c); if (c.problem.code != 0) { return 0; }
        var usize i = 0; while (i < e.edge_count) {
            let usize child = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,if (e.boolean) { node.left } else { syntax::array_element(parsed,id,i) },depth + 1);
            if (child == 0) { return 0; } state::set_edge(edges,e.edge_start + i,child); i += 1;
        }
    } else if (node.kind == 21 || node.kind == 22) {
        e.boolean = node.kind == 22;
        kind = .Borrow; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 26) {
        kind = .Slice; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        let syntax::Expression bounds = syntax::expression_at(parsed,node.right);
        if (bounds.left != 0) { e.right = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,bounds.left,depth + 1); }
        else {
            let usize start = bytes::len(payload); if (!bytes::append_byte(payload,48).is_ok) { return state::fail(c,3,node.start,node.end); }
            e.right = state::push_expression(expressions,h::Expression { .kind=.Integer,.type=.Usize,.operator=.None,.start=node.start,.end=node.end,.left=0,.right=0,.alternative=0,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=start,.payload_end=start + 1,.boolean=false },c);
        }
        if (bounds.right != 0) { e.alternative = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,bounds.right,depth + 1); }
        else { e.alternative = state::push_expression(expressions,h::Expression { .kind=.Length,.type=.Usize,.operator=.None,.start=node.start,.end=node.end,.left=e.left,.right=0,.alternative=0,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=false },c); e.boolean = true; }
    } else if (node.kind == 9) {
        kind = .Index; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        e.right = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.right,depth + 1);
    } else if (node.kind == 8 && typing::field_target(input,id) == 0) {
        kind = .Length; e.boolean = bytes::equal(syntax::expression_text(parsed,id),b"is_empty"); e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 8) {
        kind = .Field; e.binding = typing::field_target(input,id);
        e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 10) {
        kind = .Construct; e.binding = state::use(links,node.left);
        if (e.binding == 0 || h::binding_code(v::at<h::Binding>(bindings,e.binding - 1).kind) != 6) { return state::fail(c,4,node.start,node.end); }
        e.edge_count = syntax::record_field_count(parsed,id);
        e.edge_start = state::reserve_edges(edges,e.edge_count,c); if (c.problem.code != 0) { return 0; }
        var usize i = 0;
        while (i < e.edge_count) {
            let usize entry = syntax::record_field(parsed,id,i); let syntax::Expression f = syntax::expression_at(parsed,entry);
            let usize value = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,f.left,depth + 1);
            let h::Expression initializer = h::Expression { .kind = .Initializer, .type = state::type(typing::expression_type(input,entry)), .operator = .None, .start = f.start, .end = f.end, .left = value, .right = 0, .alternative = 0, .block = 0, .binding = typing::field_target(input,entry), .edge_start = 0, .edge_count = 0, .payload_start = 0, .payload_end = 0, .boolean = false };
            let usize child = state::push_expression(expressions,initializer,c);
            if (child == 0) { return 0; } state::set_edge(edges,e.edge_start + i,child); i += 1;
        }
    } else if (node.kind == 28) {
        kind=.Variant;e.binding=typing::field_target(input,id);
    } else if (node.kind == 15) {
        kind=.Switch;e.left=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth+1);
        var usize link=node.right;while(link!=0){e.edge_count+=1;link=syntax::expression_at(parsed,link).right;}
        e.edge_start=state::reserve_edges(edges,e.edge_count,c);if(c.problem.code!=0){return 0;}
        link=node.right;var usize remaining=e.edge_count;
        while(link!=0){let usize arm_id=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,arm_id);
            let usize body=block(input,bindings,links,c,expressions,statements,blocks,edges,payload,arm.right,depth+1);
            let h::Expression child=h::Expression{.kind=.Arm,.type=state::type(typing::expression_type(input,arm_id)),.operator=.None,.start=arm.start,.end=arm.end,.block=body,.binding=typing::field_target(input,arm_id),.left=0,.right=0,.alternative=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=false};
            let usize lowered=state::push_expression(expressions,child,c);if(lowered==0){return 0;}remaining-=1;state::set_edge(edges,e.edge_start+remaining,lowered);link=syntax::expression_at(parsed,link).right;
        }
    } else if (node.kind == 12) {
        kind = .Block; e.block = block(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 13) {
        kind = .If; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        let syntax::Expression branches = syntax::expression_at(parsed,node.right);
        e.block = block(input,bindings,links,c,expressions,statements,blocks,edges,payload,branches.left,depth + 1);
        e.alternative = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,branches.right,depth + 1);
    } else { return state::fail(c,4,node.start,node.end); }
    return state::push_expression(expressions,h::Expression { .kind, .type, .operator, .start = e.start, .end = e.end, .left = e.left, .right = e.right, .alternative = e.alternative, .block = e.block, .binding = e.binding, .edge_start = e.edge_start, .edge_count = e.edge_count, .payload_start = e.payload_start, .payload_end = e.payload_end, .boolean = e.boolean },c);
}

fn usize statement(ref typing::Typed input, ref v::Vec<h::Binding> bindings, ref v::Vec<state::Link> links, ref mut state::Context c, ref mut v::Vec<h::Expression> expressions, ref mut v::Vec<h::Statement> statements, ref mut v::Vec<h::Block> blocks, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(input));
    if (id > syntax::statement_count(parsed)) { return state::fail(c,4,0,0); }
    let syntax::Statement source = syntax::statement_at(parsed,id);
    if (depth >= 64) { return state::fail(c,2,source.start,source.end); }
    let h::StatementKind kind = if (source.kind == 2) { .Binding } else if (source.kind == 3) { .Return } else if (source.kind == 4) { .Assignment } else if (source.kind == 8) { .While } else if (source.kind == 14) { .For } else if (source.kind == 9) { .Break } else if (source.kind == 10) { .Continue } else if (source.kind == 11) { .Assert } else if (source.kind == 12) { .Discard } else if (source.kind == 13) { .Defer } else { .Expression };
    let h::Operator operator = if (source.kind == 4) { state::operator(syntax::assignment_operator(parsed,id)) } else { .None };
    var h::Statement s = h::Statement { .kind, .start = source.start, .end = source.end, .scope = state::scope(c), .expression = 0, .target = 0, .binding = 0, .operator, .body = 0, .initial = 0, .post = 0, .loop_target = 0 };
    let usize result = state::push_statement(statements,s,c); if (result == 0) { return 0; }
    if (source.kind == 2) {
         s.binding = state::entry(links,source.entry);
        if (s.binding == 0) { return state::fail(c,4,s.start,s.end); }
        let h::Binding b = *v::at<h::Binding>(bindings,s.binding - 1);
        if ((h::binding_code(b.kind) != h::binding_code(.Let) && h::binding_code(b.kind) != h::binding_code(.Var)) || b.owner != c.owner || b.scope != s.scope || source.expression == 0) { return state::fail(c,4,s.start,s.end); }
        s.expression = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
        if (s.expression != 0 && h::type_code(v::at<h::Expression>(expressions,s.expression - 1).type) != h::type_code(b.type)) { return state::fail(c,4,s.start,s.end); }
    } else if (source.kind == 3) {
         s.expression = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    } else if (source.kind == 4) {

        if (h::operator_code(s.operator) == h::operator_code(.None)) { return state::fail(c,4,s.start,s.end); }
        s.target = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.target,depth + 1);
        s.expression = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    } else if (source.kind == 6 || source.kind == 11 || source.kind == 12 || source.kind == 13) {

        s.expression = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    } else if (source.kind == 9 || source.kind == 10) {
         s.loop_target = c.loop_target;
        if (s.loop_target == 0) { return state::fail(c,4,s.start,s.end); }
    } else if (source.kind == 8 || source.kind == 14) {

        if (source.kind == 14) {
            if (!state::enter(c)) { return 0; } s.scope = state::scope(c);
            let syntax::Statement header = syntax::statement_at(parsed,source.second);
            if (header.kind != 15) { return state::fail(c,4,s.start,s.end); }
            s.initial = statement(input,bindings,links,c,expressions,statements,blocks,edges,payload,header.first,depth + 1);
            s.expression = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
            s.post = statement(input,bindings,links,c,expressions,statements,blocks,edges,payload,header.second,depth + 1);
        } else { s.expression = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1); }
        let usize parent_loop = c.loop_target; c.loop_target = result;
        s.body = block(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.first,depth + 1);
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

fn usize block(ref typing::Typed input, ref v::Vec<h::Binding> bindings, ref v::Vec<state::Link> links, ref mut state::Context c, ref mut v::Vec<h::Expression> expressions, ref mut v::Vec<h::Statement> statements, ref mut v::Vec<h::Block> blocks, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, usize id, usize depth) {
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
        let usize child = statement(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::block_statement(parsed,id,i),depth + 1);
        if (child == 0) { return 0; } state::set_edge(edges,start + i,child); i += 1;
    }
    let usize tail = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,source.expression,depth + 1);
    c.scope_count -= 1;
    if (c.problem.code != 0) { return 0; }
    let h::Block b = h::Block { .start = source.start, .end = source.end, .owner = c.owner, .scope, .parent_scope, .edge_start = start, .edge_count = count, .tail };
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
    var v::Vec<h::Signature> signatures = v::empty<h::Signature>(); var v::Vec<state::Link> links = v::empty<state::Link>();
    var v::Vec<h::Expression> expressions = v::empty<h::Expression>(); var v::Vec<h::Statement> statements = v::empty<h::Statement>();
    var v::Vec<h::Block> blocks = v::empty<h::Block>(); var v::Vec<usize> edges = v::empty<usize>();
    var bytes::Buffer source = bytes::empty(); var bytes::Buffer payload = bytes::empty();
    if (!prepare(ref input,module_index,ref mut bindings,ref mut signatures,ref mut links,ref mut c)) { return .err(c.problem); }
    let shapes::Table nominal = switch(shape_lower::collect(ref input)) { case .ok(value): { value } case .err(code): { return .err(h::Problem { .code, .start = 0, .end = syntax::source_view(parsed).len }); } };
    if (!bytes::append(ref mut source,typing::source_view(ref input)).is_ok) { return .err(h::Problem { .code = 3, .start = 0, .end = 0 }); }
    var usize i = 0;
    while (i < syntax::declaration_count(parsed)) {
        let syntax::Declaration d = syntax::declaration_at(parsed,i);
        if (d.kind == 3) {
            c.owner = state::function(ref links,i + 1); c.loop_target = 0;
            if (c.owner == 0 || !state::enter(ref mut c)) { if (c.problem.code == 0) { discard state::fail(ref mut c,4,d.start,d.end); } return .err(c.problem); }
            let usize scope = state::scope(ref c); let usize start = v::len<usize>(ref edges); var usize entry = d.entry_start; var usize count = 0;
            while (entry < d.entry_start + d.entry_count) {
                if (syntax::entry_at(parsed,entry).kind == 1) {
                    let usize binding = state::entry(ref links,entry + 1);
                    if (binding == 0 || v::at<h::Binding>(ref bindings,binding - 1).scope != scope) { discard state::fail(ref mut c,4,d.start,d.end); return .err(c.problem); }
                    if (!state::edge(ref mut edges,binding,ref mut c)) { return .err(c.problem); } count += 1;
                }
                entry += 1;
            }
            let usize body = block(ref input,ref bindings,ref links,ref mut c,ref mut expressions,ref mut statements,ref mut blocks,ref mut edges,ref mut payload,d.body,0);
            c.scope_count -= 1; if (c.problem.code != 0) { return .err(c.problem); }
            if (!v::push<h::Function>(ref mut functions,h::Function { .binding = c.owner, .start = d.start, .end = d.end, .scope, .public = d.public, .body, .parameter_start = start, .parameter_count = count }).is_ok) { return .err(h::Problem { .code = 3, .start = d.start, .end = d.end }); }
        } else if (d.kind != 1 && d.kind != 2 && d.kind != 4 && d.kind != 11) { return .err(h::Problem { .code = 4, .start = d.start, .end = d.end }); }
        i += 1;
    }
    let h::Program result = switch(h::assemble_nominal(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,module_index,nominal)) { case .ok(p): { p } case .err(code): { return .err(code); } };
    let h::Problem proof = views::check(ref result); if (proof.code != 0) { return .err(proof); } return .ok(result);
}
