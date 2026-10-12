module tests::module_unit_call_contracts;
import module_unit_call_contracts as contracts;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] source){let data=switch(bytes::copy(source)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};let stream=switch(scanner::collect(data,false,32768,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn void denied(ref u8[] caller,ref u8[] provider,ref u8[] expected){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(caller)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;let p=contracts::check(ref sources,ref metadata,0,3,false,ref mut work);assert(p.code==7102 && p.module_index==0 && bytes::equal(caller[p.start..p.end],expected));}
@test
fn void effect_rejection_keeps_qualified_callee_instead_of_argument_list(){
    denied(b"module main;import lib as dep;fn i32 main(){return dep::load(1);}",b"module lib;@effects(filesystem)pub fn i32 load(i32 n){return n;}",b"dep::load");
}
@test
fn void generic_effect_rejection_keeps_the_original_generic_callee_span(){
    denied(b"module main;import lib as dep;fn i32 main(){return dep::load<i32>(1);}",b"module lib;@effects(filesystem)pub fn T load<T>(T n){return n;}",b"dep::load<i32>");
}
