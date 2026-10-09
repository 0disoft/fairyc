module tests::module_unit_matches;
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

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> projected(ref v::Vec<syntax::Parsed> sources,usize first,usize stop){let metadata=switch(signatures::build(sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(sources,ref metadata,first,stop)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(sources,ref metadata,ref ready);}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);return projected(ref sources,0,1);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void builtin_switches_bind_payloads_and_contextualize_arm_values(){
    let p=switch(checked(b"module main;fn i64 option(Option<i64> input){return switch(input){case .some(n):{n}case .none:{4000000000}};}fn i64 result(Result<i64,bool> input){return switch(input){case .ok(n):{n}case .err(error):{if(error){4000000001}else{4000000002}}};}fn i64 shadow(Result<i64,bool> input){let bool n=true;discard n;return switch(input){case .ok(n):{n}case .err(_):{4000000003}};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;var usize patterns=0;var usize matches=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==15){assert(e.state==1 && values::type_at(ref p,e.type).scalar==2 && e.target!=0 && e.place.root==0);matches+=1;}if(e.kind==17){assert(e.state==2 && e.type==0 && e.target!=0 && e.expected==e.target);patterns+=1;}j+=1;}j=1;while(j<=model::local_count(c)){let local=model::local_at(c,j);if(local.kind==4){assert(local.type!=0 && local.expected==local.type && local.initializer==0);}j+=1;}i+=1;}assert(matches==3 && patterns==5);
}
@test fn void nominal_payload_bindings_feed_fields_calls_and_inferred_locals(){
    let p=switch(checked(b"module main;struct Packet{i64 n;}fn i64 take(Packet packet){return packet.n;}fn i64 result(Result<Packet,bool> input){let packet=switch(input){case .ok(packet):{packet}case .err(_):{Packet{.n=4000000000}}};return take(packet);}fn i64 option(Option<Packet> input){return switch(input){case .some(packet):{packet.n}case .none:{4000000001}};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,1);let local=model::local_at(c,3);assert(local.expected==0 && values::type_at(ref p,local.type).kind==2);
}
@test fn void result_return_and_false_assert_rules_preserve_option_difference(){
    let p=switch(checked(b"module main;fn i64 result(Result<i64,bool> input){return switch(input){case .ok(n):{n}case .err(_):{return 4000000000;}};}fn i64 optional(Option<i64> input){return switch(input){case .some(n):{n}case .none:{assert(false);}};}fn void all_return(Result<i64,bool> input){switch(input){case .ok(n):{return;}case .err(_):{return;}};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==5 && values::unresolved(ref p)==0);let c=values::at(ref p,2);assert(model::checked_consumers(c)==1);var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==15){assert(e.state==1 && values::type_at(ref p,e.type).scalar==7);}i+=1;}
}
@test fn void enum_patterns_keep_provider_fields_and_separate_symbol_roots(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(p::Code code){return switch(code){case .More{.valid=flag,.n=n}:{if(flag){n}else{4000000000}}case .Empty:{0}};}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub enum Code{Empty;More{i64 n;bool valid;};}")).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);var usize i=1;var usize fields=0;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==33 || e.kind==17){assert(e.state==2 && e.type==0 && e.field.id!=0 && e.field.module_index==1 && e.field.declaration==2);let f=syntax::field_at(v::at<syntax::Parsed>(ref sources,1),e.field.source_index);assert(f.kind==4);if(e.kind==17){assert(e.target==e.expected && e.place.root>1 && e.place.scope>1);}fields+=1;}i+=1;}assert(fields==4);
}
@test fn void generic_and_cleanup_switch_bindings_keep_provider_contexts(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(Result<i64,bool> input){return p::go<i64>(input);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{let Option<i64> opt=.some(4000000000);let i64 value=switch(opt){case .some(n):{n}case .none:{0}};discard value;return;}}pub fn T go<T>(Result<T,bool> input){let Item item=Item{.n=1};discard item;return switch(input){case .ok(n):{n}case .err(_):{assert(false);}};}" )).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==15){assert(model::frame(c).module_index==1 && e.state==1);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(original.start==e.start && original.end==e.end);count+=1;}j+=1;}i+=1;}assert(count==2);
}
@test fn void switch_errors_keep_reference_codes_and_spans(){
    failure(b"module main;fn i32 main(){return switch(1){case .ok(n):{n}case .err(_):{0}};}",2,b"1");
    failure(b"module main;fn i32 main(Option<i32> input){return switch(input){case .some(n):{n}};}",2,b"switch(input){case .some(n):{n}}");
    failure(b"module main;fn i32 main(Option<i32> input){return switch(input){case .some(n):{n}case .some(m):{m}};}",2,b".some(m)");
    failure(b"module main;fn i32 main(Option<i32> input){return switch(input){case .some:{0}case .none:{0}};}",2,b".some");
    failure(b"module main;fn i32 main(Option<i32> input){return switch(input){case .some(n):{n}case .none(m):{0}};}",2,b".none(m)");
    failure(b"module main;fn i32 main(Result<i32,bool> input){return switch(input){case .ok(n):{n}case .err:{0}};}",2,b"case .err:{0}");
    failure(b"module main;fn i32 main(Option<i32> input){return switch(input){case .some(n):{n}case .none:{true}};}",1006,b"{true}");
    failure(b"module main;fn i32 main(Option<i32> input){return switch(input){case .some(n):{return n;}case .none:{0}};}",1006,b"{return n;}");
    failure(b"module main;fn i32 main(Result<i32,bool> input){return switch(input){case .ok(n):{if(true){return n;}}case .err(_):{0}};}",1006,b"{if(true){return n;}}");
    failure(b"module main;fn u8 main(Option<u8> input){return switch(input){case .some(n):{n}case .none:{256}};}",1013,b"256");
    failure(b"module main;enum Code{A;B;}fn i32 main(Code code){return switch(code){case .A:{1}};}",2,b"switch(code){case .A:{1}}");
    failure(b"module main;enum Code{A;B;}fn i32 main(Code code){return switch(code){case .A:{1}case .Missing:{2}};}",2,b".Missing");
    failure(b"module main;enum Code{A;B;}fn i32 main(Code code){return switch(code){case .A:{1}case .A:{2}};}",2,b".A");
    failure(b"module main;enum Code{Empty;More{i32 n;bool valid;};}fn i32 main(Code code){return switch(code){case .Empty:{0}case .More{.n=n}:{n}};}",2,b".More{.n=n}");
    failure(b"module main;enum Code{Empty;More{i32 n;};}fn i32 main(Code code){return switch(code){case .Empty:{0}case .More{.n=a,.n=b}:{a}};}",2,b".n=b");
    failure(b"module main;enum Code{A;B;}fn i32 main(Code code){return switch(code){case .A:{assert(false);}case .B:{1}};}",1006,b"{assert(false);}");
}
