module tests::module_signatures;
import module_signatures as signatures;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn signatures::Program owned(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(signatures::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return owned(b"module a;",b"module b;");}};}
fn catalog::Problem rejected(ref u8[] a,ref u8[] b){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(a)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(b)).is_ok);return switch(signatures::build(ref files)){case .ok(_):{assert(false);catalog::Problem{.code=0,.module_index=0,.start=0,.end=0}}case .err(p):{p}};}
@test fn void owned_module_signatures_keep_nominal_and_contract_identity(){
    let ref u8[] a=b"module a;import b as lib;struct Token{i32 n;}fn Token own(Token n){}fn lib::Token forwarded(lib::Token n){}fn Option<lib::Handle> wrapped(ref lib::Token n){}";
    let ref u8[] b=b"module b;import a as peer;pub struct Token{i32 n;}pub resource Handle{i32 n;cleanup{}}pub enum State{Ready;}@must_use @effects(console)pub unsafe fn Token accept(Token n){}pub fn Option<Handle> wrap(ref Token n){}pub fn State state(State n){}fn i32 hidden(){}";
    // Source syntax and catalog leave scope before the returned owner's queries.
    let p=owned(a,b);assert(signatures::count(ref p)==7);
    let own=signatures::at(ref p,1);let forwarded=signatures::at(ref p,2);let imported=signatures::lookup(ref p,0,b"lib::accept");assert(imported.code==0);let accept=signatures::at(ref p,imported.item);
    assert(own.result!=accept.result && forwarded.result==accept.result && forwarded.parameter_count==1);
    assert(signatures::parameter_at(ref p,forwarded.parameter_start)==signatures::parameter_at(ref p,accept.parameter_start));
    assert(accept.effects==1 && accept.must_use && accept.unsafe_call && !accept.foreign && accept.public && accept.module_index==1);
    let wrap=signatures::at(ref p,signatures::lookup(ref p,0,b"lib::wrap").item);let caller=signatures::at(ref p,3);assert(caller.result==wrap.result && signatures::parameter_at(ref p,caller.parameter_start)==signatures::parameter_at(ref p,wrap.parameter_start));
    assert(signatures::type_at(ref p,signatures::type_at(ref p,wrap.result).first).kind==3);
    let state=signatures::at(ref p,signatures::lookup(ref p,0,b"lib::state").item);assert(signatures::type_at(ref p,state.result).kind==4);
    assert(signatures::lookup(ref p,0,b"lib::hidden").code==6 && signatures::lookup(ref p,0,b"lib::Token").code==9);
    let bad=rejected(b"module a;import b as lib;fn lib::Hidden leak(){}",b"module b;struct Hidden{i32 n;}");assert(bad.code==6 && bad.module_index==0 && bad.end>bad.start);
    assert(rejected(b"module a;",b"module b;@must_use pub fn void bad(){}").code==4201);
    assert(rejected(b"module a;",b"module b;@effects(unknown)pub fn i32 bad(){}").code==7101);
    assert(rejected(b"module a;",b"module b;pub fn i32 bad(void n){}").code==9);
    let ffi=owned(b"module a;import b as lib;unsafe fn ptr mut void forward(ptr mut void p){}",b"module b;@link_name(\"host_slot\")pub unsafe extern C fn c::int foreign(ptr mut ptr mut void p);");let usize host_id=signatures::lookup(ref ffi,0,b"lib::foreign").item;let host=signatures::at(ref ffi,host_id);assert(host.foreign && host.unsafe_call && signatures::type_at(ref ffi,host.result).scalar==132 && signatures::type_at(ref ffi,signatures::parameter_at(ref ffi,host.parameter_start)).scalar==1048601 && bytes::equal(signatures::linker_name(ref ffi,host_id),b"host_slot"));
    assert(rejected(b"module a;@link_name(\"host_alias\")unsafe extern C fn i32 first(i32 n);",b"module b;@link_name(\"host_alias\")unsafe extern C fn i64 second(i32 n);").code==5106);
    let aliases=owned(b"module a;@link_name(\"host_alias\")unsafe extern C fn i32 first(i32 n);",b"module b;@link_name(\"host_alias\")unsafe extern C fn i32 second(i32 n);");assert(signatures::at(ref aliases,1).result==signatures::at(ref aliases,2).result);
}
