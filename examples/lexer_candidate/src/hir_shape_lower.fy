module hir_shape_lower;
import hir_shapes as shapes;
import tagged_types as tagged;
import hir_array_shapes as arrays;
import type_check as typing;
import nominal_types as nominal;
import name_resolution as names;
import syntax as syntax;
import std::vec as v;

pub fn Result<shapes::Table,u32> collect(ref typing::Typed input) {
    let ref names::Resolved resolved = typing::resolved(input);
    let ref syntax::Parsed parsed = names::parsed(resolved);
    var v::Vec<shapes::Shape> records = v::empty<shapes::Shape>();
    var v::Vec<shapes::Field> fields = v::empty<shapes::Field>();
    var usize index = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration d = syntax::declaration_at(parsed,index);
        if (d.kind == 4 || d.kind == 11) {
            var usize binding = 0; var usize id = 1;
            while (id <= names::symbol_count(resolved)) {
                let names::Symbol s = names::symbol_at(resolved,id);
                if ((s.kind == 6 || s.kind == 7) && s.declaration == index + 1) { binding = id; } id += 1;
            }
            if (binding == 0 || d.field_start != v::len<shapes::Field>(ref fields)) { return .err(4); }
            let u32 owner = typing::symbol_type(input,binding);
            if (!v::push<shapes::Shape>(ref mut records,shapes::Shape { .type = owner, .binding, .field_start = d.field_start, .field_count = d.field_count, .public = d.public }).is_ok) { return .err(3); }
            var usize j = 0;
            while (j < d.field_count) {
                let f = syntax::field_at(parsed,d.field_start + j);
                if (!v::push<shapes::Field>(ref mut fields,shapes::Field { .owner, .type = nominal::resolve(parsed,f.type_root), .start = f.name_start, .end = f.name_end, .payload_start = f.payload_start, .payload_count = f.payload_count }).is_ok) { return .err(3); } j += 1;
            }
        } else if (d.kind != 1 && d.kind != 2 && d.kind != 3) { return .err(4); }
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
    return .ok(shapes::assemble(records,fields));
}
