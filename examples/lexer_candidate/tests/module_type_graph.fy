module tests::module_type_graph;
import module_catalog as catalog;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn catalog::Catalog build(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(catalog::build(ref files)){case .ok(c):{c}case .err(_):{assert(false);return build(b"module a;",b"module b;");}};}
fn u32 rejection(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(catalog::build(ref files)){case .ok(_):{0}case .err(p):{p.code}};}
fn usize result_root(ref syntax::Parsed p,usize ordinal){var usize i=0;var usize functions=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3){if(functions==ordinal){return d.result_type;}functions+=1;}i+=1;}assert(false);return 0;}
fn void catalog_retains_exact_identity_visibility_and_source(){
    let ref u8[] a=b"module a;import b as first;import b as second;pub struct Token{i32 n;}resource Private{i32 n;cleanup{}}enum Local{Ready;}";
    let ref u8[] b=b"module b;pub struct Token{i32 n;}pub resource Handle{i32 n;cleanup{}}pub enum State{Ready;}struct Hidden{i32 n;}";
    let c=build(a,b);assert(catalog::module_count(ref c)==2 && catalog::declaration_count(ref c)==7);
    let own=catalog::lookup(ref c,0,b"Token");let imported=catalog::lookup(ref c,0,b"first /* trivia */ :: Token");let alias=catalog::lookup(ref c,0,b"second::Token");
    assert(own.code==0 && imported.code==0 && imported.item==alias.item);
    assert(!catalog::same_declaration(catalog::declaration_at(ref c,own.item),catalog::declaration_at(ref c,imported.item)));
    assert(catalog::lookup(ref c,0,b"first::Handle").code==0 && catalog::lookup(ref c,0,b"first::State").code==0);
    assert(catalog::lookup(ref c,0,b"first::Hidden").code==6 && catalog::lookup(ref c,0,b"missing::Token").code==4 && catalog::lookup(ref c,0,b"first::missing").code==5);
    let pa=parsed(a);let pb=parsed(b);assert(catalog::source_matches(ref c,0,ref pa) && !catalog::source_matches(ref c,0,ref pb));
}
fn void canonical_types_do_not_merge_equal_file_local_ids(){
    let ref u8[] a=b"module a;import b as lib;pub struct Token{i32 n;}fn Token local(){}fn lib::Token imported(){}fn Option<lib::Token> option(){}fn ref lib::Token shared(){}fn ref mut lib::Token mutable(){}";
    let ref u8[] b=b"module b;import a as peer;pub struct Token{i32 n;}fn Token local(){}fn Option<Token> option(){}fn ref Token shared(){}";
    let c=build(a,b);let p=parsed(a);let q=parsed(b);var v::Vec<types::Type> graph=v::empty<types::Type>();
    let local=types::resolve(ref c,ref p,0,result_root(ref p,0),ref mut graph);let imported=types::resolve(ref c,ref p,0,result_root(ref p,1),ref mut graph);let provider=types::resolve(ref c,ref q,1,result_root(ref q,0),ref mut graph);
    assert(local.problem.code==0 && imported.problem.code==0 && provider.problem.code==0 && local.type!=imported.type && imported.type==provider.type);
    assert(types::at(ref graph,local.type).module_index==0 && types::at(ref graph,imported.type).module_index==1);
    let option=types::resolve(ref c,ref p,0,result_root(ref p,2),ref mut graph);let provider_option=types::resolve(ref c,ref q,1,result_root(ref q,1),ref mut graph);assert(option.problem.code==0 && option.type==provider_option.type && types::at(ref graph,option.type).first==provider.type);
    let shared=types::resolve(ref c,ref p,0,result_root(ref p,3),ref mut graph);let provider_shared=types::resolve(ref c,ref q,1,result_root(ref q,2),ref mut graph);let mutable=types::resolve(ref c,ref p,0,result_root(ref p,4),ref mut graph);assert(shared.problem.code==0 && mutable.problem.code==0 && shared.type==provider_shared.type && shared.type!=mutable.type);
    let owned=switch(types::assemble(ref c,graph)){case .ok(g):{g}case .err(_):{assert(false);return;}};assert(types::graph_count(ref owned)==5 && types::graph_at(ref owned,imported.type).module_index==1);
}
fn void graph_publication_rejects_forged_nominal_or_child_identity(){
    let c=build(b"module a;pub struct Token{i32 n;}",b"module b;");var v::Vec<types::Type> graph=v::empty<types::Type>();
    assert(v::push<types::Type>(ref mut graph,types::Type{.kind=2,.scalar=0,.module_index=8,.declaration=2,.first=0,.second=0}).is_ok);assert(types::validate(ref c,ref graph)==9);
    discard v::pop<types::Type>(ref mut graph);assert(v::push<types::Type>(ref mut graph,types::Type{.kind=5,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=0}).is_ok);assert(types::validate(ref c,ref graph)==9);
    assert(!types::assemble(ref c,graph).is_ok);
}
fn void graph_errors_leave_no_partial_interned_types(){
    let ref u8[] a=b"module a;import b as lib;fn Result<i32,lib::Hidden> invalid(){}fn Vec<i32> vec(){}fn ptr ptr i32 pointer(){}";let ref u8[] b=b"module b;struct Hidden{i32 n;}";
    let c=build(a,b);let p=parsed(a);var v::Vec<types::Type> graph=v::empty<types::Type>();let rejected=types::resolve(ref c,ref p,0,result_root(ref p,0),ref mut graph);assert(rejected.problem.code==6 && rejected.problem.end>rejected.problem.start && types::count(ref graph)==0);
    assert(types::resolve(ref c,ref p,0,result_root(ref p,1),ref mut graph).problem.code==9 && types::count(ref graph)==0);
    assert(types::resolve(ref c,ref p,0,result_root(ref p,2),ref mut graph).problem.code==9 && types::count(ref graph)==0);
    let changed=parsed(b"module a;fn i32 replacement(){}");assert(types::resolve(ref c,ref changed,0,result_root(ref changed,0),ref mut graph).problem.code==12 && types::count(ref graph)==0);
}
fn void catalog_rejects_duplicate_and_unresolved_coordinates(){
    assert(rejection(b"module a;",b"module a;")==1);
    assert(rejection(b"module a;import missing as lib;",b"module b;")==3);
    assert(rejection(b"module a;import b as lib;import b as lib;",b"module b;")==7);
    assert(rejection(b"module a;import b as Token;struct Token{i32 n;}",b"module b;")==2);
    assert(rejection(b"module a;struct Token{i32 n;}resource Token{cleanup{}}",b"module b;")==2);
}
@test fn void module_type_graph_native_contract(){
    catalog_retains_exact_identity_visibility_and_source();
    canonical_types_do_not_merge_equal_file_local_ids();
    graph_errors_leave_no_partial_interned_types();
    catalog_rejects_duplicate_and_unresolved_coordinates();
    graph_publication_rejects_forged_nominal_or_child_identity();
}
