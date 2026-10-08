module hir_validate_symbols;
import hir_model as h;
import sdk_scopes as sdk;
import must_use_flow as must_use;
import ffi_symbols as ffi;
import array_types as arrays;
import resource_identity as resources;
import enum_types as enums;
import tagged_types as tagged;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

pub fn h::Problem check(ref bytes::Buffer source, ref v::Vec<h::Function> functions, ref v::Vec<h::Binding> bindings, ref v::Vec<h::Block> blocks, ref v::Vec<usize> edges, ref v::Vec<h::Signature> signatures, ref shapes::Table nominal, usize module_index){
    let usize ns=bytes::len(source);
    let usize nf=v::len<h::Function>(functions);
    let usize nb=v::len<h::Binding>(bindings);
    let usize nk=v::len<h::Block>(blocks);
    let usize na=v::len<usize>(edges);
    let usize ntypes=v::len<h::Signature>(signatures);
    if (!shapes::valid(nominal,ns,nb)) { return h::Problem { .code = 4, .start = 0, .end = ns }; }
    if(!shapes::abi_valid(nominal,bytes::view(source))){return h::Problem{.code=5102,.start=0,.end=ns};}
    var usize shape_index = 0;
    while (shape_index < shapes::count(nominal)) {
        let shapes::Shape shape = shapes::at(nominal,shape_index);
        if(tagged::builtin(shape.type) || arrays::owned(shape.type)){shape_index+=1;continue;}
        let h::Binding b = *v::at<h::Binding>(bindings,shape.binding - 1);
        let usize abi_slot=shapes::abi_find(nominal,shape.type);
        if(abi_slot!=0){let a=shapes::abi_at(nominal,abi_slot-1);if(a.name_start!=b.start || a.name_end!=b.end || a.declaration!=b.declaration){return h::Problem{.code=5102,.start=b.start,.end=b.end};}}
        if (h::binding_code(b.kind) != if(resources::valid(shape.type)){8}else if(enums::valid(shape.type)){7}else{6} || h::type_code(b.type) != shape.type || b.declaration != (shape.type - if(resources::valid(shape.type)){98303}else if(enums::valid(shape.type)){131071}else{255}) as usize || b.signature_count != 0 || b.module_index != module_index) { return h::Problem { .code = 4, .start = b.start, .end = b.end }; }
        shape_index += 1;
    }
    var usize i = 0;
    while (i < nb) {
        let h::Binding b = *v::at<h::Binding>(bindings,i);
        if(h::binding_code(b.kind)==11){if(!b.unsafe_call || b.link_start>=b.link_end || b.link_end>ns || !shapes::foreign_result(nominal,h::type_code(b.type))){return h::Problem{.code=5106,.start=b.start,.end=b.end};}
            let usize rule=shapes::sdk_rule(nominal,bytes::view(source),b.start,b.end,b.link_start,b.link_end);
            if(!ffi::valid(bytes::view(source)[b.link_start..b.link_end]) && rule==0){return h::Problem{.code=5106,.start=b.start,.end=b.end};}
            if(rule!=0 && (!shapes::sdk_result(nominal,bytes::view(source),rule,h::type_code(b.type)) || b.effects!=sdk::effects(rule) || b.signature_count!=sdk::parameter_count(rule))){return h::Problem{.code=5102,.start=b.start,.end=b.end};}
            var usize n=0;while(n<b.signature_count){if(b.signature_start+n>=ntypes || !shapes::foreign_scalar(nominal,h::type_code(v::at<h::Signature>(signatures,b.signature_start+n).type),false) || rule!=0 && !shapes::sdk_parameter(nominal,bytes::view(source),rule,n,h::type_code(v::at<h::Signature>(signatures,b.signature_start+n).type))){return h::Problem{.code=5102,.start=b.start,.end=b.end};}n+=1;}
            var usize other=0;while(other<i){let old=*v::at<h::Binding>(bindings,other);if(h::binding_code(old.kind)==11 && old.link_end<=ns && bytes::equal(bytes::view(source)[old.link_start..old.link_end],bytes::view(source)[b.link_start..b.link_end])){if(h::type_code(old.type)!=h::type_code(b.type) || old.signature_count!=b.signature_count){return h::Problem{.code=5106,.start=b.start,.end=b.end};}n=0;while(n<b.signature_count){if(h::type_code(v::at<h::Signature>(signatures,old.signature_start+n).type)!=h::type_code(v::at<h::Signature>(signatures,b.signature_start+n).type)){return h::Problem{.code=5106,.start=b.start,.end=b.end};}n+=1;}}other+=1;}
        }else if(b.link_start!=0 || b.link_end!=0){return h::Problem{.code=4,.start=b.start,.end=b.end};}
        if (b.must_use && (h::binding_code(b.kind)!=1 && h::binding_code(b.kind)!=5 && h::binding_code(b.kind)!=11 || !must_use::eligible(h::type_code(b.type))) || b.effects>4095 || (b.effects!=0 && !h::function_kind(b.kind) && h::binding_code(b.kind)!=5 && h::binding_code(b.kind)!=8 && h::binding_code(b.kind)!=11) || b.start > b.end || b.end > ns || b.owner > nb || h::type_code(b.type) == h::type_code(.Invalid) || h::type_code(b.type) == h::type_code(.Callable) || b.signature_start > ntypes || b.signature_count > ntypes - b.signature_start) { return h::Problem { .code = 4, .start = b.start, .end = b.end }; }
        if (!h::represented_type(b.type) || !shapes::valid_type(nominal,h::type_code(b.type),true) || ((h::binding_code(b.kind) >= 6 && h::binding_code(b.kind) <= 8) && shapes::find(nominal,h::type_code(b.type)) == 0)) { return h::Problem { .code = 4, .start = b.start, .end = b.end }; }
        if ((h::binding_code(b.kind) >= 6 && h::binding_code(b.kind) <= 8) && shapes::at(nominal,shapes::find(nominal,h::type_code(b.type)) - 1).binding != i + 1) { return h::Problem { .code = 4, .start = b.start, .end = b.end }; }
        if (h::function_kind(b.kind) || (h::binding_code(b.kind) == h::binding_code(.External) || h::binding_code(b.kind)==11) || h::binding_code(b.kind) == h::binding_code(.Struct) || h::binding_code(b.kind) == 7 || h::binding_code(b.kind)==8) { if (b.owner != 0 || b.scope != 0 || b.declaration == 0) { return h::Problem { .code = 4, .start = b.start, .end = b.end }; } }
        else if (b.owner == 0 || b.scope == 0 || h::type_code(b.type) == h::type_code(.Void) || b.signature_count != 0 || !h::function_kind(v::at<h::Binding>(bindings,b.owner - 1).kind)) { return h::Problem { .code = 4, .start = b.start, .end = b.end }; }
        i += 1;
    }
    i = 0; while (i < ntypes) { let h::Type type = v::at<h::Signature>(signatures,i).type; if (!h::represented_type(type) || !shapes::valid_type(nominal,h::type_code(type),false)) { return h::Problem { .code = 4, .start = 0, .end = ns }; } i += 1; }
    i = 0;
    while (i < nf) {
        let h::Function f = *v::at<h::Function>(functions,i);
        if (f.start > f.end || f.end > ns || f.binding == 0 || f.binding > nb || f.body == 0 || f.body > nk || f.parameter_start > na || f.parameter_count > na - f.parameter_start || f.scope == 0) { return h::Problem { .code = 4, .start = f.start, .end = f.end }; }
        let h::Binding b = *v::at<h::Binding>(bindings,f.binding - 1); let h::Block body = *v::at<h::Block>(blocks,f.body - 1);
        if (!h::function_kind(b.kind) || b.module_index != module_index || b.signature_count != f.parameter_count || body.owner != f.binding || body.parent_scope != f.scope) { return h::Problem { .code = 4, .start = f.start, .end = f.end }; }
        var usize j = 0;
        while (j < f.parameter_count) {
            let usize id = *v::at<usize>(edges,f.parameter_start + j);
            if (id == 0 || id > nb) { return h::Problem { .code = 4, .start = f.start, .end = f.end }; }
            let h::Binding parameter = *v::at<h::Binding>(bindings,id - 1);
            if (h::binding_code(parameter.kind) != if(h::binding_code(b.kind)==9){10}else{2} || parameter.owner != f.binding || parameter.scope != f.scope || h::type_code(parameter.type) != h::type_code(v::at<h::Signature>(signatures,b.signature_start + j).type)) { return h::Problem { .code = 4, .start = parameter.start, .end = parameter.end }; }
            j += 1;
        }
        i += 1;
    }
    return h::Problem{.code=0,.start=0,.end=0};
}
