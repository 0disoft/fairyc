module tests::module_unit_origin_bindings;
import module_unit_origin_inputs as inputs;
import module_unit_origin_factory as input_factory;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);primary(sources,metadata)}};}
@test fn void reassigned_reference_keeps_its_lexical_owner(){
    let text=b"module main;fn ref i32 select(ref i32 a,ref i32 b,bool flag){var ref i32 r=a;if(flag){r=b;};return r;}";
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=primary(ref sources,ref metadata);let observed=switch(input_factory::build(ref sources,ref metadata,ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(inputs::frame_count(ref observed)==1);let body=inputs::body(ref observed,ref p,1);var usize i=1;var usize aliases=0;
    while(i<=m::expression_count(body)){let e=m::expression_at(body,i);if(e.kind==2 && bytes::equal(text[e.start..e.end],b"r")){assert(e.binding==4 && e.place.root==1);let owner=switch(inputs::binding_owner(ref observed,1,e)){case .ok(n):{n}case .err(_):{assert(false);return;}};let local=inputs::owner(ref observed,owner);assert(local.local==4 && local.kind==3 && local.borrowed && bytes::equal(text[local.start..local.end],b"r"));aliases+=1;}else if(e.kind!=2){assert(e.binding==0);}i+=1;}assert(aliases==2 && m::checked_assignments(body)==1);
    var e=m::expression_at(body,1);e.binding=m::local_count(body)+1;e.kind=2;switch(inputs::binding_owner(ref observed,1,e)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.start==e.start && p.end==e.end);}};
    e.binding=1;switch(inputs::binding_owner(ref observed,0,e)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12);}};
}
@test fn void shadowed_reference_aliases_keep_distinct_bindings_and_scopes(){
    let text=b"module main;fn void main(){let i32 a=1;let ref i32 r=ref a;{let i32 b=2;let ref i32 r=ref b;discard r;}discard r;}";
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=primary(ref sources,ref metadata);let observed=switch(input_factory::build(ref sources,ref metadata,ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};let body=inputs::body(ref observed,ref p,1);var usize i=1;var usize inner=0;var usize outer=0;
    while(i<=m::expression_count(body)){let e=m::expression_at(body,i);if(e.kind==2 && bytes::equal(text[e.start..e.end],b"r")){let owner=switch(inputs::binding_owner(ref observed,1,e)){case .ok(n):{n}case .err(_):{assert(false);return;}};if(e.binding==4){assert(e.place.root==3);inner=owner;}else{assert(e.binding==2 && e.place.root==1);outer=owner;}}else if(e.binding==0){assert(switch(inputs::binding_owner(ref observed,1,e)){case .ok(n):{n==0}case .err(_):{false}});}i+=1;}
    assert(inner!=0 && outer!=0 && inner!=outer && inputs::owner(ref observed,inner).scope!=inputs::owner(ref observed,outer).scope);
}
