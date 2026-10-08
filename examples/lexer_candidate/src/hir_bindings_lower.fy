module hir_bindings_lower;
import candidate_limits as limits;
import nominal_types as nominal;
import hir_model as h;
import foreign_declarations as foreign;
import effect_declarations as effects;
import hir_state as state;
import hir_byte_loop as byte_loop;
import type_check as typing;
import name_resolution as names;
import syntax as syntax;
import std::vec as v;

pub fn bool prepare(ref typing::Typed input, usize module_index, ref mut v::Vec<h::Binding> bindings, ref mut v::Vec<h::Signature> signatures, ref mut v::Vec<state::Link> links, ref mut state::Context c) {
    let ref names::Resolved resolved = typing::resolved(input);
    let ref syntax::Parsed parsed = names::parsed(resolved);
    if (syntax::source_view(parsed).len > limits::program_bytes() || syntax::declaration_count(parsed) > 512 || names::symbol_count(resolved) > 1152 || syntax::expression_count(parsed) > limits::program_expressions() || syntax::entry_count(parsed) > 1024 || syntax::statement_count(parsed) > 8192 || names::reference_count(resolved) > limits::program_expressions()) { discard state::fail(c,2,0,syntax::source_view(parsed).len); return false; }
    var usize[513] dropping=[0;513];var usize declaration=0;var usize synthetic=names::symbol_count(resolved);
    while(declaration<syntax::declaration_count(parsed)){let d=syntax::declaration_at(parsed,declaration);
        if(d.kind==6 && (syntax::block_statement_count(parsed,d.cleanup_body)!=0 || syntax::statement_at(parsed,d.cleanup_body).expression!=0)){synthetic+=1;dropping[declaration+1]=synthetic;}
        declaration+=1;}
    if(synthetic>1152){discard state::fail(c,2,0,syntax::source_view(parsed).len);return false;}
    let usize length = if (syntax::expression_count(parsed) > 1024) { syntax::expression_count(parsed) } else { 1024 };
    var usize i = 0;
    while (i < length) { if (!v::push<state::Link>(links,state::Link { .use = 0, .entry = 0, .function = 0,.iteration_view=0,.iteration_index=0 }).is_ok) { discard state::fail(c,3,0,0); return false; } i += 1; }
    i = 1;
    while (i <= names::symbol_count(resolved)) {
        let names::Symbol s = names::symbol_at(resolved,i);
        let h::Type type = state::type(typing::symbol_type(input,i));
        if (s.kind < 1 || s.kind > 12 || h::type_code(type) == h::type_code(.Invalid) || h::type_code(type) == h::type_code(.Callable) || s.start > s.end || s.end > syntax::source_view(parsed).len || s.owner > names::symbol_count(resolved)) { discard state::fail(c,4,s.start,s.end); return false; }
        var usize target_module = module_index; var usize declaration = s.declaration;
        if (s.kind == 5) { let signature = names::external_signature(resolved,i); target_module = signature.module_index; declaration = signature.declaration; }
        let usize owner=if(s.owner!=0 && names::symbol_at(resolved,s.owner).kind==9){dropping[names::symbol_at(resolved,s.owner).declaration]}else{s.owner};
        let ffi=if(s.kind==11){foreign::read(parsed,s.declaration-1)}else{foreign::Info{.code=0,.start=0,.end=0,.link_start=0,.link_end=0}};
        let h::Binding binding = h::Binding { .link_start=ffi.link_start,.link_end=ffi.link_end, .unsafe_call=if(s.kind==5){names::external_signature(resolved,i).unsafe_call}else{(s.kind==1 || s.kind==11) && syntax::declaration_at(parsed,s.declaration-1).unsafe_end!=0}, .must_use = if(s.kind==5){names::external_signature(resolved,i).must_use}else{(s.kind==1 || s.kind==11) && effects::read(parsed,s.declaration-1).must_use}, .effects = if(s.kind==1 || s.kind==9 || s.kind==11){effects::read(parsed,s.declaration-1).bits}else if(s.kind==5){names::external_signature(resolved,i).effects}else{0}, .kind = if(s.kind==8){.Let}else{state::binding_kind(s.kind)}, .type, .start = s.start, .end = s.end, .owner, .scope = s.scope, .module_index = target_module, .declaration = if(s.kind==8 || s.kind==10){0}else{declaration}, .signature_start = 0, .signature_count = 0 };
        if (!v::push<h::Binding>(bindings,binding).is_ok) { discard state::fail(c,3,s.start,s.end); return false; }
        if(s.kind==8){if(s.declaration==0 || s.declaration>length){discard state::fail(c,4,s.start,s.end);return false;}let ref mut state::Link link=v::at_mut<state::Link>(links,s.declaration-1);link.use=i;}
        if (s.entry != 0) {
            if (s.entry > length || state::entry(links,s.entry) != 0) { discard state::fail(c,4,s.start,s.end); return false; }
            let ref mut state::Link link = v::at_mut<state::Link>(links,s.entry - 1); link.entry = i;
        }
        if(s.kind==12){if(s.declaration==0 || s.declaration>length || state::use(links,s.declaration)!=0){discard state::fail(c,4,s.start,s.end);return false;}let ref mut state::Link link=v::at_mut<state::Link>(links,s.declaration-1);link.use=i;}
        if (s.kind == 1) {
            if (s.declaration == 0 || s.declaration > syntax::declaration_count(parsed) || state::function(links,s.declaration) != 0 || s.owner != 0 || s.scope != 0) { discard state::fail(c,4,s.start,s.end); return false; }
            let ref mut state::Link link = v::at_mut<state::Link>(links,s.declaration - 1); link.function = i;
        }
        if ((s.kind == 2 || s.kind == 3 || s.kind == 4) && (s.owner == 0 || s.scope == 0 || s.entry == 0 || h::type_code(type) == h::type_code(.Void))) { discard state::fail(c,4,s.start,s.end); return false; }
        i += 1;
    }
    declaration=0;while(declaration<syntax::declaration_count(parsed)){let d=syntax::declaration_at(parsed,declaration);let usize target=dropping[declaration+1];
        if(target!=0){
            if(target!=v::len<h::Binding>(bindings)+1 || !v::push<h::Binding>(bindings,h::Binding{ .link_start=0,.link_end=0, .unsafe_call=false,.must_use=false,.effects=effects::read(parsed,declaration).bits,.kind=.DropFunction,.type=.Void,.start=d.cleanup_start,.end=d.cleanup_end,.owner=0,.scope=0,.module_index,.declaration=declaration+1,.signature_start=0,.signature_count=0}).is_ok){discard state::fail(c,3,d.cleanup_start,d.cleanup_end);return false;}
            let ref mut state::Link link=v::at_mut<state::Link>(links,declaration);link.function=target;
        }declaration+=1;}
    i = 1;
    while (i <= names::symbol_count(resolved)) {
        let names::Symbol s = names::symbol_at(resolved,i); let h::Binding b = *v::at<h::Binding>(bindings,i - 1);
        if (b.owner != 0 && !h::function_kind(v::at<h::Binding>(bindings,b.owner - 1).kind)) { discard state::fail(c,4,b.start,b.end); return false; }
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
        } else if(s.kind==11){let d=syntax::declaration_at(parsed,s.declaration-1);var usize entry=d.entry_start;while(entry<d.entry_start+d.entry_count){let e=syntax::entry_at(parsed,entry);if(e.kind==1){if(!v::push<h::Signature>(signatures,h::Signature{.type=state::type(nominal::resolve(parsed,e.type_root))}).is_ok){discard state::fail(c,3,e.start,e.end);return false;}}entry+=1;}
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
    i=names::symbol_count(resolved)+1;while(i<=v::len<h::Binding>(bindings)){
        let h::Binding b=*v::at<h::Binding>(bindings,i-1);let usize start=v::len<h::Signature>(signatures);var usize receiver=0;var usize n=1;
        while(n<=names::symbol_count(resolved)){let h::Binding candidate=*v::at<h::Binding>(bindings,n-1);if(h::binding_code(candidate.kind)==10 && candidate.owner==i){receiver=n;}n+=1;}
        if(receiver==0 || v::len<h::Signature>(signatures)>=32768){discard state::fail(c,4,b.start,b.end);return false;}
        let h::Type type=v::at<h::Binding>(bindings,receiver-1).type;
        if(!v::push<h::Signature>(signatures,h::Signature{.type}).is_ok){discard state::fail(c,3,b.start,b.end);return false;}
        let ref mut h::Binding stored=v::at_mut<h::Binding>(bindings,i-1);stored.signature_start=start;stored.signature_count=1;i+=1;
    }
    i = 0;
    while (i < names::reference_count(resolved)) {
        let names::Reference use = names::reference_at(resolved,i);
        if (use.expression == 0 || use.expression > syntax::expression_count(parsed) || use.symbol == 0 || use.symbol > v::len<h::Binding>(bindings) || state::use(links,use.expression) != 0) { discard state::fail(c,4,use.start,use.end); return false; }
        let ref mut state::Link link = v::at_mut<state::Link>(links,use.expression - 1); link.use = use.symbol; i += 1;
    }
    return byte_loop::prepare(input,bindings,links,c);
}
