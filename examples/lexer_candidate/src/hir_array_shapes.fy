module hir_array_shapes;
import array_types as arrays;
import hir_shapes as shapes;
import type_check as typing;
import name_resolution as names;
import syntax as syntax;
import std::vec as v;

fn u32 append(ref syntax::Parsed p,u32 type,ref mut v::Vec<shapes::Shape> records,ref mut v::Vec<shapes::Field> fields) {
    if(!arrays::owned(type)){return 0;}var usize i=0;
    while(i<v::len<shapes::Shape>(records)){if(v::at<shapes::Shape>(records,i).type==type){return 0;}i+=1;}
    if(v::len<shapes::Shape>(records)>=512 || v::len<shapes::Field>(fields)>=1024){return 2;}
    let u32 child=arrays::source_element(p,type);if(child==0){return 4;}
    let usize offset=v::len<shapes::Field>(fields);
    if(!v::push<shapes::Shape>(records,shapes::Shape{.type,.binding=0,.field_start=offset,.field_count=1,.public=false}).is_ok
        || !v::push<shapes::Field>(fields,shapes::Field{.owner=type,.type=child,.start=0,.end=syntax::source_view(p).len,.payload_start=0,.payload_count=0}).is_ok){return 3;}
    return append(p,child,records,fields);
}
pub fn u32 collect(ref typing::Typed input,ref mut v::Vec<shapes::Shape> records,ref mut v::Vec<shapes::Field> fields) {
    let ref names::Resolved resolved=typing::resolved(input);let ref syntax::Parsed p=names::parsed(resolved);var usize i=1;
    while(i<=syntax::type_count(p)){let u32 code=append(p,arrays::resolve(p,i),records,fields);if(code!=0){return code;}i+=1;}
    i=1;while(i<=names::symbol_count(resolved)){let u32 code=append(p,typing::symbol_type(input,i),records,fields);if(code!=0){return code;}i+=1;}
    i=1;while(i<=syntax::expression_count(p)){let u32 code=append(p,typing::expression_type(input,i),records,fields);if(code!=0){return code;}i+=1;}
    return 0;
}
