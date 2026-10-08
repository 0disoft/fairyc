module name_resolution;
import syntax as syntax;
import raw_view_intrinsics as raw;
import effect_declarations as effects;
import foreign_declarations as foreign;
import repr_c_declarations as repr;
import generic_expand as generics;
import nominal_types as nominal;
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
    usize scope; usize next_scope; usize owner; usize receiver; bool modules;
}
pub resource Resolved { syntax::Parsed source; Context context; v::Vec<Symbol> symbols; v::Vec<Reference> references; v::Vec<modules::Item> external; v::Vec<u32> parameters; Problem problem; cleanup { } }
fn Problem ok() { return Problem { .code = 0, .start = 0, .end = 0 }; }
fn Problem fail(u32 code, usize start, usize end) { return Problem { .code, .start, .end }; }
fn Context empty() {
    return Context { .globals = [Name { .start = 0, .end = 0, .scope = 0, .symbol = 0 }; 128], .global_count = 0,
        .locals = [Name { .start = 0, .end = 0, .scope = 0, .symbol = 0 }; 128], .local_count = 0, .scope = 0, .next_scope = 1, .owner = 0, .receiver = 0, .modules = false };
}
fn ref u8[] source_name(ref syntax::Parsed p,syntax::Declaration d){return syntax::source_view(p)[d.name_start..d.name_end];}
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
    while (index > 0) { index -= 1;
        let Name name = context.locals[index];
        if (name.symbol == context.receiver && context.receiver != 0) {
            if (bytes::equal(source[start..end],b"this")) { return name.symbol; }
        } else if (same(source,name,start,end)) { return name.symbol; }
    }
    index = 0;
    while (index < context.global_count) {
        if (syntax::module_index(parsed,context.globals[index].start)==syntax::module_index(parsed,start) && same(source, context.globals[index], start, end)) { return context.globals[index].symbol; } index += 1;
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
    let u32 type = nominal::resolve(parsed,id);
    return type != 0 && (result || type != 7);
}
fn Problem expression(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth, ref mut v::Vec<Symbol> symbols, ref mut v::Vec<Reference> references) {
    if (id == 0) { return ok(); }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return fail(4,node.start,node.end); }
    let ref u8[] spelling = syntax::expression_text(parsed,id);
    if (node.kind == 1) {
        return if ((syntax::literal_kind(parsed,id) == 5 || syntax::literal_kind(parsed,id) == 7 || syntax::literal_kind(parsed,id) == 8 || syntax::literal_kind(parsed,id) == 9) || bytes::equal(spelling,b"true") || bytes::equal(spelling,b"false") || bytes::equal(spelling,b"null"))
            { ok() } else { fail(3,node.start,node.end) };
    }
    if (node.kind == 2) {
        if (context.modules && !simple(spelling)) {
            var usize index = 0;
            while (index < v::len<Reference>(references)) { if (v::at<Reference>(references,index).expression == id) { return ok(); } index += 1; }
        }
        if (!simple(spelling) && node.right==0) { return fail(3,node.start,node.end); }
        var usize symbol=0;
        if(node.right!=0){var usize n=0;while(n<v::len<Symbol>(symbols)){if((v::at<Symbol>(symbols,n).kind==1 || v::at<Symbol>(symbols,n).kind==6 || v::at<Symbol>(symbols,n).kind==7 || v::at<Symbol>(symbols,n).kind==9 || v::at<Symbol>(symbols,n).kind==11) && v::at<Symbol>(symbols,n).declaration==node.right){symbol=n+1;}n+=1;}}
        else{var usize n=0;while(n<syntax::declaration_count(parsed)){let d=syntax::declaration_at(parsed,n);if(d.kind==12 && bytes::equal(source_name(parsed,d),spelling)){return fail(4501,node.start,node.end);}n+=1;}symbol=known(parsed,context,node.start,node.end);}
        if (symbol == 0) { return fail(1,node.start,node.end); }
        let Reference value = Reference { .expression = id, .start = node.start, .end = node.end, .symbol };
        if (!v::push<Reference>(references,value).is_ok) { return fail(5,node.start,node.end); }
        return ok();
    }
    if (node.kind == 3 || node.kind == 4 || node.kind == 5) {
        // Safe dereference is checked against exact reference types by typing.
        let Problem left = expression(parsed,context,node.left,depth + 1,symbols,references);
        if (left.code != 0) { return left; }
        return if (node.kind == 5) { expression(parsed,context,node.right,depth + 1,symbols,references) } else { ok() };
    }
    if (node.kind == 6) {
        let u32 callee_kind=syntax::expression_at(parsed,node.left).kind;
        if (callee_kind != 2 && callee_kind != 28) { return fail(3,node.start,node.end); }
        if(callee_kind==2 && raw::call(parsed,id)==0){let Problem callee = expression(parsed,context,node.left,depth + 1,symbols,references);if (callee.code != 0) { return callee; }}
        var usize index = 0;
        while (index < syntax::call_argument_count(parsed,id)) {
            let usize entry_id = syntax::call_argument_entry(parsed,id,index);
            let syntax::Expression argument = syntax::expression_at(parsed,entry_id);
            if (syntax::call_argument_labeled(parsed,entry_id)) { return fail(3,argument.start,argument.end); }
            let Problem value = expression(parsed,context,argument.left,depth + 1,symbols,references);
            if (value.code != 0) { return value; } index += 1;
        }
        return ok();
    }
    if (node.kind == 21 || node.kind == 22 || node.kind == 30) { return expression(parsed,context,node.left,depth + 1,symbols,references); }
    if (node.kind == 26) {
        let Problem owner = expression(parsed,context,node.left,depth + 1,symbols,references); if (owner.code != 0) { return owner; }
        let syntax::Expression bounds = syntax::expression_at(parsed,node.right);
        let Problem start = expression(parsed,context,bounds.left,depth + 1,symbols,references);
        return if (start.code != 0) { start } else { expression(parsed,context,bounds.right,depth + 1,symbols,references) };
    }
    if (node.kind == 9) {
        let Problem left = expression(parsed,context,node.left,depth + 1,symbols,references);
        return if (left.code != 0) { left } else { expression(parsed,context,node.right,depth + 1,symbols,references) };
    }
    if (node.kind == 24 || node.kind == 29) {
        var usize i = 0; let usize count = if (node.kind == 29) { 1 } else { syntax::array_element_count(parsed,id) };
        while (i < count) { let Problem p = expression(parsed,context,if (node.kind == 29) { node.left } else { syntax::array_element(parsed,id,i) },depth + 1,symbols,references); if (p.code != 0) { return p; } i += 1; }
        return ok();
    }
    if(node.kind==23){return expression(parsed,context,node.left,depth+1,symbols,references);}
    if (node.kind == 8) { return expression(parsed,context,node.left,depth + 1,symbols,references); }
    if (node.kind == 10) {
        let Problem target = expression(parsed,context,node.left,depth + 1,symbols,references);
        if (target.code != 0) { return target; }
        var usize index = 0; while (index < syntax::record_field_count(parsed,id)) {
            let syntax::Expression field = syntax::expression_at(parsed,syntax::record_field(parsed,id,index));
            let Problem value = expression(parsed,context,field.left,depth + 1,symbols,references);
            if (value.code != 0) { return value; } index += 1;
        }
        return ok();
    }
    if (node.kind == 28) { return ok(); }
    if (node.kind == 15) {
        let Problem subject=expression(parsed,context,node.left,depth + 1,symbols,references);if(subject.code!=0){return subject;}
        var usize link=node.right;while(link!=0){let arm=syntax::expression_at(parsed,syntax::expression_at(parsed,link).left);let pattern=syntax::expression_at(parsed,arm.left);if((pattern.kind!=16 || pattern.right!=0) && pattern.kind!=32){return fail(3,pattern.start,pattern.end);}
            let Problem checked=scoped_block(parsed,context,arm.right,depth + 1,symbols,references,arm.left);if(checked.code!=0){return checked;}link=syntax::expression_at(parsed,link).right;
        }return ok();
    }
    if (node.kind == 13) {
        let Problem condition = expression(parsed,context,node.left,depth + 1,symbols,references);
        if (condition.code != 0) { return condition; }
        let syntax::Expression branches = syntax::expression_at(parsed,node.right);
        let Problem consequent = block(parsed,context,branches.left,depth + 1,symbols,references);
        if (consequent.code != 0) { return consequent; }
        return expression(parsed,context,branches.right,depth + 1,symbols,references);
    }
    if ((node.kind==12 || node.kind==31)) { return block(parsed,context,node.left,depth + 1,symbols,references); }
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
    if (item.kind == 3 || item.kind == 4 || item.kind == 6 || item.kind == 11 || item.kind == 12 || item.kind == 13 || item.kind==16) {
        if (item.kind == 4) {
            if (syntax::expression_at(parsed,item.target).kind != 2 && syntax::expression_at(parsed,item.target).kind != 8 && syntax::expression_at(parsed,item.target).kind != 9 && syntax::expression_at(parsed,item.target).kind!=4 && syntax::expression_at(parsed,item.target).kind!=3) { return fail(3,item.start,item.end); }
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
    return scoped_block(parsed,context,id,depth,symbols,references,0);
}
fn Problem scoped_block(ref syntax::Parsed parsed, ref mut Context context, usize id, usize depth, ref mut v::Vec<Symbol> symbols, ref mut v::Vec<Reference> references, usize pattern) {
    let syntax::Statement body = syntax::statement_at(parsed,id);
    if (depth >= 64) { return fail(4,body.start,body.end); }
    let usize base = context.local_count; let usize parent = context.scope;
    context.scope = context.next_scope; context.next_scope += 1;
    if(pattern!=0){let pat=syntax::expression_at(parsed,pattern);let usize count=if(pat.kind==32){syntax::pattern_field_count(parsed,pattern)}else if(pat.left!=0){1}else{0};
        var usize i=0;while(i<count){
            let usize name_id=if(pat.kind==32){syntax::expression_at(parsed,syntax::pattern_field(parsed,pattern,i)).left}else{pat.left};let name=syntax::expression_at(parsed,name_id);
            if(!bytes::equal(syntax::expression_text(parsed,name_id),b"_")){
                if(!simple(syntax::expression_text(parsed,name_id))){return fail(3,name.start,name.end);}
                var usize j=base;while(j<context.local_count){if(same(syntax::source_view(parsed),context.locals[j],name.start,name.end)){return fail(2,name.start,name.end);}j+=1;}
                if(context.local_count>=128){return fail(4,name.start,name.end);}
                if(!v::push<Symbol>(symbols,Symbol{.kind=8,.start=name.start,.end=name.end,.scope=context.scope,.owner=context.owner,.type_root=0,.entry=0,.declaration=name_id}).is_ok){return fail(5,name.start,name.end);}
                let usize slot=context.local_count;let usize scope=context.scope;context.locals[slot]=Name{.start=name.start,.end=name.end,.scope,.symbol=v::len<Symbol>(symbols)};context.local_count+=1;
            }i+=1;
        }
    }
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
    if (syntax::problem_count(parsed) != 0) {let p=syntax::problem_at(parsed,0);return if(p.code>=4501 || p.code==2){fail(p.code,p.start,p.end)}else{fail(6,0,0)}; }
    let ref u8[] source = syntax::source_view(parsed);
    var usize index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration item = syntax::declaration_at(parsed,index);
        if(item.attribute_count!=0){let e=effects::read(parsed,index);if(e.code==4201 || item.kind==12 && e.code!=0){return fail(e.code,e.start,e.end);}}
        if (item.kind != 1 && item.kind!=12 && !(context.modules && item.kind == 2)) {
            if ((item.kind != 3 && item.kind != 4 && item.kind != 6 && item.kind != 9 && item.kind != 11) || item.owner_end != 0 || (item.attribute_count != 0 && item.kind!=4 && item.kind!=6 && item.kind!=3 && item.kind!=9) ||
                item.generic_parameters != 0 || (item.kind == 3 && !scalar_type(parsed,item.result_type,true))) { return fail(3,item.start,item.end); }
            if(item.kind==3 || item.kind==6 || item.kind==9){let e=effects::read(parsed,index);if(e.code!=0){return fail(e.code,e.start,e.end);}}
            if(item.kind==4){let representation=repr::read(parsed,index);if(representation.code!=0){return fail(representation.code,representation.start,representation.end);}}
            if(item.kind==9){let f=foreign::read(parsed,index);if(f.code!=0){return fail(f.code,f.start,f.end);}}
            // Resource attributes stay closed except for existing abilities.
            if(item.kind==6 && (item.cleanup_body==0 || item.cleanup_attribute_count!=0)){return fail(3,item.start,item.end);}
            if (!simple(source[item.name_start..item.name_end])) { return fail(3,item.name_start,item.name_end); }
            var usize previous = 0;
            while (previous < context.global_count) {
                if (syntax::module_index(parsed,context.globals[previous].start)==syntax::module_index(parsed,item.start) && same(source,context.globals[previous],item.name_start,item.name_end)) {let old=syntax::declaration_at(parsed,v::at<Symbol>(symbols,context.globals[previous].symbol-1).declaration-1);if(item.generic_start==0 || item.generic_start!=old.generic_start || item.generic_end==0 || old.generic_end==0){return fail(2,item.name_start,item.name_end);} }
                previous += 1;
            }
            if (context.global_count >= 128) { return fail(4,item.name_start,item.name_end); }
            let usize slot = context.global_count;
            let Symbol value = Symbol { .kind = if (item.kind == 4) { 6 } else if (item.kind == 6) { 9 } else if (item.kind == 11) { 7 } else if(item.kind==9){11}else { 1 }, .start = item.name_start, .end = item.name_end, .scope = 0,
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
            context.receiver = 0;
            context.local_count = 0; context.scope = context.next_scope; context.next_scope += 1;
            context.owner=0;var usize n=0;while(n<v::len<Symbol>(symbols)){if(v::at<Symbol>(symbols,n).kind==1 && v::at<Symbol>(symbols,n).declaration==index+1){context.owner=n+1;}n+=1;}
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
        } else if (item.kind == 6 && (syntax::block_statement_count(parsed,item.cleanup_body)!=0 || syntax::statement_at(parsed,item.cleanup_body).expression!=0)) {
            context.local_count=0;context.scope=context.next_scope;context.next_scope+=1;
            context.owner=known(parsed,context,item.name_start,item.name_end);
            let usize start=item.cleanup_start;let usize end=start+7;
            // Kind 10 is a compiler-owned dropping receiver, never a second
            // ordinary by-value parameter or a source-visible callable.
            if(!v::push<Symbol>(symbols,Symbol{.kind=10,.start,.end,.scope=context.scope,.owner=context.owner,.type_root=0,.entry=0,.declaration=index+1}).is_ok){return fail(5,start,end);}
            context.receiver=v::len<Symbol>(symbols);
            let usize scope=context.scope;let usize receiver=context.receiver;
            context.locals[0]=Name{.start,.end,.scope,.symbol=receiver};context.local_count=1;
            let Problem result=block(parsed,context,item.cleanup_body,0,symbols,references);
            if(result.code!=0){return result;}
            context.receiver=0;
        }
        index += 1;
    }
    return ok();
}

/// Keep source, syntax and declaration-use links in one owner. Failure retains
/// only diagnostic authority; table queries cannot publish partial resolution.
pub fn Resolved resolve_program(syntax::Parsed parsed){var Context context=empty();context.modules=true;var v::Vec<Symbol> symbols=v::empty<Symbol>();var v::Vec<Reference> references=v::empty<Reference>();let Problem problem=collect(ref parsed,ref mut context,ref mut symbols,ref mut references);return Resolved{.source=parsed,.context,.symbols,.references,.external=v::empty<modules::Item>(),.parameters=v::empty<u32>(),.problem};}
pub fn Resolved resolve(syntax::Parsed parsed) {
    let syntax::Parsed prepared=generics::expand(parsed);
    var Context context = empty();
    var v::Vec<Symbol> symbols = v::empty<Symbol>(); var v::Vec<Reference> references = v::empty<Reference>();
    let Problem problem = collect(ref prepared,ref mut context,ref mut symbols,ref mut references);
    return Resolved { .source = prepared, .context, .symbols, .references, .external = v::empty<modules::Item>(), .parameters = v::empty<u32>(), .problem };
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
