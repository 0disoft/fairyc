module tests::module_unit_bodies;
import module_unit_bodies as bodies;
import module_unit_body_ast as ast;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn bodies::Plan retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;import provider as q;fn i32 main(){discard p::flip<i32,bool>(1,true);discard q::flip<i32,bool>(2,false);return q::flip<bool,i32>(false,3);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;fn i32 ignored(){return 42;}@must_use @effects(console)pub unsafe fn B flip<A,B>(A a,B b){let ref str label=\"hello\\n\";let A[2] pair=[a;2];if(true){discard label;}return choose<B,A>(b,a);}fn A choose<A,B>(A a,B b){return a;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return retained();}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return retained();}};
    let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return retained();}};
    return switch(bodies::build(ref sources,ref metadata,ref imported)){case .ok(p):{p}case .err(_):{assert(false);retained()}};
}
@test fn void owned_instances_keep_ordered_types_literals_and_private_edges(){
    let p=retained();assert(bodies::count(ref p)==4 && bodies::source_count(ref p)==1 && bodies::work(ref p)<1048576);var usize parents=0;var usize children=0;var usize labels=0;var usize arrays=0;var usize id=1;
    while(id<=bodies::count(ref p)){let b=bodies::at(ref p,id);let s=bodies::signature(b);let a=bodies::body_ast(b);let d=ast::declaration(a);let stream=bodies::source_at(ref p,id);let text=scanner::source_view(stream);
        assert(s.module_index==1 && d.generic_parameters==0 && d.body>0 && d.body<=ast::statement_count(a) && d.start<d.end && d.end<=text.len);
        assert(bodies::annotation_at(b,d.result_type)==s.result && bodies::argument_at(b,0)==bodies::parameter_at(b,0));
        let bool parent=bytes::equal(text[d.name_start..d.name_end],b"flip");assert(parent || bytes::equal(text[d.name_start..d.name_end],b"choose"));
        if(parent){parents+=1;assert(s.must_use && s.unsafe_call && s.effects==1 && ast::attribute_count(a)==2 && bodies::edge_count(b)==1);let edge=bodies::edge_at(b,0);let child=bodies::at(ref p,edge.target.instance);assert(edge.target.owner==id && ast::expression_origin(a,edge.expression)==edge.target.source_index && ast::expression_at(a,edge.expression).kind==20);assert(bodies::argument_at(b,0)==bodies::argument_at(child,1) && bodies::argument_at(b,1)==bodies::argument_at(child,0));}
        else{children+=1;assert(!s.must_use && !s.unsafe_call && s.effects==0 && bodies::edge_count(b)==0);}
        var usize e=0;while(e<ast::entry_count(a)){let entry=ast::entry_at(a,e);assert(entry.id==e+1 && (entry.type_root==0 || entry.type_root<=ast::type_count(a)) && (entry.expression==0 || entry.expression<=ast::expression_count(a)));if(entry.type_root!=0){let t=bodies::type_at(ref p,bodies::annotation_at(b,entry.type_root));if(t.kind==16){assert(t.second==2 && t.first==bodies::argument_at(b,0));arrays+=1;}}e+=1;}
        e=1;while(e<=ast::expression_count(a)){let n=ast::expression_at(a,e);assert(n.start>=d.start && n.end<=d.end && n.token<scanner::token_count(stream));if(bytes::equal(scanner::token_cooked(stream,n.token),b"68656c6c6f0a")){labels+=1;}let usize left_bound=if(n.kind==12 || n.kind==14 || n.kind==31){ast::statement_count(a)}else{ast::expression_count(a)};let usize right_bound=if(n.kind==18){ast::statement_count(a)}else if(n.kind==20 || n.kind==23){ast::type_count(a)}else{ast::expression_count(a)};assert(n.left<=left_bound && n.right<=right_bound);e+=1;}
        id+=1;
    }assert(parents==2 && children==2 && labels>=2 && arrays==2 && bodies::usage(ref p).declarations==4);
}
@test fn void stale_provider_bytes_fail_before_cloning(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::id<i32>(1);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn T id<T>(T n){return n;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    discard v::pop<syntax::Parsed>(ref mut sources);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn T id<T>(T n){return 0;}")).is_ok);let refreshed=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    switch(bodies::build(ref sources,ref refreshed,ref imported)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==1);}};
}
@effects(filesystem)
@test fn void authenticated_sdk_bodies_own_storage_intrinsic_and_resource_annotations(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let source=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import std::vec as v;resource Token{i32 n;cleanup{}}fn usize main(){let v::Vec<Token> value=v::empty<Token>();return v::len<Token>(ref value);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(bodies::build(ref sources,ref metadata,ref imported)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(bodies::count(ref p)==2 && bodies::cleanup_count(ref p)==2 && bodies::source_count(ref p)==2);let empty=bodies::at(ref p,1);let arg=bodies::argument_at(empty,0);assert(bodies::type_at(ref p,arg).kind==3 && bodies::type_at(ref p,arg).module_index==0);assert(bodies::edge_count(empty)==3);var usize aggregates=0;var usize intrinsics=0;var usize i=0;while(i<bodies::edge_count(empty)){let edge=bodies::edge_at(empty,i);if(edge.target.aggregate_type!=0){assert(bodies::type_at(ref p,edge.target.aggregate_type).first==arg);aggregates+=1;}if(edge.target.intrinsic!=0){assert(edge.target.intrinsic_type==arg);intrinsics+=1;}i+=1;}assert(aggregates==1 && intrinsics==2);
    i=1;var usize vectors=0;var usize locals=0;while(i<=bodies::cleanup_count(ref p)){let c=bodies::cleanup_at(ref p,i);let info=bodies::cleanup_info(c);let a=bodies::cleanup_ast(c);let d=ast::declaration(a);let stream=bodies::cleanup_source_at(ref p,i);assert(info.receiver_kind==10 && info.abilities==1 && info.effects==0 && d.cleanup_body>0 && d.cleanup_body<=ast::statement_count(a) && d.body==0 && d.field_count==0);
        if(bodies::type_at(ref p,info.type).kind==13){vectors+=1;assert(info.argument_count==1 && bodies::cleanup_argument_at(c,0)==arg && bodies::cleanup_edge_count(c)==1);let edge=bodies::cleanup_edge_at(c,0);assert(edge.target.owner==i && ast::expression_origin(a,edge.expression)==edge.target.source_index && edge.target.instance==0 && edge.target.intrinsic==3 && edge.target.intrinsic_type==arg && edge.target.inline_declaration!=0 && edge.target.inline_unsafe);var usize e=0;while(e<ast::entry_count(a)){let entry=ast::entry_at(a,e);if(bytes::equal(scanner::source_view(stream)[entry.name_start..entry.name_end],b"value")){assert(bodies::cleanup_annotation_at(c,entry.type_root)==arg);locals+=1;}e+=1;}}
        else{assert(info.type==arg && info.argument_count==0 && bodies::cleanup_edge_count(c)==0);}i+=1;
    }assert(vectors==1 && locals==1);
}
@test fn void invalid_provider_local_annotation_preserves_original_span(){
    let text=b"module provider;pub fn T id<T>(T n){let Unknown x=n;return n;}";var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::id<i32>(1);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==5 && p.module_index==1 && bytes::equal(text[p.start..p.end],b"Unknown"));}};
}
