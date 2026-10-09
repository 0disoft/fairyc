module tests::module_unit_control;
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
@test fn void contextual_if_blocks_feed_inference_arguments_and_returns(){
    let p=switch(checked(b"module main;fn i64 take(i64 value){return value;}fn i64 main(){let bool b=true;let i64 n=if(b){4000000000}else if(false){4000000001}else{4000000002};let inferred=if(b){1}else{2};discard inferred;return take(if(b){n}else{4000000003});}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==2 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,1);assert(model::checked_consumers(c)==1);var usize count=0;var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==13 || e.kind==12){assert(e.state==1 && e.type!=0 && e.place.root==0);count+=1;}if(e.kind==14){assert(e.state==2 && e.type==0);}i+=1;}assert(count==7);
}
@test fn void block_tails_and_unsafe_blocks_keep_context(){
    let p=switch(checked(b"module main;fn i64 main(){let i64 n={let i64 value=4000000000;value};return unsafe{if(true){n}else{4000000001}};}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==1 && values::unresolved(ref p)==0);let c=values::at(ref p,0);assert(model::local_at(c,2).type!=0);var usize i=1;while(i<=model::expression_count(c)){let e=model::expression_at(c,i);if(e.kind==12 || e.kind==31){assert(e.state==1 && e.expected==e.type && e.place.root==0);}i+=1;}
}
@test fn void statement_consumers_check_void_control_and_bool_headers(){
    let p=switch(checked(b"module main;fn bool yes(){return true;}fn void main(){if(yes()){assert(true);}else{assert(false);}if(false){return;}while(yes()){break;}for(;true;){discard if(true){1}else{2};break;}return;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);let c=values::at(ref p,1);assert(model::consumer_count(c)==model::checked_consumers(c) && model::consumer_count(c)==7);var usize i=0;while(i<model::consumer_count(c)){let s=model::consumer_at(c,i);let e=model::expression_at(c,s.expression);assert(s.checked && s.actual==e.type && s.source_index==e.source_index);if(s.expected!=0){assert(values::type_at(ref p,s.actual).scalar==6);}i+=1;}
}
@test fn void control_errors_keep_reference_codes_and_spans(){
    failure(b"module main;fn i32 main(){return if(1){2}else{3};}",1007,b"1");
    failure(b"module main;fn i32 main(){return if(true){1}else{false};}",1006,b"{false}");
    failure(b"module main;fn i32 main(){return if(true){return 1;}else{2};}",1006,b"{2}");
    failure(b"module main;fn i32 main(){return if(true){assert(false);}else{2};}",1006,b"{2}");
    failure(b"module main;fn void main(){let i32 n=if(true){1};return;}",1006,b"if(true){1}");
    failure(b"module main;fn i32 main(){return if(true){1};}",1009,b"if(true){1}");
    failure(b"module main;fn u8 main(){return if(true){256}else{1};}",1013,b"256");
    failure(b"module main;fn void main(){if(1){return;}return;}",1007,b"1");
    failure(b"module main;fn void main(){assert(1);return;}",1007,b"1");
    failure(b"module main;fn void main(){while(1){break;}return;}",1007,b"1");
    failure(b"module main;fn void main(){for(;1;){break;}return;}",1007,b"1");
    failure(b"module main;fn bool yes(){return true;}fn void main(){for(;yes();){break;}return;}",2,b"yes()");
    failure(b"module main;fn void main(){discard if(true){1}else{false};return;}",1006,b"{false}");
    failure(b"module main;fn void main(){discard [1,true];return;}",1006,b"true");
    failure(b"module main;fn void main(){let flag=true;let n=if(flag){1}else{2};if(true){n+true};return;}",1006,b"true");
    failure(b"module main;fn void main(){let bool b=true;let ref bool r=ref b;for(;*r;){break;}return;}",2,b"*r");
}
@test fn void provider_generic_and_cleanup_controls_keep_original_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i64 main(){return p::go<i64>(4000000000);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{if(true){assert(true);}else{assert(false);}return;}}pub fn T go<T>(T n){let Item item=Item{.n=1};discard item;return if(true){n}else{n};}" )).is_ok);let p=switch(projected(ref sources,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::checked_returns(ref p)==3 && values::unresolved(ref p)==0);var usize i=0;var usize count=0;while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=1;while(j<=model::expression_count(c)){let e=model::expression_at(c,j);if(e.kind==13){assert(model::frame(c).module_index==1 && e.state==1);let original=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),e.source_index);assert(original.start==e.start && original.end==e.end);count+=1;}j+=1;}assert(model::checked_consumers(c)==model::consumer_count(c));i+=1;}assert(count==2);
}
