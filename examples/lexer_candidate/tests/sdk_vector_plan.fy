module tests::sdk_vector_plan;
import sdk_vector_plan as plan;
import module_catalog as catalog;
import module_types as types;
import module_signatures as signatures;
import module_program as program;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::fs as fs;
import std::process as process;
import std::text as utf8;

fn syntax::Parsed parsed(bytes::Buffer input){
    let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(bytes::empty());}};
    let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;
}
fn syntax::Parsed text(ref u8[] source){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,source).is_ok);return parsed(b);}
@effects(filesystem)
fn bytes::Buffer sdk_source(){
    var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);
    let usize n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};
    let filename=switch(utf8::validate_utf8(bytes::view(ref path)[..n])){case .ok(s):{s}case .err(_):{assert(false);return bytes::empty();}};
    var fs::File file=switch(fs::open_read_utf8(filename)){case .ok(f):{f}case .err(_):{assert(false);return bytes::empty();}};
    var bytes::Buffer input=bytes::empty();assert(bytes::resize_zeroed(ref mut input,65537).is_ok);var usize used=0;
    while(true){let usize read=switch(fs::read(ref mut file,bytes::view_mut(ref mut input)[used..])){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};used+=read;assert(used<=65536);if(read==0){break;}}
    assert(fs::close(file).is_ok && bytes::resize_zeroed(ref mut input,used).is_ok);return input;
}
fn usize result_root(ref syntax::Parsed p,usize ordinal){var usize i=0;var usize count=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3){if(count==ordinal){return d.result_type;}count+=1;}i+=1;}assert(false);return 0;}
fn plan::Plan owned(ref u8[] sdk){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,text(b"module user;import std::vec as v;import std::vec as alias;import other as other;resource Token{i32 n;cleanup{}}fn v::Vec<i64> a(){}fn alias::Vec<i64> b(){}fn v::Vec<Token> c(){}fn v::Vec<other::Token> d(){}fn ref mut v::Vec<i64> e(){}fn v::Vec<v::Vec<i64>> f(){}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,text(sdk)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,text(b"module other;pub resource Token{i32 n;cleanup{}}")).is_ok);
    let c=switch(catalog::build(ref sources)){case .ok(c):{c}case .err(_):{assert(false);return owned(sdk);}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=0;
    while(i<6){assert(types::resolve(ref c,v::at<syntax::Parsed>(ref sources,0),0,result_root(v::at<syntax::Parsed>(ref sources,0),i),ref mut graph).problem.code==0);i+=1;}
    let g=switch(types::assemble(ref c,graph)){case .ok(g):{g}case .err(_):{assert(false);return owned(sdk);}};
    return switch(plan::build(c,g)){case .ok(p):{p}case .err(_):{assert(false);return owned(sdk);}};
}
@effects(filesystem)
@test fn void concrete_vec_plan_survives_parser_owners(){
    let source=sdk_source();let p=owned(bytes::view(ref source));assert(plan::count(ref p)==4 && plan::module_count(ref p)==3);
    let scalar=plan::at(ref p,0);let local=plan::at(ref p,1);let imported=plan::at(ref p,2);let nested=plan::at(ref p,3);
    assert(plan::type_at(ref p,scalar.element).scalar==2);
    assert(local.element!=imported.element && plan::type_at(ref p,local.element).module_index==0 && plan::type_at(ref p,imported.element).module_index==2);
    assert(plan::type_at(ref p,nested.element).kind==13 && nested.element==scalar.type);
    var usize i=0;while(i<plan::count(ref p)){let item=plan::at(ref p,i);assert(item.module_index==1 && plan::find(ref p,item.type)==i+1);i+=1;}
    let a=plan::declaration_lookup(ref p,0,b"v::Vec");let b=plan::declaration_lookup(ref p,0,b"alias::Vec");assert(a.code==0 && a.item==b.item && plan::declaration_at(ref p,a.item).declaration==scalar.declaration);
    let fresh=text(bytes::view(ref source));assert(plan::source_matches(ref p,1,ref fresh));
    let changed=text(b"module std::vec;pub resource Vec<T>{cleanup{}}");assert(!plan::source_matches(ref p,1,ref changed));
    assert(plan::field_count()==6 && plan::storage_size()==48 && plan::storage_alignment()==8);
    i=0;while(i<6){assert(plan::field_offset(i)==i*8 && plan::field_type(i)==if(i==0){1048591}else{5});i+=1;}
    assert(plan::field_type(6)==0 && plan::find(ref p,scalar.element)==0);
}
@effects(filesystem)
@test fn void concrete_vec_plan_rejects_missing_and_forged_instances(){
    let source=sdk_source();var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut sources,text(b"module user;import std::vec as v;fn v::Vec<i64> a(){}fn v::Vec<i32> b(){}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,text(bytes::view(ref source))).is_ok);
    let c=switch(catalog::build(ref sources)){case .ok(c):{c}case .err(_):{assert(false);return;}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();let input=v::at<syntax::Parsed>(ref sources,0);
    let a=types::resolve(ref c,input,0,result_root(input,0),ref mut graph);let b=types::resolve(ref c,input,0,result_root(input,1),ref mut graph);assert(a.problem.code==0 && b.problem.code==0);
    let g=switch(types::assemble(ref c,graph)){case .ok(g):{g}case .err(_):{assert(false);return;}};
    var v::Vec<plan::Instance> instances=v::empty<plan::Instance>();assert(plan::validate(ref c,ref g,ref instances)==9);
    let ta=types::graph_at(ref g,a.type);let tb=types::graph_at(ref g,b.type);
    let first=plan::Instance{.type=a.type,.element=ta.first,.module_index=ta.module_index,.declaration=ta.declaration};
    let second=plan::Instance{.type=b.type,.element=tb.first,.module_index=tb.module_index,.declaration=tb.declaration};
    assert(v::push<plan::Instance>(ref mut instances,first).is_ok && v::push<plan::Instance>(ref mut instances,second).is_ok && plan::validate(ref c,ref g,ref instances)==0);
    {let ref mut plan::Instance item=v::at_mut<plan::Instance>(ref mut instances,0);item.element=tb.first;}assert(plan::validate(ref c,ref g,ref instances)==9);
    {let ref mut plan::Instance item=v::at_mut<plan::Instance>(ref mut instances,0);*item=first;item.module_index=0;}assert(plan::validate(ref c,ref g,ref instances)==9);
    {let ref mut plan::Instance item=v::at_mut<plan::Instance>(ref mut instances,0);*item=second;}assert(plan::validate(ref c,ref g,ref instances)==9);
    {let ref mut plan::Instance item=v::at_mut<plan::Instance>(ref mut instances,0);*item=first;}
    assert(v::push<plan::Instance>(ref mut instances,second).is_ok && plan::validate(ref c,ref g,ref instances)==9);
    while(v::len<plan::Instance>(ref instances)<=plan::limit()){assert(v::push<plan::Instance>(ref mut instances,first).is_ok);}assert(plan::validate(ref c,ref g,ref instances)==4503);
    assert(switch(plan::assemble(c,g,instances)){case .ok(_):{false}case .err(code):{code==4503}});
}
@test fn void ordinary_module_signature_consumer_keeps_empty_vector_plan(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,text(b"module user;fn i32 main(){return 42;}")).is_ok);
    let p=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(signatures::vector_count(ref p)==0 && signatures::count(ref p)==1 && signatures::lookup(ref p,0,b"main").code==0);
    assert(signatures::type_at(ref p,signatures::at(ref p,1).result).scalar==1);
    assert(switch(program::compile(ref sources)){case .ok(_):{true}case .err(_):{false}});
}
