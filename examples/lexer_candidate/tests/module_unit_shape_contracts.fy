module tests::module_unit_shape_contracts;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let bound=switch(binding::bind(sources,ref metadata,0,1)){case .ok(b):{b}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(model::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources);}
@effects(console) fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{discard io::write("unexpected success\n");assert(false);}case .err(p):{if(p.code!=code || p.module_index!=0 || !bytes::equal(text[p.start..p.end],at)){discard io::write_u32(p.code);discard io::write(" at ");discard io::write_bytes(text[p.start..p.end]);discard io::write("\n");assert(false);}}};}


@test fn void unused_nominals_and_valid_inline_arrays_are_retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;struct Leaf{i64 n;}struct Box{Leaf[2] items;i32[2] row;}resource R{Box value;cleanup{}}enum Flag{A;B;}struct Empty{}fn void main(){}" )).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let p=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(interfaces::shape_count(ref p)==5 && interfaces::field_count(ref p)==6 && interfaces::work(ref p)<1048576);
}
@test @effects(console) fn void malformed_unused_fields_fail_before_body_typing(){
    failure(b"module main;struct S{i32 n;i64 n;}fn void main(){}",102,b"n");
    failure(b"module main;resource R{i32 n;i64 n;cleanup{}}fn void main(){}",102,b"n");
    failure(b"module main;enum Flag{A;A;}fn void main(){}",102,b"A");
    failure(b"module main;enum Flag{A{i32 n;i64 n;};}fn void main(){}",102,b"n");
    failure(b"module main;struct S{ref i32 n;}fn void main(){}",1003,b"ref i32");
    failure(b"module main;resource R{ref i32 n;cleanup{}}fn void main(){}",1003,b"ref i32");
    failure(b"module main;resource R{i32[2] n;cleanup{}}fn void main(){}",2,b"i32[2]");
    failure(b"module main;resource R{i32 n;cleanup{}}struct S{R r;}fn void main(){}",1002,b"R");
    failure(b"module main;resource R{i32 n;cleanup{}}struct S{Option<R> r;}fn void main(){}",1002,b"Option<R>");
    failure(b"module main;struct S{i32[2][2] n;}fn void main(){}",2,b"i32[2][2]");
    failure(b"module main;enum E{A;}struct S{E[2] n;}fn void main(){}",2,b"E[2]");
    failure(b"module main;struct Empty{}struct S{Empty[2] n;}fn void main(){}",2,b"Empty[2]");
    failure(b"module main;struct Leaf{i32[2] row;}struct S{Leaf[2] n;}fn void main(){}",2,b"Leaf[2]");
    failure(b"module main;@abilities(unknown)resource R{i32 n;cleanup{}}fn void main(){}",1107,b"unknown");
}
@test fn void many_deferred_calls_fit_the_original_work_budget(){
    var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module main;fn void sink(){}fn void main(){").is_ok);var usize i=0;while(i<900){assert(bytes::append(ref mut text,b"sink();").is_ok);i+=1;}i=0;while(i<900){assert(bytes::append(ref mut text,b"defer sink();").is_ok);i+=1;}assert(bytes::append(ref mut text,b"}").is_ok);let p=switch(checked(bytes::view(ref text))){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_calls(ref p)==1800 && values::checked_arguments(ref p)==0 && values::unresolved(ref p)==0 && values::work(ref p)<1048576);let c=values::at(ref p,1);assert(model::checked_consumers(c)==1800);
}
