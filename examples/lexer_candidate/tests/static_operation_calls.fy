module tests::static_operation_calls;
import scanner as scanner;
import syntax as syntax;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_value_model as model;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
@test @effects(console) fn void selected_ordinary_calls_keep_exact_types(){
var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module root;@interface fn T transform<T>(T value){}@implements(transform,i32) fn i32 twice(i32 x){return x*2;}@implements(transform,i64) fn i64 triple(i64 y){return y*3;}fn T algorithm<T>(T value){return transform<T>(value);}fn i32 main(){assert(algorithm<i64>(14)==42);return algorithm<i32>(21);}")).is_ok);
let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);assert(false);return;}};
let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return;}};
let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);assert(false);return;}};
let checked=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);assert(false);return;}};
var usize operations=0;var usize i=0;while(i<values::count(ref checked)){let body=values::at(ref checked,i);let f=model::frame(body);if(f.declaration==2){assert(model::call_count(body)==1);let call=model::call_at(body,0);let scalar=values::type_at(ref checked,f.result).scalar;assert(call.target_declaration==if(scalar==1){3}else{4});operations+=1;}i+=1;}assert(operations==2);
}

@test fn void provider_bodies_keep_ordinary_type_checks(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module root;@interface fn T op<T>(T x){}@implements(op,i32) fn i32 broken(i32 x){return true;}fn i32 main(){return op<i32>(1);}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(switch(values::build(ref sources,ref metadata,ref ready)){case .ok(_):{false}case .err(e):{e.code==1009}});
}
