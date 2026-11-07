module tests::native_mir_artifact_current;
import native_mir_artifact_current as current;
import native_mir_artifact_encode as encoder;
import native_mir_artifact_decode as decoder;
import candidate_artifact_wire as wire;
import candidate_limits as limits;
import mir_model as m;
import mir_verify as proof;
import mir_lower as mir;
import hir_lower as hir;
import hir_shapes as shapes;
import name_resolution as names;
import type_check as typing;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;

fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
fn m::Program program(ref u8[] raw,usize module_index){
    let stream=switch(scanner::collect(copied(raw),false,limits::program_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return program(raw,module_index);}};
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return program(raw,module_index);}};
    assert(syntax::problem_count(ref parsed)==0);
    let resolved=names::resolve(parsed);assert(names::problem(ref resolved).code==0);
    let typed=typing::check(resolved);assert(typing::problem(ref typed).code==0);
    let high=switch(hir::lower(typed,module_index)){case .ok(p):{p}case .err(_):{assert(false);return program(raw,module_index);}};
    return switch(mir::lower(high)){case .ok(p):{p}case .err(_):{assert(false);program(raw,module_index)}};
}
fn wire::Payload encoded(ref m::Program p){var usize work=0;return switch(encoder::encode(p,ref mut work)){case .ok(data):{data}case .err(_):{assert(false);return encoded(p);}};}
fn m::Program restored(ref u8[] raw,ref u8[] source,usize module_index,ref mut usize work){return switch(current::restore(raw,source,module_index,work)){case .ok(p):{p}case .err(_):{assert(false);return restored(raw,source,module_index,work);}};}
fn u32 rejected(ref u8[] raw,ref u8[] source,usize module_index,ref mut usize work){return switch(current::restore(raw,source,module_index,work)){case .ok(_):{0}case .err(code):{code}};}
fn bool detached(ref u8[] raw){var usize work=0;return switch(decoder::decode(raw,ref mut work)){case .ok(p):{proof::check(ref p).code==0}case .err(_):{false}};}

// Both producer and payload are closed before the caller inspects this owner.
fn m::Program retained(ref u8[] source,usize module_index){let original=program(source,module_index);let data=encoded(ref original);var usize work=0;return restored(wire::content(ref data),source,module_index,ref mut work);}
fn void roundtrip(ref u8[] source,usize module_index,usize closures){
    let original=program(source,module_index);let data=encoded(ref original);let marker=if(closures==0){b"FYNMIR00"}else{b"FYNMIR01"};
    assert(bytes::equal(wire::content(ref data)[..8],marker));
    let trusted=retained(source,module_index);assert(proof::check(ref trusted).code==0 && m::closure_count(ref trusted)==closures && m::module_index(ref trusted)==module_index);
    assert(bytes::equal(m::source_view(ref trusted),source));let again=encoded(ref trusted);assert(bytes::equal(wire::content(ref data),wire::content(ref again)));
}
@test fn void source_bound_native_mir_restores_both_exact_families(){
    roundtrip(b"module main;fn i32 main(){return 42;}",0,0);
    roundtrip(b"module main;fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}fn i32 main(){let fn i32(i32) action=make(40);return action(2);}",17,1);
    roundtrip(b"module main;fn i32 main(){let fn i32(i32) action=fn [] i32(i32 x){return x+1;};return action(41);}",limits::source_files()-1,1);
    roundtrip(b"module main;struct Box{fn i32(i32) action;}fn fn i32(i32) make(i32 base){let fn i32(i32) action=fn [i32 base] i32(i32 x){return base+x;};return action;}fn i32 apply(fn i32(i32) action,i32 x){return action(x);}fn i32 main(){let Box first=Box{.action=make(40)};let Box second=first;let fn i32(i32) action=second.action;return apply(action,2);}",0,1);
}

@test fn void source_bound_native_mir_rejects_stale_source_and_module(){
    let source=b"module main;fn i32 main(){return 42;}";let p=program(source,0);let data=encoded(ref p);let raw=wire::content(ref data);var usize work=0;
    assert(rejected(raw,b"module main;fn i32 main(){return 43;}",0,ref mut work)==1001);
    work=0;assert(rejected(raw,b"module main;fn i32 main(){return 42;} ",0,ref mut work)==1001);
    work=0;assert(rejected(raw,source,1,ref mut work)==1001);
    work=0;assert(rejected(raw,source,limits::source_files(),ref mut work)==1001);
}

// Private exact format offsets are derived from the actual canonical owner.
// These tests mutate records without using an encoder that might refuse them.
struct Layout{usize targets;usize signatures;usize functions;usize blocks;usize operations;usize edges;usize closures;}
fn Layout layout(ref m::Program p){
    var usize at=32+m::source_view(p).len+m::payload_view(p).len;
    at+=8+shapes::count(m::nominal(p))*40;at+=8+shapes::field_count(m::nominal(p))*48;
    at+=8+shapes::abi_count(m::nominal(p))*48;at+=8+shapes::sdk_count(m::nominal(p))*24;at+=8+m::drop_count(p)*32;
    let targets=at+8;at=targets+m::target_count(p)*112;let signatures=at+8;at=signatures+m::signature_count(p)*8;
    let functions=at+8;at=functions+m::function_count(p)*88;at+=8+m::local_count(p)*56;let blocks=at+8;at=blocks+m::block_count(p)*160;
    let operations=at+8;at=operations+m::operation_count(p)*136;let edges=at+8;at=edges+m::edge_count(p)*8;
    return Layout{.targets,.signatures,.functions,.blocks,.operations,.edges,.closures=at+8};
}
fn u8 octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn void field(ref mut bytes::Buffer out,usize at,usize value){var usize rest=value;var usize i=0;let raw=bytes::view_mut(out);assert(at<=raw.len && raw.len-at>=8);while(i<8){raw[at+7-i]=octet(rest%256);rest/=256;i+=1;}assert(rest==0);}
fn ref u8[] closure_source(){return b"module main;fn fn i32(i32) make(i32 base,i32 extra){let fn i32(i32) action=fn [i32 base,i32 extra] i32(i32 x){return base+x+extra+1;};return action;}fn fn i32(i32) other(i32 base,i32 extra){let fn i32(i32) action=fn [i32 base,i32 extra] i32(i32 x){return base-x+extra+1;};return action;}fn i32 main(){let fn i32(i32) action=make(40,0);let fn i32(i32) unused=other(1,2);discard unused;return action(1);}";}
fn bytes::Buffer mutated(ref m::Program p,ref u8[] raw,u32 mutation){
    var out=copied(raw);let positions=layout(p);assert(m::closure_count(p)==2);let c=m::closure_at(p,0);let helper=m::target_at(p,c.target);
    var usize creation=0;var usize function=0;var usize i=1;
    while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.kind==40 && o.operator as usize==c.target){creation=i;}i+=1;}
    i=1;while(i<=m::function_count(p)){if(m::function_at(p,i).target==c.target){function=i;}i+=1;}assert(creation!=0 && function!=0);
    let made=m::operation_at(p,creation);let target=positions.targets+(c.target-1)*112;
    if(mutation==1){
        // Swap both constructors so every saved helper receipt remains used.
        // A one-sided retarget would be rejected by detached assembly itself.
        let other=m::closure_at(p,1);assert(c.captures==2 && other.captures==c.captures);
        var usize first=0;var usize second=0;i=1;
        while(i<=m::operation_count(p)){let o=m::operation_at(p,i);
            if(o.kind==40 && o.operator as usize==c.target){field(ref mut out,positions.operations+(i-1)*136+48,other.target);first+=1;}
            if(o.kind==40 && o.operator as usize==other.target){field(ref mut out,positions.operations+(i-1)*136+48,c.target);second+=1;}
            i+=1;
        }
        assert(first==1 && second==1);
    }
    if(mutation==2){assert(made.argument_count==2);field(ref mut out,positions.edges+made.argument_start*8,m::edge_at(p,made.argument_start+1));}
    if(mutation==3){field(ref mut out,positions.signatures+helper.signature_start*8,6);}
    if(mutation==4){field(ref mut out,target+32,1);}
    if(mutation==5){field(ref mut out,target+24,1);}
    if(mutation==6){field(ref mut out,target+64,13);}
    if(mutation==7){field(ref mut out,positions.closures+8,1);}
    if(mutation==8){field(ref mut out,positions.functions+(function-1)*88+80,1);}
    if(mutation==9){
        var usize changed=0;i=1;while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.function==function && (o.kind==4 || o.kind==5) && o.operator==1){field(ref mut out,positions.operations+(i-1)*136+48,2);changed+=1;}i+=1;}assert(changed!=0);
        // Checked arithmetic binds guard, successful operation and terminator.
        // Keep all three coherent so only source authentication can refuse it.
        i=1;while(i<=m::block_count(p)){let b=m::block_at(p,i);if(b.function==function && b.kind==4 && b.operator==1){field(ref mut out,positions.blocks+(i-1)*160+64,2);}i+=1;}
    }
    if(mutation==10){var bool changed=false;i=1;while(i<=m::operation_count(p) && !changed){let o=m::operation_at(p,i);if(o.function==function && o.kind==1 && o.payload_end-o.payload_start==1){bytes::view_mut(ref mut out)[32+m::source_view(p).len+o.payload_start]=50;changed=true;}i+=1;}assert(changed);}
    return out;
}
pub fn u32 source_bound_native_mir_forgery_probe(u32 mutation){
    let source=closure_source();let p=program(source,0);let data=encoded(ref p);assert(detached(wire::content(ref data)));
    let bad=mutated(ref p,wire::content(ref data),mutation);
    // Structurally valid malicious programs require source rebinding too.
    if((mutation==1 || mutation==2 || mutation==9 || mutation==10) && !detached(bytes::view(ref bad))){return 1;}
    var usize work=0;let code=rejected(bytes::view(ref bad),source,0,ref mut work);return if(code==1001){0}else{code+2};
}
@test fn void source_bound_native_mir_rejects_forged_helpers_captures_signatures_effects_and_bodies(){
    var u32 mutation=1;while(mutation<=10){assert(source_bound_native_mir_forgery_probe(mutation)==0);mutation+=1;}
}

@test fn void source_bound_native_mir_revalidates_source_even_when_saved_source_is_updated(){
    let source=b"module main;fn i32 main(){return 42;}";let changed=b"module main;fn i32 main(){return 43;}";
    let p=program(source,0);let data=encoded(ref p);var bad=copied(wire::content(ref data));
    // The embedded source now equals current input, but literal payload/MIR is
    // still from the old source. A source-only comparison would accept it.
    let text=bytes::view_mut(ref mut bad);var usize i=0;while(i<changed.len){text[24+i]=changed[i];i+=1;}
    assert(detached(bytes::view(ref bad)));var usize work=0;assert(rejected(bytes::view(ref bad),changed,0,ref mut work)==1001);
}

@test fn void source_bound_native_mir_rebuilds_written_helper_body_and_rejects_invalid_source(){
    let source=closure_source();let p=program(source,0);let data=encoded(ref p);var usize at=0;
    while(at+6<=source.len && !bytes::equal(source[at..at+6],b"base+x")){at+=1;}assert(at+6<=source.len);
    // Change only the original helper spelling, keeping old executable facts.
    // '-' is valid new source, '=' fails source admission, and '?' is invalid
    // syntax. A detached structural verifier cannot confer source authority.
    let replacements=b"-=?";var usize mutation=0;while(mutation<replacements.len){
        var changed=copied(source);bytes::view_mut(ref mut changed)[at+4]=replacements[mutation];
        var bad=copied(wire::content(ref data));bytes::view_mut(ref mut bad)[24+at+4]=replacements[mutation];
        assert(detached(bytes::view(ref bad)));var usize work=0;
        let code=rejected(bytes::view(ref bad),bytes::view(ref changed),0,ref mut work);
        assert(if(mutation==0){code==1001}else{code!=0});mutation+=1;
    }
}

@test fn void source_bound_native_mir_charges_exact_shared_work_without_reset(){
    let source=closure_source();let p=program(source,0);let data=encoded(ref p);let raw=wire::content(ref data);
    var usize cost=0;let baseline=restored(raw,source,0,ref mut cost);assert(proof::check(ref baseline).code==0 && cost>raw.len && cost<16777216);
    var usize work=16777216-cost;let fit=restored(raw,source,0,ref mut work);assert(work==16777216 && proof::check(ref fit).code==0);
    work=16777216-cost+1;assert(rejected(raw,source,0,ref mut work)==1002 && work<=16777216);
    work=16777216;assert(rejected(raw,source,0,ref mut work)==1002 && work==16777216);
    work=16777217;assert(rejected(raw,source,0,ref mut work)==1002 && work==16777217);
}

@test fn void source_bound_native_mir_rejects_noncanonical_versions_and_trailing_bytes(){
    let source=b"module main;fn i32 main(){return 42;}";let p=program(source,0);let data=encoded(ref p);var bad=copied(wire::content(ref data));var usize work=0;
    bytes::view_mut(ref mut bad)[7]=49;assert(rejected(bytes::view(ref bad),source,0,ref mut work)==1001);
    bad=copied(wire::content(ref data));assert(bytes::append_byte(ref mut bad,0).is_ok);work=0;assert(rejected(bytes::view(ref bad),source,0,ref mut work)==1001);
    work=0;assert(rejected(wire::content(ref data)[..7],source,0,ref mut work)==1001);
    let closure=program(closure_source(),0);let closure_data=encoded(ref closure);bad=copied(wire::content(ref closure_data));bytes::view_mut(ref mut bad)[7]=48;
    work=0;assert(rejected(bytes::view(ref bad),closure_source(),0,ref mut work)==1001);
}
