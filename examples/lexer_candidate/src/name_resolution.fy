module name_resolution;
import syntax as syntax;
import module_registry as modules;
import std::bytes as bytes;
import std::vec as v;

// Bounded name-only candidate. No type, flow, call-arity or execution facts.
pub struct Problem { u32 code; usize start; usize end; }
pub struct Symbol { u32 kind; usize start; usize end; usize scope; usize owner; usize type_root; usize entry; usize declaration; }
pub struct Reference { usize expression; usize start; usize end; usize symbol; }
struct Name { usize start; usize end; usize scope; usize symbol; }
struct Context {
    Name[128] globals; usize global_count; Name[128] locals; usize local_count;
    usize scope; usize next_scope; usize owner; bool modules;
}
pub resource Resolved { syntax::Parsed source; Context context; v::Vec<Symbol> symbols; v::Vec<Reference> references; v::Vec<modules::Item> external; v::Vec<u32> parameters; Problem problem; cleanup { } }
fn Problem ok() { return Problem { .code = 0, .start = 0, .end = 0 }; }
fn Problem fail(u32 code, usize start, usize end) { return Problem { .code, .start, .end }; }
fn Context empty() {
    return Context { .globals = [Name { .start = 0, .end = 0, .scope = 0, .symbol = 0 }; 128], .global_count = 0,
        .locals = [Name { .start = 0, .end = 0, .scope = 0, .symbol = 0 }; 128], .local_count = 0, .scope = 0, .next_scope = 1, .owner = 0, .modules = false };
}
fn bool simple(ref u8[] name) {
    if (name.len == 0) { return false; }
    for (let u8 byte in name) {
        if (!((byte >= 65 && byte <= 90) || (byte >= 97 && byte <= 122) ||
            (byte >= 48 && byte <= 57) || byte == 95)) { return false; }
    }
    return !bytes::equal(name, b"source_location");
}
fn bool same(ref u8[] source, Name name, usize start, usize end) {
    return bytes::equal(source[name.start..name.end], source[start..end]);
}
fn usize known(ref syntax::Parsed parsed, ref Context context, usize start, usize end) {
    let ref u8[] source = syntax::source_view(parsed);
    var usize index = context.local_count;
    while (index > 0) { index -= 1; if (same(source, context.locals[index], start, end)) { return context.locals[index].symbol; } }
    index = 0;
    while (index < context.global_count) {
        if (same(source, context.globals[index], start, end)) { return context.globals[index].symbol; } index += 1;
    }
    return 0;
}
fn Problem local(ref syntax::Parsed parsed, ref mut Context context, syntax::Entry entry, usize entry_id, ref mut v::Vec<Symbol> symbols) {
    let usize start = entry.name_start; let usize end = entry.name_end;
    let ref u8[] source = syntax::source_view(parsed);
    if (!simple(source[start..end])) { return fail(3, start, end); }
    var usize index = context.local_count;
    while (index > 0) {
        index -= 1;
        if (context.locals[index].scope == context.scope && same(source, context.locals[index], start, end)) {
            return fail(2, start, end);
        }
    }
    if (context.local_count >= 128) { return fail(4, start, end); }
    let usize slot = context.local_count;
    let usize scope = context.scope;
    let Symbol value = Symbol { .kind = if (entry.kind == 1) { 2 } else if (entry.kind == 2) { 3 } else { 4 },
        .start, .end, .scope, .owner = context.owner, .type_root = entry.type_root, .entry = entry_id, .declaration = 0 };
    if (!v::push<Symbol>(symbols,value).is_ok) { return fail(5,start,end); }
    let usize symbol = v::len<Symbol>(symbols);
    context.locals[slot] = Name { .start, .end, .scope, .symbol }; context.local_count += 1;
    return ok();
}
fn bool scalar_type(ref syntax::Parsed parsed, usize id, bool result) {
    if (id == 0) { return true; } // Inferred local: resolution does not infer its type.
    if (syntax::type_at(parsed, id).kind != 1) { return false; }
    let ref u8[] name = syntax::type_text(parsed, id);
    return bytes::equal(name,b"i32") || bytes::equal(name,b"i64") || bytes::equal(name,b"u8") ||
        bytes::equal(name,b"u32") || bytes::equal(name,b"usize") || bytes::equal(name,b"bool") ||
        (result && bytes::equal(name,b"void"));
}
fn Problem expression(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth, ref mut v::Vec<Symbol> symbols, ref mut v::Vec<Reference> references) {
    if (id == 0) { return ok(); }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return fail(4,node.start,node.end); }
    let ref u8[] spelling = syntax::expression_text(parsed,id);
    if (node.kind == 1) {
        return if (syntax::literal_kind(parsed,id) == 5 || bytes::equal(spelling,b"true") || bytes::equal(spelling,b"false"))
            { ok() } else { fail(3,node.start,node.end) };
    }
    if (node.kind == 2) {
        if (context.modules && !simple(spelling)) {
            var usize index = 0;
            while (index < v::len<Reference>(references)) { if (v::at<Reference>(references,index).expression == id) { return ok(); } index += 1; }
        }
        if (!simple(spelling)) { return fail(3,node.start,node.end); }
        let usize symbol = known(parsed,context,node.start,node.end);
        if (symbol == 0) { return fail(1,node.start,node.end); }
        let Reference value = Reference { .expression = id, .start = node.start, .end = node.end, .symbol };
        if (!v::push<Reference>(references,value).is_ok) { return fail(5,node.start,node.end); }
        return ok();
    }
    if (node.kind == 3 || node.kind == 4 || node.kind == 5) {
        if (node.kind == 4 && bytes::equal(spelling,b"*")) { return fail(3,node.start,node.end); }
        let Problem left = expression(parsed,context,node.left,depth + 1,symbols,references);
        if (left.code != 0) { return left; }
        return if (node.kind == 5) { expression(parsed,context,node.right,depth + 1,symbols,references) } else { ok() };
    }
    if (node.kind == 6) {
        if (syntax::expression_at(parsed,node.left).kind != 2) { return fail(3,node.start,node.end); }
        let Problem callee = expression(parsed,context,node.left,depth + 1,symbols,references);
        if (callee.code != 0) { return callee; }
        var usize index = 0;
        while (index < syntax::call_argument_count(parsed,id)) {
            let usize entry_id = syntax::call_argument_entry(parsed,id,index);
            let syntax::Expression argument = syntax::expression_at(parsed,entry_id);
            if (syntax::source_view(parsed)[argument.start] == 46) { return fail(3,argument.start,argument.end); }
            let Problem value = expression(parsed,context,argument.left,depth + 1,symbols,references);
            if (value.code != 0) { return value; } index += 1;
        }
        return ok();
    }
    if (node.kind == 13) {
        let Problem condition = expression(parsed,context,node.left,depth + 1,symbols,references);
        if (condition.code != 0) { return condition; }
        let syntax::Expression branches = syntax::expression_at(parsed,node.right);
        let Problem consequent = block(parsed,context,branches.left,depth + 1,symbols,references);
        if (consequent.code != 0) { return consequent; }
        return expression(parsed,context,branches.right,depth + 1,symbols,references);
    }
    if (node.kind == 12) { return block(parsed,context,node.left,depth + 1,symbols,references); }
    return fail(3,node.start,node.end);
}
fn Problem statement(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth, ref mut v::Vec<Symbol> symbols, ref mut v::Vec<Reference> references) {
    if (id == 0) { return ok(); }
    let syntax::Statement item = syntax::statement_at(parsed,id);
    if (depth >= 64) { return fail(4,item.start,item.end); }
    if (item.kind == 2) {
        let syntax::Entry entry = syntax::entry_at(parsed,item.entry - 1);
        if (!scalar_type(parsed,entry.type_root,false)) { return fail(3,entry.type_start,entry.type_end); }
        let Problem value = expression(parsed,context,item.expression,depth,symbols,references);
        if (value.code != 0) { return value; }
        return local(parsed,context,entry,item.entry,symbols);
    }
    if (item.kind == 3 || item.kind == 4 || item.kind == 6 || item.kind == 11 || item.kind == 12 || item.kind == 13) {
        if (item.kind == 4) {
            if (syntax::expression_at(parsed,item.target).kind != 2) { return fail(3,item.start,item.end); }
            let Problem target = expression(parsed,context,item.target,depth,symbols,references);
            if (target.code != 0) { return target; }
        }
        return expression(parsed,context,item.expression,depth,symbols,references);
    }
    if (item.kind == 9 || item.kind == 10) { return ok(); } // Type phase owns loop validity.
    if (item.kind == 8) {
        let Problem condition = expression(parsed,context,item.expression,depth,symbols,references);
        if (condition.code != 0) { return condition; }
        return block(parsed,context,item.first,depth,symbols,references);
    }
    if (item.kind == 14) {
        let usize base = context.local_count; let usize parent = context.scope;
        context.scope = context.next_scope; context.next_scope += 1;
        let syntax::Statement header = syntax::statement_at(parsed,item.second);
        let Problem initial = statement(parsed,context,header.first,depth + 1,symbols,references);
        if (initial.code != 0) { return initial; }
        let Problem condition = expression(parsed,context,item.expression,depth,symbols,references);
        if (condition.code != 0) { return condition; }
        // Resolve post before the body's declarations, matching the reference.
        let Problem post = statement(parsed,context,header.second,depth + 1,symbols,references);
        if (post.code != 0) { return post; }
        let Problem body = block(parsed,context,item.first,depth,symbols,references);
        context.local_count = base; context.scope = parent;
        return body;
    }
    return fail(3,item.start,item.end);
}
fn Problem block(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth, ref mut v::Vec<Symbol> symbols, ref mut v::Vec<Reference> references) {
    let syntax::Statement body = syntax::statement_at(parsed,id);
    if (depth >= 64) { return fail(4,body.start,body.end); }
    let usize base = context.local_count; let usize parent = context.scope;
    context.scope = context.next_scope; context.next_scope += 1;
    var usize index = 0;
    while (index < syntax::block_statement_count(parsed,id)) {
        let Problem checked = statement(parsed,context,syntax::block_statement(parsed,id,index),depth + 1,symbols,references);
        if (checked.code != 0) { return checked; }
        index += 1;
    }
    let Problem tail = expression(parsed,context,body.expression,depth + 1,symbols,references);
    context.local_count = base; context.scope = parent;
    return tail;
}
fn Problem collect(ref syntax::Parsed parsed, ref mut Context context, ref mut v::Vec<Symbol> symbols, ref mut v::Vec<Reference> references) {
    if (syntax::problem_count(parsed) != 0) { return fail(6,0,0); }
    let ref u8[] source = syntax::source_view(parsed);
    var usize index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration item = syntax::declaration_at(parsed,index);
        if (item.kind != 1 && !(context.modules && item.kind == 2)) {
            if (item.kind != 3 || item.owner_end != 0 || item.unsafe_end != 0 || item.attribute_count != 0 ||
                item.generic_parameters != 0 || !scalar_type(parsed,item.result_type,true)) { return fail(3,item.start,item.end); }
            if (!simple(source[item.name_start..item.name_end])) { return fail(3,item.name_start,item.name_end); }
            var usize previous = 0;
            while (previous < context.global_count) {
                if (same(source,context.globals[previous],item.name_start,item.name_end)) { return fail(2,item.name_start,item.name_end); }
                previous += 1;
            }
            if (context.global_count >= 128) { return fail(4,item.name_start,item.name_end); }
            let usize slot = context.global_count;
            let Symbol value = Symbol { .kind = 1, .start = item.name_start, .end = item.name_end, .scope = 0,
                .owner = 0, .type_root = item.result_type, .entry = 0, .declaration = index + 1 };
            if (!v::push<Symbol>(symbols,value).is_ok) { return fail(5,item.name_start,item.name_end); }
            let usize symbol = v::len<Symbol>(symbols);
            context.globals[slot] = Name { .start = item.name_start, .end = item.name_end, .scope = 0, .symbol }; context.global_count += 1;
        }
        index += 1;
    }
    index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration item = syntax::declaration_at(parsed,index);
        if (item.kind == 3) {
            context.local_count = 0; context.scope = context.next_scope; context.next_scope += 1;
            context.owner = known(parsed,context,item.name_start,item.name_end);
            var usize entry_index = item.entry_start;
            while (entry_index < item.entry_start + item.entry_count) {
                let syntax::Entry entry = syntax::entry_at(parsed,entry_index);
                if (entry.kind == 1) {
                    if (!scalar_type(parsed,entry.type_root,false)) { return fail(3,entry.type_start,entry.type_end); }
                    let Problem added = local(parsed,context,entry,entry_index + 1,symbols);
                    if (added.code != 0) { return added; }
                }
                entry_index += 1;
            }
            let Problem result = block(parsed,context,item.body,0,symbols,references);
            if (result.code != 0) { return result; }
        }
        index += 1;
    }
    return ok();
}

/// Keep source, syntax and declaration-use links in one owner. Failure retains
/// only diagnostic authority; table queries cannot publish partial resolution.
pub fn Resolved resolve(syntax::Parsed parsed) {
    var Context context = empty();
    var v::Vec<Symbol> symbols = v::empty<Symbol>(); var v::Vec<Reference> references = v::empty<Reference>();
    let Problem problem = collect(ref parsed,ref mut context,ref mut symbols,ref mut references);
    return Resolved { .source = parsed, .context, .symbols, .references, .external = v::empty<modules::Item>(), .parameters = v::empty<u32>(), .problem };
}
// Registry lookup binds actual qualified expression IDs to copied signatures.
// The result owns all type facts; it borrows neither the registry nor other files.
pub fn Resolved resolve_modules(syntax::Parsed parsed, ref modules::Registry registry, usize module_index) {
    var Context context = empty(); context.modules = true;
    var v::Vec<Symbol> symbols = v::empty<Symbol>(); var v::Vec<Reference> references = v::empty<Reference>();
    var v::Vec<modules::Item> external = v::empty<modules::Item>(); var v::Vec<u32> parameters = v::empty<u32>();
    var Problem problem = ok(); var usize id = 1;
    while (id <= syntax::expression_count(ref parsed)) {
        let syntax::Expression node = syntax::expression_at(ref parsed,id);
        if (node.kind == 2 && !simple(syntax::expression_text(ref parsed,id))) {
            let modules::Lookup lookup = modules::lookup(registry,module_index,syntax::expression_text(ref parsed,id));
            if (lookup.code != 0) {
                problem = fail(if (lookup.code == 4) { 7 } else if (lookup.code == 5) { 8 } else if (lookup.code == 6) { 9 } else if (lookup.code == 11) { 5 } else { 10 },node.start,node.end); break;
            }
            if (v::len<Symbol>(ref symbols) >= 1024) { problem = fail(4,node.start,node.end); break; }
            var modules::Item signature = modules::item_at(registry,lookup.item - 1);
            let usize parameter_start = v::len<u32>(ref parameters); var usize argument = 0;
            while (argument < signature.parameter_count) {
                if (!v::push<u32>(ref mut parameters,modules::parameter_at(registry,signature.parameter_start + argument)).is_ok) { problem = fail(5,node.start,node.end); break; }
                argument += 1;
            }
            if (problem.code != 0) { break; }
            signature.parameter_start = parameter_start;
            if (!v::push<modules::Item>(ref mut external,signature).is_ok) { problem = fail(5,node.start,node.end); break; }
            let Symbol symbol = Symbol { .kind = 5, .start = node.start, .end = node.end, .scope = 0, .owner = 0, .type_root = 0, .entry = 0, .declaration = v::len<modules::Item>(ref external) };
            if (!v::push<Symbol>(ref mut symbols,symbol).is_ok || !v::push<Reference>(ref mut references,Reference { .expression = id, .start = node.start, .end = node.end, .symbol = v::len<Symbol>(ref symbols) }).is_ok) { problem = fail(5,node.start,node.end); break; }
        }
        id += 1;
    }
    if (problem.code == 0) { problem = collect(ref parsed,ref mut context,ref mut symbols,ref mut references); }
    return Resolved { .source = parsed, .context, .symbols, .references, .external, .parameters, .problem };
}
pub fn modules::Item external_signature(ref Resolved resolved, usize symbol_id) {
    let Symbol symbol = symbol_at(resolved,symbol_id); assert(symbol.kind == 5);
    return *v::at<modules::Item>(ref resolved.external,symbol.declaration - 1);
}
pub fn u32 external_parameter(ref Resolved resolved, usize index) { return *v::at<u32>(ref resolved.parameters,index); }
pub fn Problem check(ref syntax::Parsed parsed) {
    var Context context = empty(); var v::Vec<Symbol> symbols = v::empty<Symbol>(); var v::Vec<Reference> references = v::empty<Reference>();
    return collect(parsed,ref mut context,ref mut symbols,ref mut references);
}
pub fn Problem problem(ref Resolved resolved) { return resolved.problem; }
pub fn ref syntax::Parsed parsed(ref Resolved resolved) { return ref resolved.source; }
pub fn ref u8[] source_view(ref Resolved resolved) { return syntax::source_view(ref resolved.source); }
pub fn usize symbol_count(ref Resolved resolved) {
    assert(resolved.problem.code == 0); return v::len<Symbol>(ref resolved.symbols);
}
pub fn Symbol symbol_at(ref Resolved resolved, usize id) {
    assert(resolved.problem.code == 0 && id > 0 && id <= v::len<Symbol>(ref resolved.symbols));
    return *v::at<Symbol>(ref resolved.symbols,id - 1);
}
pub fn usize reference_count(ref Resolved resolved) {
    assert(resolved.problem.code == 0); return v::len<Reference>(ref resolved.references);
}
pub fn Reference reference_at(ref Resolved resolved, usize index) {
    assert(resolved.problem.code == 0); return *v::at<Reference>(ref resolved.references,index);
}
