module name_resolution;
import syntax as syntax;
import std::bytes as bytes;

// Bounded name-only candidate. No type, flow, call-arity or execution facts.
pub struct Problem { u32 code; usize start; usize end; }
struct Name { usize start; usize end; usize scope; }
struct Context { Name[128] globals; usize global_count; Name[128] locals; usize local_count; usize scope; }
fn Problem ok() { return Problem { .code = 0, .start = 0, .end = 0 }; }
fn Problem fail(u32 code, usize start, usize end) { return Problem { .code, .start, .end }; }
fn Context empty() {
    return Context { .globals = [Name { .start = 0, .end = 0, .scope = 0 }; 128], .global_count = 0,
        .locals = [Name { .start = 0, .end = 0, .scope = 0 }; 128], .local_count = 0, .scope = 0 };
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
fn bool known(ref syntax::Parsed parsed, ref Context context, usize start, usize end) {
    let ref u8[] source = syntax::source_view(parsed);
    var usize index = context.local_count;
    while (index > 0) { index -= 1; if (same(source, context.locals[index], start, end)) { return true; } }
    index = 0;
    while (index < context.global_count) {
        if (same(source, context.globals[index], start, end)) { return true; } index += 1;
    }
    return false;
}
fn Problem local(ref syntax::Parsed parsed, ref mut Context context, usize start, usize end) {
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
    context.locals[slot] = Name { .start, .end, .scope }; context.local_count += 1;
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
fn Problem expression(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth) {
    if (id == 0) { return ok(); }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return fail(4,node.start,node.end); }
    let ref u8[] spelling = syntax::expression_text(parsed,id);
    if (node.kind == 1) {
        return if (syntax::literal_kind(parsed,id) == 5 || bytes::equal(spelling,b"true") || bytes::equal(spelling,b"false"))
            { ok() } else { fail(3,node.start,node.end) };
    }
    if (node.kind == 2) {
        if (!simple(spelling)) { return fail(3,node.start,node.end); }
        return if (known(parsed,context,node.start,node.end)) { ok() } else { fail(1,node.start,node.end) };
    }
    if (node.kind == 3 || node.kind == 4 || node.kind == 5) {
        if (node.kind == 4 && bytes::equal(spelling,b"*")) { return fail(3,node.start,node.end); }
        let Problem left = expression(parsed,context,node.left,depth + 1);
        if (left.code != 0) { return left; }
        return if (node.kind == 5) { expression(parsed,context,node.right,depth + 1) } else { ok() };
    }
    if (node.kind == 6) {
        if (syntax::expression_at(parsed,node.left).kind != 2) { return fail(3,node.start,node.end); }
        let Problem callee = expression(parsed,context,node.left,depth + 1);
        if (callee.code != 0) { return callee; }
        var usize index = 0;
        while (index < syntax::call_argument_count(parsed,id)) {
            let usize entry_id = syntax::call_argument_entry(parsed,id,index);
            let syntax::Expression argument = syntax::expression_at(parsed,entry_id);
            if (syntax::source_view(parsed)[argument.start] == 46) { return fail(3,argument.start,argument.end); }
            let Problem value = expression(parsed,context,argument.left,depth + 1);
            if (value.code != 0) { return value; } index += 1;
        }
        return ok();
    }
    if (node.kind == 12) { return block(parsed,context,node.left,depth + 1); }
    return fail(3,node.start,node.end);
}
fn Problem block(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth) {
    let syntax::Statement body = syntax::statement_at(parsed,id);
    if (depth >= 64) { return fail(4,body.start,body.end); }
    let usize base = context.local_count; context.scope += 1;
    var usize index = 0;
    while (index < syntax::block_statement_count(parsed,id)) {
        let syntax::Statement item = syntax::statement_at(parsed,syntax::block_statement(parsed,id,index));
        if (item.kind == 2) {
            let syntax::Entry entry = syntax::entry_at(parsed,item.entry - 1);
            if (!scalar_type(parsed,entry.type_root,false)) { return fail(3,entry.type_start,entry.type_end); }
            let Problem value = expression(parsed,context,item.expression,depth + 1);
            if (value.code != 0) { return value; }
            let Problem added = local(parsed,context,entry.name_start,entry.name_end);
            if (added.code != 0) { return added; }
        } else if (item.kind == 3 || item.kind == 4 || item.kind == 6) {
            if (item.kind == 4) {
                if (syntax::expression_at(parsed,item.target).kind != 2) { return fail(3,item.start,item.end); }
                let Problem target = expression(parsed,context,item.target,depth + 1);
                if (target.code != 0) { return target; }
            }
            let Problem value = expression(parsed,context,item.expression,depth + 1);
            if (value.code != 0) { return value; }
        } else { return fail(3,item.start,item.end); }
        index += 1;
    }
    let Problem tail = expression(parsed,context,body.expression,depth + 1);
    context.local_count = base; context.scope -= 1;
    return tail;
}
pub fn Problem check(ref syntax::Parsed parsed) {
    var Context context = empty();
    let ref u8[] source = syntax::source_view(parsed);
    var usize index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration item = syntax::declaration_at(parsed,index);
        if (item.kind != 1) {
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
            context.globals[slot] = Name { .start = item.name_start, .end = item.name_end, .scope = 0 }; context.global_count += 1;
        }
        index += 1;
    }
    index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration item = syntax::declaration_at(parsed,index);
        if (item.kind == 3) {
            context.local_count = 0; context.scope = 0;
            var usize entry_index = item.entry_start;
            while (entry_index < item.entry_start + item.entry_count) {
                let syntax::Entry entry = syntax::entry_at(parsed,entry_index);
                if (entry.kind == 1) {
                    if (!scalar_type(parsed,entry.type_root,false)) { return fail(3,entry.type_start,entry.type_end); }
                    let Problem added = local(parsed,ref mut context,entry.name_start,entry.name_end);
                    if (added.code != 0) { return added; }
                }
                entry_index += 1;
            }
            let Problem result = block(parsed,ref mut context,item.body,0);
            if (result.code != 0) { return result; }
        }
        index += 1;
    }
    return ok();
}
