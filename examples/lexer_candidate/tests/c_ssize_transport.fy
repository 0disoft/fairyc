module tests::c_ssize_transport;
import c_abi_types as abi;
import c_emit as emit;
import mir_model as m;
import mir_lower as mir;
import mir_verify as proof;
import hir_lower as hir;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import scalar_casts as casts;
import std::bytes as bytes;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] source){var bytes::Buffer input=bytes::empty();assert(bytes::append(ref mut input,source).is_ok);let stream=switch(scanner::collect(input,false,32768,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module main;");}};return switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);parsed(b"module main;")}};}
fn m::Program program(){let source=b"module main;@link_name(\"fairyc_fixture_make_c_ssize_t\")unsafe extern C fn c::ssize_t produce();@link_name(\"fairyc_fixture_consume_c_ssize_t\")unsafe extern C fn i32 consume(c::ssize_t n);fn i32 main(){return unsafe{consume(produce())};}";let h=switch(hir::lower(typing::check(names::resolve(parsed(source))),0)){case .ok(p):{p}case .err(_):{assert(false);return program();}};return switch(mir::lower(h)){case .ok(p):{p}case .err(_):{assert(false);program()}};}
fn usize position(ref u8[] text,ref u8[] part){var usize i=0;while(i<=text.len){if(part.len<=text.len-i && bytes::equal(text[i..i+part.len],part)){return i;}i+=1;}return text.len;}
fn bytes::Buffer generated(){let p=program();assert(proof::check(ref p).code==0);var bytes::Buffer text=bytes::empty();assert(emit::render(ref p,ref mut text)==0);return text;}
@test
fn void ssize_identity_survives_foreign_results_parameters_and_mir_locals(){
    assert(abi::valid(138) && abi::named(b"c::ssize_t")==138 && bytes::equal(abi::spelling(138),b"ssize_t"));
    assert(!casts::allowed(138,2) && !casts::allowed(2,138) && !casts::allowed(138,5));
    let p=program();assert(proof::check(ref p).code==0 && m::target_count(ref p)==3);
    assert(m::target_at(ref p,1).result==138 && m::target_at(ref p,2).signature_count==1 && m::signature_at(ref p,m::target_at(ref p,2).signature_start)==138);
    var usize i=1;var bool kept=false;while(i<=m::local_count(ref p)){if(m::local_at(ref p,i).type==138){kept=true;}i+=1;}assert(kept);
}
@test
fn void ssize_literals_and_operators_stay_outside_the_transport_lane(){
    let literal=typing::check(names::resolve(parsed(b"module main;unsafe extern C fn i32 consume(c::ssize_t n);fn i32 main(){return unsafe{consume(0)};}")));assert(typing::problem(ref literal).code!=0);
    let arithmetic=typing::check(names::resolve(parsed(b"module main;unsafe extern C fn c::ssize_t produce();fn i32 main(){let c::ssize_t n=unsafe{produce()};discard n+n;return 0;}")));assert(typing::problem(ref arithmetic).code!=0);
}
@test
@effects(console)
fn void generated_posix_source_requires_the_real_target_abi_before_headers(){
    let text=generated();let raw=bytes::view(ref text);let feature=position(raw,b"#define _POSIX_C_SOURCE 200809L");let header=position(raw,b"#include <stdint.h>");
    assert(feature<header && position(raw,b"#include <sys/types.h>")<header);
    assert(position(raw,b"#if defined(__linux__) && !defined(_WIN32)")<raw.len && position(raw,b"#error c.ssize_t is unavailable for candidate target")<raw.len);
    assert(position(raw,b"sizeof(ssize_t) == 8 && _Alignof(ssize_t) == 8")<raw.len && position(raw,b"((ssize_t)-1) < 0")<raw.len);
    assert(position(raw,b"extern ssize_t fairyc_fixture_make_c_ssize_t(void)")<raw.len && position(raw,b"fairyc_fixture_consume_c_ssize_t(ssize_t)")<raw.len);
    assert(io::write_bytes(raw).is_ok);
}
