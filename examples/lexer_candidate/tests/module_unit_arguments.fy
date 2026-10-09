module tests::module_unit_arguments;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_call_closure as closure;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn Result<values::Plan,model::Problem> checked(ref u8[] text){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return checked(b"module empty;");}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return checked(b"module empty;");}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return checked(b"module empty;");}};return values::build(ref sources,ref metadata,ref ready);}
fn void failure(ref u8[] text,u32 code,ref u8[] at){switch(checked(text)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==0 && bytes::equal(text[p.start..p.end],at));}};}
@test fn void contextual_arguments_keep_order_labels_and_original_ids(){
    let text=b"module main;fn u8 leaf(u8 n,i64 wide){return n;}fn i64 large(i64 n){return n;}fn u8 main(){let copy=leaf(.n=255,.wide=4000000000+1);discard large(-9223372036854775808);return copy;}";
    let p=switch(checked(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::invocation_count(ref p)==2 && values::arguments(ref p)==3 && values::checked_calls(ref p)==2 && values::checked_arguments(ref p)==3);var usize i=0;var usize labeled=0;
    while(i<values::count(ref p)){let c=values::at(ref p,i);var usize j=0;while(j<model::argument_count(c)){let a=model::argument_at(c,j);let e=model::expression_at(c,a.expression);assert(a.checked && a.source_index==e.source_index && a.start==e.start && a.end==e.end && a.expected==a.actual);let call=model::call_at(c,a.invocation-1);assert(call.checked && a.position<call.argument_count && call.argument_start+a.position==j);if(a.label_end!=0){assert(bytes::equal(text[a.label_start..a.label_end],if(a.position==0){b"n"}else{b"wide"}));labeled+=1;}j+=1;}i+=1;}assert(labeled==2);
}
@test fn void argument_count_type_and_range_errors_keep_argument_spans(){
    failure(b"module main;fn u8 leaf(u8 n){return n;}fn u8 main(){return leaf(true);}",1006,b"true");
    failure(b"module main;fn u8 leaf(u8 n){return n;}fn u8 main(){return leaf(256);}",1013,b"256");
    failure(b"module main;fn u8 leaf(u8 n){return n;}fn u8 main(){return leaf();}",1010,b"leaf()");
    failure(b"module main;fn u8 leaf(u8 n){return n;}fn u8 main(){return leaf(1,2);}",1010,b"leaf(1,2)");
    failure(b"module main;fn i32 leaf(i32 n){return n;}fn i32 main(){return leaf(leaf);}",1006,b"leaf");
}
@test fn void labels_validate_positions_and_private_ordinary_admission(){
    failure(b"module main;fn u8 leaf(u8 n,u8 m){return n;}fn u8 main(){return leaf(.m=1,.n=2);}",1014,b".m=1");
    failure(b"module main;pub fn u8 leaf(u8 n){return n;}fn u8 main(){return leaf(.n=1);}",1014,b"leaf(.n=1)");
    failure(b"module main;fn T leaf<T>(T n){return n;}fn u8 main(){return leaf<u8>(.n=1);}",1014,b"leaf<u8>(.n=1)");
}
@test fn void shared_reborrows_are_recorded_and_reverse_conversion_is_rejected(){
    let p=switch(checked(b"module main;fn void read(ref i32 n){return;}fn void main(ref mut i32 n){read(n);return;}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::checked_calls(ref p)==1 && values::checked_arguments(ref p)==1 && values::shared_arguments(ref p)==1);
    failure(b"module main;fn void write(ref mut i32 n){return;}fn void main(ref i32 n){write(n);return;}",1006,b"n");
}
@test fn void dereferences_contextualize_literal_peers_in_arguments(){
    let p=switch(checked(b"module main;fn i64 leaf(i64 n){return n;}fn i64 main(ref i64 n){return leaf(4000000000+*n);}")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::invocation_count(ref p)==1 && values::checked_calls(ref p)==1 && values::checked_arguments(ref p)==1 && values::unresolved(ref p)==0);
}
@test fn void generic_and_cleanup_calls_use_concrete_provider_parameters(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn u8 main(){return p::go<u8>(255);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{u8 n;cleanup{discard helper(2);return;}}fn u8 helper(u8 n){return n;}pub fn T go<T>(T n){let Item item=Item{.n=1};discard item;return helper(n);}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(values::count(ref p)==3 && values::invocation_count(ref p)==3 && values::checked_calls(ref p)==3 && values::checked_arguments(ref p)==3);var usize i=0;while(i<values::count(ref p)){let c=values::at(ref p,i);let a=model::argument_at(c,0);let call=model::call_at(c,0);assert(a.checked && values::type_at(ref p,a.expected).scalar==3 && call.target_module==1);i+=1;}
}
