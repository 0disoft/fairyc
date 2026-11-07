module tests::native_mir_artifact;
import native_mir_artifact_encode as encoder;
import native_mir_artifact_decode as decoder;
import native_mir_artifact_current as current;
import candidate_artifact_wire as wire;
import mir_model as m;
import mir_lower as mir;
import mir_verify as proof;
import hir_lower as hir;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import c_emit as emit;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import std::process as process;

fn m::Program program(ref u8[] raw){
    let input=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return program(raw);}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return program(raw);}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return program(raw);}};assert(syntax::problem_count(ref parsed)==0);
    let resolved=names::resolve(parsed);assert(names::problem(ref resolved).code==0);
    let checked=typing::check(resolved);assert(typing::problem(ref checked).code==0);
    let h=switch(hir::lower(checked,0)){case .ok(p):{p}case .err(_):{assert(false);return program(raw);}};
    return switch(mir::lower(h)){case .ok(p):{p}case .err(_):{assert(false);program(raw)}};
}
fn wire::Payload packet(ref m::Program p,ref mut usize work){return switch(encoder::encode(p,work)){case .ok(p):{p}case .err(_):{assert(false);packet(p,work)}};}
fn m::Program restored(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(p):{p}case .err(_):{assert(false);restored(raw,work)}};}
fn u32 refused(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(_):{0}case .err(code):{code}};}

// Producer owns no synthetic AST or helper facts. Each packet begins at the
// original scanner/parser and is round-tripped through the complete verifier.
@effects(console) pub fn void member(ref u8[] raw){
    let p=program(raw);assert(proof::check(ref p).code==0);var usize work=0;
    let saved=packet(ref p,ref mut work);let view=wire::content(ref saved);let encode_cost=work;
    work=0;let decoded=restored(view,ref mut work);let decode_cost=work;
    assert(m::has_closures(ref p)==m::has_closures(ref decoded) && m::closure_count(ref p)==m::closure_count(ref decoded));
    assert(bytes::equal(m::source_view(ref p),m::source_view(ref decoded)) && bytes::equal(m::payload_view(ref p),m::payload_view(ref decoded)));
    work=0;let again=packet(ref decoded,ref mut work);assert(bytes::equal(view,wire::content(ref again)));
    var bytes::Buffer a=bytes::empty();var bytes::Buffer b=bytes::empty();assert(emit::render(ref p,ref mut a)==0 && emit::render(ref decoded,ref mut b)==0 && bytes::equal(bytes::view(ref a),bytes::view(ref b)));
    work=16777216-encode_cost;let exact=packet(ref p,ref mut work);assert(work==16777216 && bytes::equal(view,wire::content(ref exact)));
    work=16777216-encode_cost+1;assert(switch(encoder::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    work=16777216-decode_cost;let exact_restore=restored(view,ref mut work);assert(work==16777216 && m::function_count(ref exact_restore)==m::function_count(ref p));
    work=16777216-decode_cost+1;assert(refused(view,ref mut work)==1002 && work<=16777216);
    work=16777217;assert(refused(view,ref mut work)==1002 && work==16777217);
    assert(io::write_usize(view.len).is_ok && io::write("\n").is_ok && io::write_bytes(view).is_ok);
}

// A separate process receives only stored bytes; no producer in-memory model
// or native address survives this boundary. The source-bound owner is separate.
@effects(console) pub fn i32 consume(){
    var bytes::Buffer raw=bytes::empty();var u8[4096] chunk=[0;4096];
    while(true){let count=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 18;}};if(count==0){break;}if(bytes::len(ref raw)>wire::byte_limit()-count || !bytes::append(ref mut raw,ref chunk[..count]).is_ok){return 18;}}
    var usize work=0;let p=switch(if(process::arg_count()>2){var u8[4096] source=[0;4096];let size=switch(process::arg(2,ref mut source)){case .ok(n):{n}case .err(_):{return 18;}};current::restore(bytes::view(ref raw),ref source[..size],0,ref mut work)}else{decoder::decode(bytes::view(ref raw),ref mut work)}){case .ok(p):{p}case .err(_):{return 17;}};
    var bytes::Buffer out=bytes::empty();if(emit::render(ref p,ref mut out)!=0 || !io::write_bytes(bytes::view(ref out)).is_ok){return 19;}return 0;
}

@test fn void native_mir_artifact_requires_exact_family_and_canonical_feature(){
    let p=program(b"module main;fn i32 main(){return 42;}");var usize work=0;let saved=packet(ref p,ref mut work);let raw=wire::content(ref saved);assert(bytes::equal(raw[..8],b"FYNMIR00"));
    var bytes::Buffer bad=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return;}};bytes::view_mut(ref mut bad)[7]=49;work=0;assert(refused(bytes::view(ref bad),ref mut work)==1001);
    assert(bytes::append(ref mut bad,b"\x00\x00\x00\x00\x00\x00\x00\x00").is_ok);work=0;assert(refused(bytes::view(ref bad),ref mut work)==1001);
    bytes::view_mut(ref mut bad)[7]=50;work=0;assert(refused(bytes::view(ref bad),ref mut work)==1001);
    var usize i=0;while(i<raw.len){work=0;assert(refused(raw[..i],ref mut work)!=0);i=if(i<40){i+1}else{i+97};}
    bad=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::append(ref mut bad,b"\x00").is_ok);work=0;assert(refused(bytes::view(ref bad),ref mut work)==1001);
}
