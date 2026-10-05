module mir_lower;
import hir_model as h;
import mir_model as m;
import mir_state as state;
import std::bytes as bytes;
import std::vec as v;

// First lowering unit: sequential scalar code, nested lexical blocks, calls,
// checked operations, assertions, returns and assignments. Structured source
// control/defer is rejected before construction; its CFG lowering follows.
fn bool admitted(ref h::Program input) {
    var usize i = 1;
    while (i <= h::expression_count(input)) {
        let h::Expression e = h::expression_at(input,i); let u32 kind = h::expression_code(e.kind); let u32 op = h::operator_code(e.operator);
        if (kind == 8 || op == 19 || op == 20) { return false; } i += 1;
    }
    i = 1; while (i <= h::statement_count(input)) { let u32 kind = h::statement_code(h::statement_at(input,i).kind); if (kind == 5 || kind == 6 || kind == 7 || kind == 8 || kind == 11) { return false; } i += 1; }
    return true;
}

fn bool prepare(ref h::Program input, ref mut v::Vec<m::Target> targets, ref mut v::Vec<m::Signature> signatures, ref mut state::Context c) {
    var usize id = 1;
    while (id <= h::binding_count(input)) {
        let h::Binding b = h::binding_at(input,id); let u32 kind = h::binding_code(b.kind);
        if (kind == 1 || kind == 5) {
            if (v::len<m::Target>(targets) >= 512 || b.signature_count > 65536 - v::len<m::Signature>(signatures)) { discard state::fail(c,2,b.start,b.end); return false; }
            let usize start = v::len<m::Signature>(signatures); var usize j = 0;
            while (j < b.signature_count) { if (!v::push<m::Signature>(signatures,m::Signature { .type = h::type_code(h::signature_at(input,b.signature_start + j)) }).is_ok) { discard state::fail(c,3,b.start,b.end); return false; } j += 1; }
            if (!v::push<m::Target>(targets,m::Target { .binding = id, .module_index = b.module_index, .declaration = b.declaration, .result = h::type_code(b.type), .signature_start = start, .signature_count = b.signature_count, .start = b.start, .end = b.end, .external = kind == 5 }).is_ok) { discard state::fail(c,3,b.start,b.end); return false; }
            c.targets[id] = v::len<m::Target>(targets);
        }
        id += 1;
    }
    return true;
}
fn u32 trap_mask(u32 operator, u32 type, bool unary) {
    if (unary) { return if (operator == 2) { 1 } else { 0 }; }
    if (operator == 1 || operator == 2 || operator == 5) { return 1; }
    if (operator == 6 || operator == 7) { return if (type == 1 || type == 2) { 6 } else { 2 }; }
    if (operator == 11) { return 9; } if (operator == 12) { return 8; } return 0;
}
fn usize binary(ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut state::Context c, u32 type, u32 owner_type, u32 operator, usize left, usize right, usize start, usize end) {
    let u32 mask = trap_mask(operator,owner_type,false);
    if (mask != 0 && !state::guard(operations,blocks,c,owner_type,operator,left,right,mask,start,end)) { return 0; }
    let usize result = state::local(locals,c,type,0,false,false,start,end);
    if (result == 0 || state::emit(operations,blocks,c,state::operation(4,result,type,owner_type,operator,left,right,start,end)) == 0) { return 0; } return result;
}

fn usize expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    if (id == 0 || c.current == 0 || c.problem.code != 0) { return 0; }
    let h::Expression e = h::expression_at(input,id); let u32 kind = h::expression_code(e.kind); let u32 type = h::type_code(e.type); let u32 operator = h::operator_code(e.operator);
    if (depth >= 64) { return state::fail(c,2,e.start,e.end); }
    if (kind == 1 || kind == 2) {
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(1,result,type,type,0,0,0,e.start,e.end); o.boolean = e.boolean;
        if (kind == 1) {
            let ref u8[] text = h::integer_text(input,id);
            if (text.len > 688128 - bytes::len(payload)) { return state::fail(c,2,e.start,e.end); }
            o.payload_start = bytes::len(payload); if (!bytes::append(payload,text).is_ok) { return state::fail(c,3,e.start,e.end); } o.payload_end = bytes::len(payload);
        }
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 3) {
        let usize original = c.locals[e.binding];
        if (original == 0) { return state::fail(c,4,e.start,e.end); }
        // Freeze source reads before later child expressions can assign/call.
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(2,result,type,type,0,original,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    if (kind == 4 || kind == 5) {
        let usize left = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        if (kind == 5) {
            let usize right = expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            return binary(locals,blocks,operations,c,type,owner_type,operator,left,right,e.start,e.end);
        }
        let u32 mask = trap_mask(operator,owner_type,true);
        if (mask != 0 && !state::guard(operations,blocks,c,owner_type,operator,left,0,mask,e.start,e.end)) { return 0; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(3,result,type,owner_type,operator,left,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    if (kind == 6) {
        let usize callee = c.targets[e.binding]; if (callee == 0) { return state::fail(c,4,e.start,e.end); }
        // Nested calls also use the edge arena. Reserve this argument range first.
        if (e.edge_count > 65536 - v::len<usize>(edges)) { return state::fail(c,2,e.start,e.end); }
        let usize start = v::len<usize>(edges); var usize j = 0;
        while (j < e.edge_count) { if (!state::edge(edges,1,c)) { return 0; } j += 1; }
        j = 0;
        while (j < e.edge_count) {
            let usize argument = expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,e.edge_start + j),depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            { let ref mut usize slot = v::at_mut<usize>(edges,start + j); *slot = argument; } j += 1;
        }
        let usize destination = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let usize continuation = state::new_block(blocks,c,e.start,e.end);
        var m::Block term = state::block(c.function,e.start,e.end); term.kind = 5; term.callee = callee;
        term.destination = destination; term.argument_start = start; term.argument_count = e.edge_count; term.target = continuation;
        if (!state::terminate(blocks,c,term)) { return 0; } c.current = continuation; return destination;
    }
    if (kind == 7) { return block(input,locals,blocks,operations,edges,payload,c,e.block,depth + 1); }
    return state::fail(c,4,e.start,e.end);
}
fn bool statement(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    let h::Statement s = h::statement_at(input,id); let u32 kind = h::statement_code(s.kind);
    if (depth >= 64) { discard state::fail(c,2,s.start,s.end); return false; }
    if (kind == 1 || kind == 3) {
        let usize binding = if (kind == 1) { s.binding } else { h::expression_at(input,s.target).binding };
        let usize destination = c.locals[binding]; if (destination == 0) { discard state::fail(c,4,s.start,s.end); return false; }
        let u32 type = v::at<m::Local>(locals,destination - 1).type; let u32 operator = h::operator_code(s.operator);
        var usize left = 0;
        if (kind == 3 && operator != 21) {
            let h::Expression target = h::expression_at(input,s.target);
            left = expression(input,locals,blocks,operations,edges,payload,c,s.target,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        }
        var usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (kind == 3 && operator != 21) { value = binary(locals,blocks,operations,c,type,type,operator,left,value,s.start,s.end); }
        return state::emit(operations,blocks,c,state::operation(2,destination,type,type,0,value,0,s.start,s.end)) != 0;
    }
    if (kind == 2) {
        let usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        var m::Block term = state::block(c.function,s.start,s.end); term.kind = 1; term.value = value;
        return state::terminate(blocks,c,term);
    }
    if (kind == 4 || kind == 9 || kind == 10) {
        let usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (kind == 9) { return state::guard(operations,blocks,c,6,0,value,0,16,s.start,s.end); }
        return true;
    }
    discard state::fail(c,4,s.start,s.end); return false;
}
fn usize block(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    let h::Block b = h::block_at(input,id);
    if (depth >= 64) { return state::fail(c,2,b.start,b.end); }
    var usize i = 0;
    while (i < b.edge_count && c.current != 0 && c.problem.code == 0) {
        if (!statement(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,b.edge_start + i),depth + 1)) { return 0; } i += 1;
    }
    return expression(input,locals,blocks,operations,edges,payload,c,b.tail,depth + 1);
}

pub fn Result<m::Program,m::Problem> lower(h::Program input) {
    if (!admitted(ref input)) { return .err(m::Problem { .code = 4, .start = 0, .end = h::source_view(ref input).len }); }
    var state::Context c = state::empty(); var bytes::Buffer source = bytes::empty(); var bytes::Buffer payload = bytes::empty();
    var v::Vec<m::Target> targets = v::empty<m::Target>(); var v::Vec<m::Signature> signatures = v::empty<m::Signature>();
    var v::Vec<m::Function> functions = v::empty<m::Function>(); var v::Vec<m::Local> locals = v::empty<m::Local>();
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); var v::Vec<m::Operation> operations = v::empty<m::Operation>(); var v::Vec<usize> edges = v::empty<usize>();
    if (!prepare(ref input,ref mut targets,ref mut signatures,ref mut c)) { return .err(c.problem); }
    if (!bytes::append(ref mut source,h::source_view(ref input)).is_ok) { return .err(m::Problem { .code = 3, .start = 0, .end = 0 }); }
    var usize index = 0;
    while (index < h::function_count(ref input)) {
        let h::Function f = h::function_at(ref input,index); c.function = index + 1;
        let usize local_start = v::len<m::Local>(ref locals); let usize block_start = v::len<m::Block>(ref blocks);
        var usize id = 1;
        while (id <= h::binding_count(ref input)) {
            let h::Binding b = h::binding_at(ref input,id);
            if (b.owner == f.binding) { let usize local = state::local(ref mut locals,ref mut c,h::type_code(b.type),id,h::binding_code(b.kind) == 2,h::binding_code(b.kind) == 4,b.start,b.end); c.locals[id] = local; }
            id += 1;
        }
        let usize parameter_start = v::len<usize>(ref edges); var usize j = 0;
        while (j < f.parameter_count) { let usize parameter = c.locals[h::edge_at(ref input,f.parameter_start + j)]; if (!state::edge(ref mut edges,parameter,ref mut c)) { return .err(c.problem); } j += 1; }
        let usize entry = state::new_block(ref mut blocks,ref mut c,f.start,f.end); c.current = entry;
        discard block(ref input,ref mut locals,ref mut blocks,ref mut operations,ref mut edges,ref mut payload,ref mut c,f.body,0);
        if (c.problem.code != 0) { return .err(c.problem); }
        if (c.current != 0) {
            if (h::type_code(h::binding_at(ref input,f.binding).type) != 7) { return .err(m::Problem { .code = 4, .start = f.start, .end = f.end }); }
            var m::Block term = state::block(c.function,f.start,f.end); term.kind = 1;
            if (!state::terminate(ref mut blocks,ref mut c,term)) { return .err(c.problem); }
        }
        if (!v::push<m::Function>(ref mut functions,m::Function { .target = c.targets[f.binding], .entry, .local_start, .local_count = v::len<m::Local>(ref locals) - local_start, .block_start, .block_count = v::len<m::Block>(ref blocks) - block_start, .parameter_start, .parameter_count = f.parameter_count, .start = f.start, .end = f.end, .public = f.public }).is_ok) { return .err(m::Problem { .code = 3, .start = f.start, .end = f.end }); }
        index += 1;
    }
    return m::assemble(source,payload,targets,signatures,functions,locals,blocks,operations,edges,h::module_index(ref input));
}
