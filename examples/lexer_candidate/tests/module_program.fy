module tests::module_program;
import module_program as programs;
import scanner as scanner;
import syntax as syntax;
import mir_verify as verify;
import c_emit as c;
import std::bytes as bytes;
import std::vec as v;
import module_ast_join as join;
import module_bind as binding;
import module_signatures as signatures;
fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn u32 rendered(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);let p=switch(programs::compile(ref files)){case .ok(p):{p}case .err(e):{return e.code;}};assert(verify::check(ref p).code==0);var bytes::Buffer out=bytes::empty();assert(c::render(ref p,ref mut out)==0 && bytes::len(ref out)>0);return 0;}
fn programs::Problem rejected(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(programs::compile(ref files)){case .ok(_):{assert(false);programs::Problem{.phase=0,.code=0,.module_index=0,.start=0,.end=0}}case .err(p):{p}};}
@test fn void nominal_module_consumers_keep_identity_and_privacy(){
    assert(rendered(b"module main;import provider as p;fn i32 main(){let p::Token t=p::make();return p::value(t);}",b"module provider;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}pub fn i32 value(Token t){return t.n;}")==0);
    let mismatch=rejected(b"module main;import provider as p;struct Token{i32 n;}fn i32 main(){let Token t=Token{.n=7};return p::value(t);}",b"module provider;pub struct Token{i32 n;}pub fn i32 value(Token t){return t.n;}");assert(mismatch.phase==5 && mismatch.code==1 && mismatch.module_index==0 && mismatch.end>mismatch.start);
    assert(rendered(b"module main;import provider as p;fn i32 helper(){return 9;}fn i32 main(){return p::value();}",b"module provider;fn i32 helper(){return 7;}pub fn i32 value(){return helper();}")==0);
    assert(rendered(b"module main;import provider as p;fn i32 main(){let p::Owner owner=p::make();return p::value(ref owner);}",b"module provider;pub resource Owner{i32 n;cleanup{}}pub fn Owner make(){return Owner{.n=7};}pub fn i32 value(ref Owner owner){return owner.n;}")==0);
    let private_field=rejected(b"module main;import provider as p;fn i32 main(){let p::Owner owner=p::make();return owner.n;}",b"module provider;pub resource Owner{i32 n;cleanup{}}pub fn Owner make(){return Owner{.n=7};}");assert(private_field.phase==5 && private_field.code==6201);
    let private_ctor=rejected(b"module main;import provider as p;fn i32 main(){let p::Owner owner=p::Owner{.n=7};return 0;}",b"module provider;pub resource Owner{i32 n;cleanup{}}");assert(private_ctor.phase==5 && private_ctor.code==6201);
    let private_fn=rejected(b"module main;import provider as p;fn i32 main(){return p::secret();}",b"module provider;fn i32 secret(){return 7;}");assert(private_fn.phase==3 && private_fn.code==6);
    assert(rendered(b"module main;import provider as p;import provider as q;fn i32 main(){let p::Token t=q::make();return p::value(t);}",b"module provider;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}pub fn i32 value(Token t){return t.n;}")==0);
    assert(rendered(b"module main;import provider as p;fn i32 main(){let p::Kind k=p::make();return p::value(k);}",b"module provider;pub enum Kind{Off;On;}pub fn Kind make(){return .On;}pub fn i32 value(Kind k){return switch(k){case .Off:{0}case .On:{7}};}")==0);
    let effects=rejected(b"module main;import provider as p;fn i32 main(){return p::value();}",b"module provider;@effects(console)pub fn i32 value(){return 7;}");assert(effects.phase==5 && effects.code==7102);
    let unused=rejected(b"module main;import provider as p;fn i32 main(){p::value();return 0;}",b"module provider;@must_use pub fn i32 value(){return 7;}");assert(unused.phase==5 && unused.code==4202);
    let unsafe_call=rejected(b"module main;import provider as p;fn i32 main(){return p::value();}",b"module provider;pub unsafe fn i32 value(){return 7;}");assert(unsafe_call.phase==5 && unsafe_call.code==5101);
    let bad_provider=rejected(b"module main;import provider as p;fn i32 main(){return p::value();}",b"module provider;pub fn i32 value(){return missing;}");assert(bad_provider.phase==4 && bad_provider.module_index==1 && bad_provider.end>bad_provider.start);
    var v::Vec<syntax::Parsed> originals=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut originals,parsed(b"module main;fn i32 main(){return 7;}")).is_ok);
    let metadata=switch(signatures::build(ref originals)){case .ok(m):{m}case .err(_):{assert(false);return;}};
    var v::Vec<syntax::Parsed> changed=v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut changed,parsed(b"module main;fn i32 main(){return 8;}")).is_ok);
    let combined=switch(join::join(ref changed)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(switch(binding::bind(ref changed,ref metadata,combined)){case .ok(_):{false}case .err(p):{p.code==12}});
}
