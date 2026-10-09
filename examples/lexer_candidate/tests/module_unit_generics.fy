module tests::module_unit_generics;
import module_unit_generics as generics;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import module_types as types;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn interfaces::Interfaces retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;import provider as q;import other as r;@effects(console)fn i32 main(){let i32 n=unsafe{p::choose<i32,bool>(7,true)};let i32 m=unsafe{q::choose<i32,bool>(8,false)};let bool b=unsafe{q::choose<bool,i32>(true,9)};discard r::choose<i32,bool>(0,true);discard pointer<i32>(null);var i64[2] a=[1;2];let ref i64[] span=ref a;discard view<i64>(span);discard pair<u8>(3);return n+m;}fn ptr mut T pointer<T>(ptr mut T p){return p;}fn ref T[] view<T>(ref T[] p){return p;}fn T[2] pair<T>(T x){return [x;2];}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;@must_use @effects(console)pub unsafe fn A choose<A,B>(A a,B b){return a;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module other;pub fn A choose<A,B>(A a,B b){return a;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return retained();}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return retained();}};
    return switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);retained()}};
}
@test fn void specialization_identity_signatures_and_calls_survive_original_owners(){
    let p=retained();let plan=interfaces::specializations(ref p);assert(generics::count(plan)==6 && generics::work(plan)<1048576);
    var usize providers=0;var usize other=0;var usize pointers=0;var usize views=0;var usize arrays=0;var usize i=1;
    while(i<=generics::count(plan)){let s=generics::at(plan,i);let result=interfaces::type_at(ref p,s.result);
        if(s.module_index==1){assert(s.must_use && s.unsafe_call && s.effects==1 && s.argument_count==2 && s.parameter_count==2);assert(s.result==generics::argument_at(plan,s.argument_start) && s.result==generics::parameter_at(plan,s.parameter_start));assert(generics::argument_at(plan,s.argument_start+1)==generics::parameter_at(plan,s.parameter_start+1));providers+=1;}
        else if(s.module_index==2){assert(!s.must_use && !s.unsafe_call && s.effects==0);other+=1;}
        else if(result.kind==12){assert(result.scalar==1048579 && s.result==generics::parameter_at(plan,s.parameter_start));pointers+=1;}
        else if(result.kind==15){assert(result.scalar==20 && s.result==generics::parameter_at(plan,s.parameter_start));views+=1;}
        else if(result.kind==16){assert(result.second==2 && interfaces::type_at(ref p,result.first).scalar==3 && generics::parameter_at(plan,s.parameter_start)==result.first);arrays+=1;}
        i+=1;
    }assert(providers==2 && other==1 && pointers==1 && views==1 && arrays==1);
    var usize calls=0;var usize repeated=0;i=1;while(i<=generics::call_count(plan)){let c=generics::call_at(plan,i);if(c.instance!=0){assert(!c.deferred && c.intrinsic==0);calls+=1;if(c.instance==1){repeated+=1;}}i+=1;}assert(calls==7 && repeated==2);
}
fn void join_problem(ref u8[] text,u32 expected){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module prefix;fn i32 zero(){return 0;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,2)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==expected && p.module_index==1 && p.end>p.start && p.end<=text.len);}};
}
@test fn void invalid_template_arguments_shadowing_and_contracts_fail_with_original_spans(){
    join_problem(b"module main;struct Node{i32 n;}fn T id<T>(T n){return n;}fn i32 main(){discard id<Node>(Node{.n=0});return 0;}",4502);
    join_problem(b"module main;fn T id<T>(T n){return n;}fn i32 main(){return id<i32,bool>(1,true);}",4501);
    join_problem(b"module main;fn i32 id(i32 n){return n;}fn i32 main(){return id<i32>(1);}",4501);
    join_problem(b"module main;fn T id<T,T>(T n){return n;}fn i32 main(){return id<i32,i32>(1);}",4501);
    join_problem(b"module main;fn T id<T>(T n){return n;}fn i32 main(){let i32 id=1;return id<i32>(1);}",4501);
    join_problem(b"module main;@must_use fn void empty<T>(){}fn i32 main(){empty<i32>();return 0;}",4201);
    join_problem(b"module main;fn T id<T>(T n){return n;}fn i32 main(){discard id<c::int>(1);return 0;}",4502);
}
@test fn void rejected_collection_rolls_back_all_graph_additions(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;struct Node{i32 n;}fn T id<T>(T n){return n;}fn i32 main(){discard id<i64>(1);discard id<Node>(Node{.n=0});return 0;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;while(i<=signatures::type_count(ref metadata)){assert(v::push<types::Type>(ref mut graph,signatures::type_at(ref metadata,i)).is_ok);i+=1;}let usize before=types::count(ref graph);
    switch(generics::build(ref sources,ref metadata,ref bound,ref mut graph)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4502 && types::count(ref graph)==before);}};
}
fn void suffix(ref mut bytes::Buffer text,usize n){let digits=b"0123456789";if(n>=10){assert(bytes::append_byte(text,digits[n/10]).is_ok);}assert(bytes::append_byte(text,digits[n%10]).is_ok);}
@test fn void retained_plan_rejects_changed_provider_bytes_and_graph_prefix(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::id<i32>(1);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn T id<T>(T n){return n;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;
    while(i<=signatures::type_count(ref metadata)){assert(v::push<types::Type>(ref mut graph,signatures::type_at(ref metadata,i)).is_ok);i+=1;}
    let plan=switch(generics::build(ref sources,ref metadata,ref bound,ref mut graph)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(generics::validate(ref plan,ref sources,ref metadata,ref graph).code==0);
    let types::Type last=*v::at<types::Type>(ref graph,types::count(ref graph)-1);discard v::pop<types::Type>(ref mut graph);
    assert(generics::validate(ref plan,ref sources,ref metadata,ref graph).code==12);
    var changed=last;changed.scalar+=1;assert(v::push<types::Type>(ref mut graph,changed).is_ok);
    assert(generics::validate(ref plan,ref sources,ref metadata,ref graph).code==12);
    discard v::pop<types::Type>(ref mut graph);assert(v::push<types::Type>(ref mut graph,last).is_ok);
    assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=6,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
    assert(generics::validate(ref plan,ref sources,ref metadata,ref graph).code==0);
    discard v::pop<syntax::Parsed>(ref mut sources);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn T id<T>(T n){return 0;}")).is_ok);
    let problem=generics::validate(ref plan,ref sources,ref metadata,ref graph);assert(problem.code==12 && problem.module_index==1);
    let refreshed=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    assert(signatures::source_matches(ref refreshed,1,v::at<syntax::Parsed>(ref sources,1)));
    let changed_body=generics::validate(ref plan,ref sources,ref refreshed,ref graph);assert(changed_body.code==12 && changed_body.module_index==1);
    discard v::pop<syntax::Parsed>(ref mut sources);
    assert(generics::validate(ref plan,ref sources,ref metadata,ref graph).code==12);
}
@test fn void total_specialization_limit_stays_at_sixty_four(){
    var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module main;").is_ok);var usize i=0;
    while(i<65){assert(bytes::append(ref mut text,b"fn T f").is_ok);suffix(ref mut text,i);assert(bytes::append(ref mut text,b"<T>(T n){return n;}").is_ok);i+=1;}
    assert(bytes::append(ref mut text,b"fn i32 main(){").is_ok);i=0;while(i<65){assert(bytes::append(ref mut text,b"discard f").is_ok);suffix(ref mut text,i);assert(bytes::append(ref mut text,b"<i32>(1);").is_ok);i+=1;}assert(bytes::append(ref mut text,b"return 0;}").is_ok);join_problem(bytes::view(ref text),4503);
}
@test fn void forwarded_private_templates_share_ordered_children_after_owner_removal(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;import provider as q;fn bool main(){assert(p::flip<i32,bool>(1,true));return q::flip<i32,bool>(2,false);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn B flip<A,B>(A a,B b){return choose<B,A>(b,a);}fn A choose<A,B>(A a,B b){return a;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let p=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let plan=interfaces::specializations(ref p);assert(generics::count(plan)==2 && generics::dependency_count(plan)==1);let parent=generics::at(plan,1);let child=generics::at(plan,2);let edge=generics::dependency_at(plan,0);assert(parent.module_index==1 && child.module_index==1 && parent.declaration!=child.declaration && edge.owner==1 && edge.instance==2 && edge.source_index!=0 && edge.intrinsic==0);assert(parent.result==child.result && generics::argument_at(plan,parent.argument_start)==generics::argument_at(plan,child.argument_start+1) && generics::argument_at(plan,parent.argument_start+1)==generics::argument_at(plan,child.argument_start));
}
@test fn void direct_indirect_and_changing_argument_recursion_fail_closed(){
    join_problem(b"module main;fn T id<T>(T n){return id<T>(n);}fn i32 main(){return id<i32>(1);}",4503);
    join_problem(b"module main;fn T first<T>(T n){return second<T>(n);}fn T second<T>(T n){return first<T>(n);}fn i32 main(){return first<i32>(1);}",4503);
    join_problem(b"module main;fn i64 id<T>(T n){return id<i64>(1);}fn i32 main(){discard id<i32>(1);return 0;}",4503);
}
@test fn void forwarded_depth_limit_applies_even_to_a_completed_leaf(){
    var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module main;").is_ok);var usize i=0;while(i<33){assert(bytes::append(ref mut text,b"fn T f").is_ok);suffix(ref mut text,i);assert(bytes::append(ref mut text,b"<T>(T n){return ").is_ok);if(i==32){assert(bytes::append(ref mut text,b"n;").is_ok);}else{assert(bytes::append(ref mut text,b"f").is_ok);suffix(ref mut text,i+1);assert(bytes::append(ref mut text,b"<T>(n);").is_ok);}assert(bytes::append(ref mut text,b"}").is_ok);i+=1;}assert(bytes::append(ref mut text,b"fn i32 main(){discard f32<i32>(1);return f0<i32>(1);}").is_ok);join_problem(bytes::view(ref text),4503);
}
@effects(filesystem)
@test fn void exact_sdk_templates_specialize_named_resource_elements(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return;}};let source=switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import std::vec as v;resource Token{i32 n;cleanup{}}fn usize main(){let v::Vec<Token> value=v::empty<Token>();return v::len<Token>(ref value);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(bytes::view(ref source))).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let p=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let plan=interfaces::specializations(ref p);assert(generics::count(plan)==2 && generics::cleanup_count(plan)==2 && generics::cleanup_dependency_count(plan)==1);let empty=generics::at(plan,1);let len=generics::at(plan,2);assert(empty.module_index==1 && empty.parameter_count==0 && empty.argument_count==1 && len.module_index==1 && len.parameter_count==1);let vector=interfaces::type_at(ref p,empty.result);assert(vector.kind==13 && interfaces::type_at(ref p,vector.first).kind==3 && interfaces::type_at(ref p,vector.first).module_index==0);assert(generics::argument_at(plan,empty.argument_start)==vector.first && generics::argument_at(plan,len.argument_start)==vector.first);let borrow=interfaces::type_at(ref p,generics::parameter_at(plan,len.parameter_start));assert(borrow.kind==7 && borrow.first==empty.result);
}
