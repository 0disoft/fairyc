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

@test fn void ordered_type_batches_preserve_identity_and_rollback_on_later_errors(){
    let text=b"module a;import b as lib;fn i32 scalar(){}fn Option<lib::Token> wrapped(){}fn Result<u8,lib::Hidden> invalid(){}";
    let c=build(text,b"module b;pub struct Token{i32 n;}struct Hidden{i32 n;}");let p=parsed(text);
    var v::Vec<usize> roots=v::empty<usize>();assert(v::push<usize>(ref mut roots,result_root(ref p,0)).is_ok && v::push<usize>(ref mut roots,0).is_ok && v::push<usize>(ref mut roots,result_root(ref p,1)).is_ok);
    var v::Vec<types::Type> graph=v::empty<types::Type>();let ids=switch(types::resolve_many(ref c,ref p,0,ref roots,ref mut graph)){case .ok(ids):{ids}case .err(_):{assert(false);return;}};
    assert(v::len<usize>(ref ids)==3 && *v::at<usize>(ref ids,1)==0 && types::count(ref graph)==3);
    var v::Vec<types::Type> single=v::empty<types::Type>();var usize i=0;while(i<3){let root=*v::at<usize>(ref roots,i);if(root!=0){let r=types::resolve(ref c,ref p,0,root,ref mut single);assert(r.problem.code==0 && r.type==*v::at<usize>(ref ids,i));}i+=1;}assert(types::count(ref single)==types::count(ref graph));
    let changed=parsed(b"module a;import b as lib;fn i64 scalar(){}fn Option<lib::Token> wrapped(){}fn Result<u8,lib::Hidden> invalid(){}");
    assert(syntax::source_view(ref changed).len==syntax::source_view(ref p).len);
    switch(types::resolve_many(ref c,ref changed,0,ref roots,ref mut graph)){case .ok(_):{assert(false);}case .err(e):{assert(e.problem.code==12 && e.request==0 && types::count(ref graph)==3);}};
    discard v::pop<usize>(ref mut roots);assert(v::push<usize>(ref mut roots,result_root(ref p,2)).is_ok);
    switch(types::resolve_many(ref c,ref p,0,ref roots,ref mut graph)){case .ok(_):{assert(false);}case .err(e):{assert(e.problem.code==6 && e.request==3 && e.problem.end>e.problem.start && types::count(ref graph)==3);}};
    var v::Vec<types::Type> fresh=v::empty<types::Type>();switch(types::resolve_many(ref c,ref p,0,ref roots,ref mut fresh)){case .ok(_):{assert(false);}case .err(e):{assert(e.request==3 && types::count(ref fresh)==0);}};
}

@test fn void specialized_type_batches_keep_substitutions_and_reject_invalid_roots(){
    let text=b"module a;fn Option<T> wrap<T>(T value){return .some(value);}fn i32 scalar(){}";let c=build(text,b"module b;");let p=parsed(text);
    var v::Vec<types::Type> graph=v::empty<types::Type>();let scalar=types::resolve(ref c,ref p,0,result_root(ref p,1),ref mut graph);assert(scalar.problem.code==0);
    var v::Vec<usize> args=v::empty<usize>();assert(v::push<usize>(ref mut args,scalar.type).is_ok);
    var v::Vec<usize> roots=v::empty<usize>();assert(v::push<usize>(ref mut roots,result_root(ref p,0)).is_ok && v::push<usize>(ref mut roots,0).is_ok);
    let ids=switch(types::specialize_many(ref c,ref p,0,2,ref roots,ref args,ref mut graph)){case .ok(ids):{ids}case .err(_):{assert(false);return;}};
    let option=*v::at<usize>(ref ids,0);assert(*v::at<usize>(ref ids,1)==0 && types::at(ref graph,option).kind==5 && types::at(ref graph,option).first==scalar.type);
    let single=types::specialize(ref c,ref p,0,2,result_root(ref p,0),ref args,ref mut graph);assert(single.problem.code==0 && single.type==option && types::count(ref graph)==2);
    assert(v::push<usize>(ref mut roots,result_root(ref p,1)).is_ok);
    var v::Vec<types::Type> fresh=v::empty<types::Type>();assert(types::resolve(ref c,ref p,0,result_root(ref p,1),ref mut fresh).type==scalar.type);
    switch(types::specialize_many(ref c,ref p,0,2,ref roots,ref args,ref mut fresh)){case .ok(_):{assert(false);}case .err(e):{assert(e.problem.code==4501 && e.request==3 && types::count(ref fresh)==1);}};
    let changed=parsed(b"module a;fn Option<T> wrap<T>(T value){return .none;}fn i32 scalar(){}");switch(types::specialize_many(ref c,ref changed,0,2,ref roots,ref args,ref mut graph)){case .ok(_):{assert(false);}case .err(e):{assert(e.problem.code==12 && types::count(ref graph)==2);}};
}

@test fn void catalog_lookup_stays_inside_ordered_module_tables(){
    var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var usize module_index=0;
    while(module_index<22){var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module user").is_ok && bytes::append_usize(ref mut source,module_index).is_ok && bytes::append(ref mut source,b";import user0 as p;import user21 as q;pub fn void value(){}fn void hidden(){}").is_ok);
        var usize n=0;while(n<60){assert(bytes::append(ref mut source,b"fn void filler").is_ok && bytes::append_usize(ref mut source,n).is_ok && bytes::append(ref mut source,b"(){}").is_ok);n+=1;}
        assert(v::push<syntax::Parsed>(ref mut files,parsed(bytes::view(ref source))).is_ok);module_index+=1;
    }
    let c=switch(catalog::build(ref files)){case .ok(c):{c}case .err(_):{assert(false);return;}};assert(catalog::declaration_count(ref c)==1364);
    var usize probes=0;let own=catalog::lookup_measured(ref c,21,b"value",ref mut probes);assert(own.code==0 && probes<30 && catalog::declaration_at(ref c,own.item).module_index==21);
    probes=0;let alias=catalog::lookup_measured(ref c,0,b"q /* trivia */ :: value",ref mut probes);assert(alias.code==0 && alias.item==own.item && probes<60 && catalog::declaration_at(ref c,alias.item).declaration==4);
    probes=0;let private=catalog::lookup_measured(ref c,0,b"q::hidden",ref mut probes);assert(private.code==6 && probes<60 && !catalog::declaration_at(ref c,private.item).public);
    probes=0;assert(catalog::lookup_measured(ref c,0,b"q::absent",ref mut probes).code==5 && probes<120);
    assert(catalog::lookup(ref c,0,b"missing::value").code==4 && catalog::lookup(ref c,21,b"hidden").code==0);
    let first=catalog::lookup(ref c,21,b"p::value");assert(first.code==0 && first.item!=own.item && catalog::declaration_at(ref c,first.item).module_index==0);
}
