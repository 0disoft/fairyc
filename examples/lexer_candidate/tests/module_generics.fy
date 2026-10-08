module tests::module_generics;
import module_program as programs;
import scanner as scanner;
import syntax as syntax;
import mir_model as mir;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,text).is_ok);let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn mir::Program retained(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(b)).is_ok);return switch(programs::compile(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained(b"module a;",b"module b;");}};}
fn programs::Problem rejected(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(b)).is_ok);return switch(programs::compile(ref sources)){case .ok(_):{assert(false);programs::Problem{.phase=0,.code=0,.module_index=0,.start=0,.end=0}}case .err(p):{p}};}
@test fn void imported_specializations_retain_provider_identity(){
    let p=retained(b"module main;import provider as p;import provider as q;fn T id<T>(T x){return x;}fn i32 main(){assert(p::id<bool>(true));return p::id<i32>(q::id<i32>(id<i32>(7)));}",b"module provider;fn T helper<T>(T x){return x;}pub fn T id<T>(T x){return helper<T>(x);}");
    // Two provider id/helper specializations and one caller specialization.
    assert(mir::function_count(ref p)==6 && verify::check(ref p).code==0);
    var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0);
    let nominal=retained(b"module main;import provider as p;fn i32 main(){return p::pick<i32>(7,p::make());}",b"module provider;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}pub fn T pick<T>(T x,Token t){assert(t.n==7);return x;}");
    assert(verify::check(ref nominal).code==0);assert(bytes::resize_zeroed(ref mut out,0).is_ok);assert(c::render(ref nominal,ref mut out)==0);
    let private=rejected(b"module main;import provider as p;fn i32 main(){return p::id<i32>(7);}",b"module provider;fn T id<T>(T x){return x;}");assert(private.phase==3 && private.code==6 && private.module_index==0);
    let mismatch=rejected(b"module main;import provider as p;fn i32 main(){discard p::id<bool>(7);return 0;}",b"module provider;pub fn T id<T>(T x){return x;}");assert(mismatch.phase==5 && mismatch.code==1 && mismatch.module_index==0 && mismatch.end>mismatch.start);
    let aggregate=rejected(b"module main;fn i32 main(){return 0;}",b"module provider;pub struct Box<T>{T x;}");assert(aggregate.phase==2 && aggregate.code==4501);
}
