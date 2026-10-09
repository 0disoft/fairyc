module tests::module_unit_cleanups;
import module_unit_bodies as bodies;
import module_unit_body_ast as ast;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn bodies::Plan retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;@effects(console)fn i32 main(){let p::Owner value=p::make();return 0;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;@abilities(drop,store)@effects(console)pub resource Owner{i32 n;cleanup{let i32 value=id<i32>(this.n);discard value;}}pub fn Owner make(){return Owner{.n=7};}@effects(console)fn T id<T>(T n){return n;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return retained();}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return retained();}};let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return retained();}};
    return switch(bodies::build(ref sources,ref metadata,ref imported)){case .ok(p):{p}case .err(_):{assert(false);retained()}};
}
@test fn void owned_resource_receiver_authority_and_private_helper_survive_source_removal(){
    let p=retained();assert(bodies::count(ref p)==1 && bodies::cleanup_count(ref p)==1 && bodies::source_count(ref p)==1);let c=bodies::cleanup_at(ref p,1);let info=bodies::cleanup_info(c);let a=bodies::cleanup_ast(c);let d=ast::declaration(a);let stream=bodies::cleanup_source_at(ref p,1);let text=scanner::source_view(stream);
    assert(info.module_index==1 && info.argument_count==0 && info.receiver_kind==10 && info.effects==1 && info.abilities==3 && bodies::type_at(ref p,info.type).kind==3 && d.kind==6 && d.cleanup_body>0 && d.body==0 && d.field_count==0 && ast::attribute_count(a)==2);
    let edge=bodies::cleanup_edge_at(c,0);let helper=bodies::at(ref p,edge.target.instance);assert(bodies::cleanup_edge_count(c)==1 && edge.target.owner==1 && edge.target.source_index==ast::expression_origin(a,edge.expression) && bodies::signature(helper).module_index==1 && bodies::signature(helper).effects==1);
    var usize receivers=0;var usize i=1;while(i<=ast::expression_count(a)){let e=ast::expression_at(a,i);assert(e.start>=info.start && e.end<=info.end && e.token<scanner::token_count(stream));if(bytes::equal(text[e.start..e.end],b"this")){receivers+=1;}i+=1;}assert(receivers==1);
    i=0;var usize locals=0;while(i<ast::entry_count(a)){let e=ast::entry_at(a,i);if(e.type_root!=0){assert(bodies::type_at(ref p,bodies::cleanup_annotation_at(c,e.type_root)).scalar==1);locals+=1;}i+=1;}assert(locals==1);
}
fn void denied(ref u8[] text,u32 code){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){let p::Owner value=p::make();return 0;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==code && p.module_index==1 && p.end>p.start && p.end<=text.len);}};
}
@test fn void cleanup_generic_requests_enforce_shadowing_and_recursion(){
    denied(b"module provider;pub resource Owner{i32 n;cleanup{let i32 id=1;discard id<i32>(this.n);}}pub fn Owner make(){return Owner{.n=1};}fn T id<T>(T n){return n;}",4501);
    denied(b"module provider;pub resource Owner{i32 n;cleanup{discard id<i32>(this.n);}}pub fn Owner make(){return Owner{.n=1};}fn T id<T>(T n){return id<T>(n);}",4503);
}
@effects(filesystem)
@test fn void scalar_vec_cleanup_is_deduplicated_and_specializes_take_slot(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let source=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import std::vec as v;import std::vec as q;fn usize main(){let v::Vec<i32> first=v::empty<i32>();let q::Vec<i32> second=q::empty<i32>();return v::len<i32>(ref first);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let g=interfaces::specializations(ref imported);assert(generics::count(g)==2 && generics::cleanup_count(g)==1 && generics::cleanup_dependency_count(g)==1);
    let p=switch(bodies::build(ref sources,ref metadata,ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bodies::count(ref p)==2 && bodies::cleanup_count(ref p)==1 && bodies::source_count(ref p)==1);let c=bodies::cleanup_at(ref p,1);let info=bodies::cleanup_info(c);let arg=bodies::cleanup_argument_at(c,0);assert(info.argument_count==1 && bodies::type_at(ref p,info.type).first==arg && bodies::type_at(ref p,arg).scalar==1);
    let edge=bodies::cleanup_edge_at(c,0);assert(edge.target.instance==0 && edge.target.intrinsic==3 && edge.target.intrinsic_type==arg && edge.target.inline_declaration!=0 && edge.target.inline_unsafe);
}
