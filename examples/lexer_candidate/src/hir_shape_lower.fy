module hir_shape_lower;
import hir_shapes as shapes;
import tagged_types as tagged;
import hir_array_shapes as arrays;
import type_check as typing;
import nominal_types as nominal;
import name_resolution as names;
import syntax as syntax;
import repr_c_declarations as repr;
import std::vec as v;
import sdk_scopes as sdk;

// Removed templates leave gaps in the parser field arena. Publish compact owned
// field IDs and remap references rather than retaining uninstantiated layouts.
pub fn usize field_id(ref syntax::Parsed p,usize source_id){
    if(source_id==0){return 0;}var usize offset=0;var usize i=0;
    while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==4 || d.kind==6 || d.kind==11){
        if(source_id>d.field_start && source_id<=d.field_start+d.field_count){return offset+source_id-d.field_start;}offset+=d.field_count;
    }i+=1;}
    return if(source_id>syntax::field_count(p)){offset+source_id-syntax::field_count(p)}else{0};
}
pub fn Result<shapes::Table,u32> collect(ref typing::Typed input) {
    let ref names::Resolved resolved = typing::resolved(input);
    let ref syntax::Parsed parsed = names::parsed(resolved);
    var v::Vec<shapes::Shape> records = v::empty<shapes::Shape>();
    var v::Vec<shapes::Field> fields = v::empty<shapes::Field>();
    var v::Vec<shapes::Abi> abi=v::empty<shapes::Abi>();
    var usize index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration d = syntax::declaration_at(parsed,index);
        if (d.kind == 4 || d.kind == 6 || d.kind == 11) {
            var usize binding = 0; var usize id = 1;
            while (id <= names::symbol_count(resolved)) {
                let names::Symbol s = names::symbol_at(resolved,id);
                if ((s.kind == 6 || s.kind == 7 || s.kind==9) && s.declaration == index + 1) { binding = id; } id += 1;
            }
            if (binding == 0) { return .err(4); }
            let u32 owner = typing::symbol_type(input,binding);
            let permission=repr::read(parsed,index);
            if(permission.repr_c && !v::push<shapes::Abi>(ref mut abi,shapes::Abi{.type=owner,.declaration=index+1,.name_start=d.name_start,.name_end=d.name_end,.marker_start=permission.start,.marker_end=permission.end}).is_ok){return .err(3);}
            if (!v::push<shapes::Shape>(ref mut records,shapes::Shape { .type = owner, .binding, .field_start = v::len<shapes::Field>(ref fields), .field_count = d.field_count, .public = d.public }).is_ok) { return .err(3); }
            var usize j = 0;
            while (j < d.field_count) {
                let f = syntax::field_at(parsed,d.field_start + j);
                if (!v::push<shapes::Field>(ref mut fields,shapes::Field { .owner, .type = nominal::resolve(parsed,f.type_root), .start = f.name_start, .end = f.name_end, .payload_start = if(f.payload_count==0){0}else{field_id(parsed,f.payload_start+1)-1}, .payload_count = f.payload_count }).is_ok) { return .err(3); } j += 1;
            }
        } else if (d.kind != 12 && d.kind != 1 && d.kind != 2 && d.kind != 3 && d.kind!=9) { return .err(4); }
        index += 1;
    }
    var v::Vec<u32> instances=v::empty<u32>();
    if(!tagged::instances(parsed,ref mut instances)){return .err(2);}index=0;
    while(index<v::len<u32>(ref instances)){
        let u32 type=*v::at<u32>(ref instances,index);let usize root=tagged::first(parsed,type);let node=syntax::type_at(parsed,root);
        let usize start=v::len<shapes::Field>(ref fields);let bool option=tagged::option(type);let usize count=if(option){3}else{4};
        if(start>1024-count || v::len<shapes::Shape>(ref records)>=512){return .err(2);}
        if(!v::push<shapes::Shape>(ref mut records,shapes::Shape{.type,.binding=0,.field_start=start,.field_count=count,.public=false}).is_ok
            || !v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=tagged::payload_type(type,true),.start=node.start,.end=node.end,.payload_start=0,.payload_count=0}).is_ok
            || !v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=node.start,.end=node.end,.payload_start=start,.payload_count=1}).is_ok){return .err(3);}
        if(!option && !v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=tagged::failure(type),.start=node.start,.end=node.end,.payload_start=0,.payload_count=0}).is_ok){return .err(3);}
        if(!v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=type,.type=0,.start=node.start,.end=node.end,.payload_start=if(option){0}else{start+2},.payload_count=if(option){0}else{1}}).is_ok){return .err(3);}
        index+=1;
    }
    let u32 array_code=arrays::collect(input,ref mut records,ref mut fields);if(array_code!=0){return .err(array_code);}
    let scopes=switch(sdk::collect(parsed)){case .ok(s):{s}case .err(code):{return .err(code);}};
    return .ok(shapes::assemble_sdk(records,fields,abi,scopes));
}
