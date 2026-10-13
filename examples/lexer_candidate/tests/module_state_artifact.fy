module tests::module_state_artifact;
import tests::module_state_artifact_fixture as fixture_owner;
import module_state_artifact as codec;
import module_state_layout as state;
import module_metadata_artifact_model as metadata;
import module_metadata_artifact_encode as meta_encoder;
import module_metadata_artifact_decode as meta_decoder;
import module_cleanup_artifact_encode as clean_encoder;
import module_cleanup_artifact_decode as clean_decoder;
import module_body_artifact_encode as body_encoder;
import module_body_artifact_decode as body_decoder;
import module_sdk_artifact as sdk_codec;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_sdk_sources as sdk;
import candidate_artifact_wire as wire;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
@effects(filesystem) fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};}
@effects(console) fn wire::Payload encoded(ref state::Plan p,ref mut usize work){return switch(codec::encode(p,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p,work);}};}
@effects(console) fn state::Plan decoded(ref u8[] raw,ref mut usize work){return switch(codec::decode(raw,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return decoded(raw,work);}};}
fn u32 rejected(ref u8[] raw){var usize work=0;return switch(codec::decode(raw,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
fn void same(ref state::Plan a,ref state::Plan b){assert(state::count(a)==state::count(b) && state::field_count(a)==state::field_count(b) && bytes::equal(state::names(a),state::names(b)));var usize i=0;
    while(i<state::count(a)){let x=state::at(a,i);let y=state::at(b,i);assert(x.owner==y.owner && x.module_index==y.module_index && x.declaration==y.declaration && x.start==y.start && x.end==y.end && x.uid_start==y.uid_start && x.uid_end==y.uid_end && x.field_start==y.field_start && x.field_count==y.field_count && state::equal(x.hash,y.hash));i+=1;}
    i=0;while(i<state::field_count(a)){let x=state::field_at(a,i);let y=state::field_at(b,i);assert(x.kind==y.kind && x.key==y.key && x.value==y.value && x.name_start==y.name_start && x.name_end==y.name_end && x.start==y.start && x.end==y.end && state::equal(x.prefix,y.prefix));i+=1;}
}
@effects(filesystem,console) fn state::Plan detached(){let raw=fixture();var usize work=0;return decoded(bytes::view(ref raw),ref mut work);}
@test @effects(filesystem,console)
fn void state_binary_matches_golden_and_preserves_detached_identity(){let raw=fixture();let p=fixture_owner::sample();var usize work=0;let data=encoded(ref p,ref mut work);assert(bytes::equal(wire::content(ref data),bytes::view(ref raw)));let restored=detached();same(ref p,ref restored);assert(state::count(ref restored)==1 && state::field_count(ref restored)==5 && bytes::equal(state::uid(ref restored,0),b"example.token"));
    work=0;assert(state::validate(ref restored,ref mut work)==0);work=0;let again=encoded(ref restored,ref mut work);assert(bytes::equal(wire::content(ref data),wire::content(ref again)));
    let empty=state::empty();work=0;let blank=encoded(ref empty,ref mut work);assert(wire::content(ref blank).len==32);work=0;let copy=decoded(wire::content(ref blank),ref mut work);assert(state::count(ref copy)==0 && state::name_bytes(ref copy)==0);
    let scopes=fixture_owner::scoped();work=0;let scoped_data=encoded(ref scopes,ref mut work);work=0;let scoped_copy=decoded(wire::content(ref scoped_data),ref mut work);same(ref scopes,ref scoped_copy);assert(state::count(ref scoped_copy)==2 && state::at(ref scoped_copy,0).owner==0 && state::at(ref scoped_copy,1).owner==1);
}
@test @effects(filesystem,console)
fn void state_codec_accounts_for_hash_work_and_exact_boundaries(){let raw=fixture();let p=fixture_owner::sample();var usize proof_work=0;assert(state::validate(ref p,ref mut proof_work)==0 && proof_work>0);
    var usize cost=0;let data=encoded(ref p,ref mut cost);assert(cost==9+95*9+49+proof_work);
    var usize work=16777216-cost;let fit=encoded(ref p,ref mut work);assert(work==16777216 && bytes::equal(wire::content(ref fit),wire::content(ref data)));
    work=16777216-cost+1;assert(switch(codec::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    cost=0;let restored=decoded(bytes::view(ref raw),ref mut cost);assert(cost==9+95*9+50+proof_work);work=16777216-cost;let exact=decoded(bytes::view(ref raw),ref mut work);assert(work==16777216);same(ref restored,ref exact);
    work=16777216-cost+1;assert(switch(codec::decode(bytes::view(ref raw),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    work=16777217;assert(switch(codec::decode(bytes::view(ref raw),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==16777217);
    var changed=copied(bytes::view(ref raw));bytes::view_mut(ref mut changed)[95]=0;work=11;assert(switch(codec::decode(bytes::view(ref changed),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}} && work==cost+11);
}
@test @effects(filesystem)
fn void state_codec_rejects_truncation_bad_hashes_and_noncanonical_fields(){let raw=fixture();let view=bytes::view(ref raw);assert(view.len==816);var usize i=0;while(i<view.len){assert(rejected(view[..i])!=0);i+=1;}var trailing=copied(view);assert(bytes::append_byte(ref mut trailing,0).is_ok && rejected(bytes::view(ref trailing))==1001);
    var u32 mode=0;while(mode<21){var changed=copied(view);let data=bytes::view_mut(ref mut changed);
        if(mode==0){data[7]=49;}else if(mode==1){data[8]=255;}else if(mode==2){data[16]=1;}else if(mode==3){data[24]=1;}else if(mode==4){data[39]=0;}
        else if(mode==5){data[47]=151;}else if(mode==6){data[63]=1;}else if(mode==7){data[71]=255;}else if(mode==8){data[79]=1;}else if(mode==9){data[87]=6;}
        else if(mode==10){data[88]=1;}else if(mode==11){data[95]=0;}else if(mode==12){data[152]=255;}else if(mode==13){data[167]=0;}else if(mode==14){data[175]=1;}
        else if(mode==15){data[183]=0;}else if(mode==16){data[191]=0;}else if(mode==17){data[199]=255;}else if(mode==18){data[216]=1;}else if(mode==19){data[223]=0;}else{data[760]=255;}
        assert(rejected(bytes::view(ref changed))==if(mode==1 || mode==12 || mode==20){1002}else{1001});mode+=1;
    }
    var bad_uid=copied(view);bytes::view_mut(ref mut bad_uid)[768]=47;assert(rejected(bytes::view(ref bad_uid))==1001);
}
@effects(console) fn ir::Program source_restored(){let original=fixture_owner::program();var usize used=0;let states_data=encoded(ir::state_layouts(ref original),ref mut used);used=0;let states=decoded(wire::content(ref states_data),ref mut used);same(ir::state_layouts(ref original),ref states);
    used=0;let clean_data=switch(clean_encoder::encode(ir::cleanup_ir(ref original),ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};used=0;let cleaned=switch(clean_decoder::decode(wire::content(ref clean_data),ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};
    var v::Vec<ir::Body> bodies=v::empty<ir::Body>();var usize frame=1;while(frame<=ir::count(ref original)){used=0;let body_data=switch(body_encoder::encode(ref original,frame,ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};used=0;let b=switch(body_decoder::decode(wire::content(ref body_data),ref cleaned,frame,ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};assert(v::push<ir::Body>(ref mut bodies,b).is_ok);frame+=1;}
    used=0;let sdk_data=switch(sdk_codec::encode(ir::sdk_units(ref original),ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};used=0;let sources=switch(sdk_codec::decode(wire::content(ref sdk_data),ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};
    used=0;let meta_data=switch(meta_encoder::program(ref original,ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};used=0;let meta=switch(meta_decoder::decode(wire::content(ref meta_data),ref cleaned,sdk::module_count(ref sources),ref mut used)){case .ok(p):{p}case .err(_):{assert(false);return source_restored();}};
    used=0;return switch(metadata::restore(ref meta,states,sources,cleaned,bodies,ref mut used)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return source_restored();}};
}
@test @effects(console)
fn void source_state_roundtrip_reassembles_all_program_components(){let p=source_restored();assert(proof::check(ref p).code==0 && ir::work(ref p)==0 && state::count(ir::state_layouts(ref p))==1 && state::field_count(ir::state_layouts(ref p))==5 && bytes::equal(state::uid(ir::state_layouts(ref p),0),b"example.token"));assert(state::equal(state::at(ir::state_layouts(ref p),0).hash,[2228573728,2676884447,2887346710,2989097650,72141878,1087492126,1089733372,2723509058]));}
