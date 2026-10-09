module tests::module_unit_sdk_calls;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import file_source as files;
import module_types as types;
import module_unit_value_storage as storage;
import module_unit_interfaces as interfaces;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}

@effects(filesystem) fn bytes::Buffer sdk(){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};return switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,console) fn values::Plan planned(ref u8[] text){let sdk=sdk();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref sdk))).is_ok);return switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);discard io::write(" at ");discard io::write_usize(p.start);discard io::write(".. ");discard io::write_usize(p.end);discard io::write("\n");assert(false);return planned(b"module main;");}};}

@test @effects(filesystem,console) fn void ordinary_sdk_view_signatures_feed_canonical_caller_arguments_and_results(){
    let sdk=sdk();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();let text=b"module main;import std::vec as v;fn void main(){var v::Vec<u8> x=v::empty<u8>();let ref u8[] a=v::view_u8(ref x);discard a;let ref mut u8[] b=v::view_mut_u8(ref mut x);discard b;return;}";assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref sdk))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(p):{discard io::write_u32(p.code);assert(false);return;}};let shared=signatures::lookup(ref metadata,0,b"v::view_u8");let exclusive=signatures::lookup(ref metadata,0,b"v::view_mut_u8");assert(shared.code==0 && exclusive.code==0 && signatures::vector_count(ref metadata)==5);
    let a=signatures::at(ref metadata,shared.item);let b=signatures::at(ref metadata,exclusive.item);assert(a.module_index==1 && b.module_index==1 && a.public && b.public && !a.foreign && !b.foreign && a.parameter_count==1 && b.parameter_count==1);assert(signatures::type_at(ref metadata,a.result).kind==10 && signatures::type_at(ref metadata,b.result).kind==11);let sa=signatures::type_at(ref metadata,signatures::parameter_at(ref metadata,a.parameter_start));let sb=signatures::type_at(ref metadata,signatures::parameter_at(ref metadata,b.parameter_start));assert(sa.kind==7 && sb.kind==8 && sa.first==sb.first && signatures::type_at(ref metadata,sa.first).kind==13);
    let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);discard io::write_usize(p.start);assert(false);return;}};var usize i=0;var usize found=0;while(i<values::count(ref p)){let c=values::at(ref p,i);if(model::frame(c).module_index==0 && model::frame(c).kind==3){assert(model::checked_calls(c)==3);var usize j=0;while(j<model::call_count(c)){let call=model::call_at(c,j);if(call.kind==1 && call.target_module==1 && (call.target_declaration==a.declaration || call.target_declaration==b.declaration)){assert(call.checked && call.argument_count==1);let e=model::expression_at(c,call.expression);assert(e.state==1 && e.type==if(call.target_declaration==a.declaration){a.result}else{b.result});let arg=model::argument_at(c,call.argument_start);assert(arg.checked && arg.expected==arg.actual && arg.invocation==j+1);found+=1;}j+=1;}}i+=1;}assert(found==2);
}
@test @effects(filesystem,console) fn void ordinary_sdk_storage_slice_bodies_keep_owner_roots_and_original_coordinates(){
    let sdk=sdk();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref sdk))).is_ok);let p=switch(projected(ref sources,1,2)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);discard io::write_usize(p.start);assert(false);return;}};
    var usize count=0;var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==6 && storage::borrow(e.builtin)){assert(model::frame(c).kind==3 && model::frame(c).module_index==1 && e.state==1 && e.builtin.slice && e.builtin.unsafe_context && e.place.root==1 && e.place.indirect && e.place.mutable==(e.builtin.kind==107));let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(e.start==original.start && e.end==original.end);let t=values::type_at(ref p,e.type);let element=values::type_at(ref p,e.builtin.element);assert(element.kind==1 && element.scalar>=1 && element.scalar<=6);assert(if(element.scalar==3){t.kind==if(e.builtin.kind==107){11}else{10}}else{t.kind==15});count+=1;}j+=1;}i+=1;}assert(count==10);
}

@effects(filesystem) fn void sdk_failure(ref u8[] text,u32 code,ref u8[] token){let sdk=sdk();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref sdk))).is_ok);switch(projected(ref sources,0,1)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],token));}};}
@test @effects(filesystem) fn void ordinary_sdk_call_errors_keep_caller_types_arity_and_original_spans(){
    sdk_failure(b"module main;import std::vec as v;fn void main(){let v::Vec<u8> x=v::empty<u8>();let ref i64[] a=v::view_u8(ref x);discard a;}",1006,b"v::view_u8(ref x)");
    sdk_failure(b"module main;import std::vec as v;fn void main(){let v::Vec<u8> x=v::empty<u8>();discard v::view_u8(ref x,0);}",1010,b"v::view_u8(ref x,0)");
    sdk_failure(b"module main;import std::vec as v;fn void main(){discard v::view_u8(1);}",1006,b"1");
}
