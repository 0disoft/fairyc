module tests::module_unit_bind;
import module_unit_bind as binding;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){
    let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;
}
fn binding::Bound retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module prefix;")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import a as p;import a as q;import b as r;struct Token{i32 n;}fn i32 main(){let p::Token x=p::make();let q::Token y=q::make();let r::Token z=r::make();let Token own=Token{.n=0};return 7;}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module a;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module b;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=8};}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained();}};
    return switch(binding::bind(ref sources,ref metadata,1,2)){case .ok(p):{p}case .err(_):{assert(false);retained()}};
}
fn usize type_target(ref binding::Bound p,ref u8[] name){let ast=binding::parsed(p);var usize i=1;while(i<=syntax::type_count(ast)){if(syntax::type_at(ast,i).kind==1 && bytes::equal(syntax::type_text(ast,i),name)){return binding::type_target(p,i);}i+=1;}assert(false);return 0;}
fn usize value_target(ref binding::Bound p,ref u8[] name){let ast=binding::parsed(p);var usize i=1;while(i<=syntax::expression_count(ast)){if(syntax::expression_at(ast,i).kind==2 && bytes::equal(syntax::expression_text(ast,i),name)){return binding::value_target(p,i);}i+=1;}assert(false);return 0;}

@test fn void aliases_and_external_nominals_keep_owned_full_source_coordinates(){
    let p=retained();assert(binding::first(ref p)==1 && binding::stop(ref p)==2 && binding::work(ref p)<1048576);
    let usize a=type_target(ref p,b"p::Token");let usize alias=type_target(ref p,b"q::Token");let usize b=type_target(ref p,b"r::Token");assert(a!=0 && a==alias && a!=b);
    let first=binding::target_at(ref p,a);let second=binding::target_at(ref p,b);assert(first.kind==4 && second.kind==4 && first.declaration==2 && second.declaration==2 && first.module_index==2 && second.module_index==3);
    let usize f=value_target(ref p,b"p::make");assert(f==value_target(ref p,b"q::make") && f!=value_target(ref p,b"r::make"));let function=binding::target_at(ref p,f);assert(function.kind==3 && function.module_index==2 && function.declaration==3 && function.start<function.end);
    let local=binding::target_at(ref p,type_target(ref p,b"Token"));assert(local.module_index==1 && local.declaration==5 && binding::definition_at(ref p,5).kind==4 && binding::definition_at(ref p,5).module_index==1 && binding::definition_at(ref p,5).declaration==5);
    assert(syntax::module_count(binding::parsed(ref p))==1 && syntax::problem_count(binding::parsed(ref p))==0);
}

@test fn void private_targets_and_stale_providers_fail_with_original_locations(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module skipped;")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::secret();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;fn i32 secret(){return 7;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(binding::bind(ref sources,ref metadata,1,2)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==6 && p.module_index==1 && bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,1))[p.start..p.end],b"p::secret"));}};
    switch(binding::bind(ref sources,ref metadata,1,3)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==6 && p.module_index==1 && p.start<p.end);}};
    var v::Vec<syntax::Parsed> changed=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut changed,parsed(b"module skipped;")).is_ok);assert(v::push<syntax::Parsed>(ref mut changed,parsed(b"module main;import provider as p;fn i32 main(){return p::secret();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut changed,parsed(b"module provider;fn i32 secret(){return 8;}")).is_ok);
    switch(binding::bind(ref changed,ref metadata,1,2)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==2);}};
    switch(binding::bind(ref sources,ref metadata,2,1)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==9);}};
}

@test fn void generic_parameters_remain_unbound_and_private_types_are_rejected(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn U identity<U>(U value){return value;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(binding::target_count(ref bound)==0);
    var v::Vec<syntax::Parsed> private_sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut private_sources,parsed(b"module main;import provider as p;fn i32 main(){let p::Hidden x=p::Hidden{.n=7};return 7;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut private_sources,parsed(b"module provider;struct Hidden{i32 n;}")).is_ok);
    let private_metadata=switch(signatures::build(ref private_sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(binding::bind(ref private_sources,ref private_metadata,0,1)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==6 && p.module_index==0 && bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref private_sources,0))[p.start..p.end],b"p::Hidden"));}};
}
