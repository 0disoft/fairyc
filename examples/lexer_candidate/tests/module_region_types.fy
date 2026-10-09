module tests::module_region_types;
import module_catalog as catalog;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import std::io as io;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn catalog::Catalog build(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(catalog::build(ref files)){case .ok(c):{c}case .err(_):{assert(false);return build(b"module a;",b"module b;");}};}
fn u32 rejection(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(catalog::build(ref files)){case .ok(_):{0}case .err(p):{p.code}};}
fn usize result_root(ref syntax::Parsed p,usize ordinal){var usize i=0;var usize functions=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3){if(functions==ordinal){return d.result_type;}functions+=1;}i+=1;}assert(false);return 0;}

@test @effects(console) fn void canonical_region_type_rejects_forged_representation_and_wrappers(){
    let a=b"module a;fn Region f(){}";let b=b"module b;fn Region g(){}";let c=build(a,b);let p=parsed(a);let q=parsed(b);var v::Vec<types::Type> graph=v::empty<types::Type>();
    let x=types::resolve(ref c,ref p,0,result_root(ref p,0),ref mut graph);let y=types::resolve(ref c,ref q,1,result_root(ref q,0),ref mut graph);discard io::write_u32(x.problem.code);discard io::write(" ");discard io::write_u32(y.problem.code);discard io::write(" ");discard io::write_usize(x.type);discard io::write(" ");discard io::write_usize(y.type);discard io::write(" ");discard io::write_usize(types::count(ref graph));discard io::write("\n");assert(x.problem.code==0 && y.problem.code==0 && x.type==y.type && types::count(ref graph)==1);discard io::write_u32(types::at(ref graph,x.type).kind);discard io::write(" ");discard io::write_u32(types::validate(ref c,ref graph));discard io::write("\n");assert(types::at(ref graph,x.type).kind==19 && types::validate(ref c,ref graph)==0);
    let expected=types::at(ref graph,1);var changed=expected;changed.scalar=1;{let slot=v::at_mut<types::Type>(ref mut graph,0);*slot=changed;}discard io::write_u32(types::validate(ref c,ref graph));discard io::write("\n");assert(types::validate(ref c,ref graph)==9);changed=expected;changed.module_index=1;{let slot=v::at_mut<types::Type>(ref mut graph,0);*slot=changed;}discard io::write_u32(types::validate(ref c,ref graph));discard io::write("\n");assert(types::validate(ref c,ref graph)==9);changed=expected;changed.first=1;{let slot=v::at_mut<types::Type>(ref mut graph,0);*slot=changed;}discard io::write_u32(types::validate(ref c,ref graph));discard io::write("\n");assert(types::validate(ref c,ref graph)==9);{let slot=v::at_mut<types::Type>(ref mut graph,0);*slot=expected;}
    assert(v::push<types::Type>(ref mut graph,types::Type{.kind=5,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=0}).is_ok);discard io::write_u32(types::validate(ref c,ref graph));discard io::write("\n");assert(types::validate(ref c,ref graph)==9);discard v::pop<types::Type>(ref mut graph);
    assert(v::push<types::Type>(ref mut graph,types::Type{.kind=7,.scalar=0,.module_index=0,.declaration=0,.first=1,.second=0}).is_ok);discard io::write_u32(types::validate(ref c,ref graph));discard io::write("\n");assert(types::validate(ref c,ref graph)==9);
}
