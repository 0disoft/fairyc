module tests::generic_aggregate_artifact;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_program_artifact_encode as artifact_pe;
import module_program_artifact_decode as artifact_pd;
import module_project_artifact_encode as artifact_encode;
import module_project_artifact_decode as artifact_decode;
import module_project_artifact_model as artifact_model;
import module_cleanup_artifact_encode as artifact_ce;
import module_cleanup_artifact_decode as artifact_cd;
import module_identity_artifact_encode as artifact_ie;
import module_identity_artifact_decode as artifact_id;
import module_identity_artifact_model as artifact_ids;
import module_program_artifact_wire as artifact_wire;
import candidate_artifact_wire as artifact_component;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) pub fn void verify_artifact(ref core::Complete complete,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table,bool controls){
 let p=checked::ir_program(core::checked_program(complete));assert(clean::aggregates(ir::cleanup_ir(p)));var usize work=0;
 let packet=switch(artifact_pe::encode(p,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("aggregate program encode ");discard io::write_stderr_u32(code);assert(false);return;}};let raw=artifact_component::content(ref packet);assert(bytes::equal(raw[..8],b"FYPROG01"));work=0;
 let restored=switch(artifact_pd::decode_aggregates(raw,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("aggregate program decode ");discard io::write_stderr_u32(code);assert(false);return;}};assert(clean::aggregates(ir::cleanup_ir(ref restored)) && clean::project(ir::cleanup_ir(ref restored)) && ir::count(ref restored)==ir::count(p));work=0;
 let replay=switch(artifact_pe::encode(ref restored,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(raw,artifact_component::content(ref replay)));
 work=0;assert(switch(artifact_pd::decode(raw,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});work=0;
 let project=switch(artifact_encode::encode(core::checked_program(complete),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("aggregate project encode ");discard io::write_stderr_u32(code);assert(false);return;}};let project_raw=artifact_component::content(ref project);assert(bytes::equal(project_raw[..8],b"FYPRJ001"));work=0;
 let saved=switch(artifact_decode::decode_aggregates(project_raw,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("aggregate project decode ");discard io::write_stderr_u32(code);assert(false);return;}};let cost=work;work=0;let again=switch(artifact_encode::restored(ref saved,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(project_raw,artifact_component::content(ref again)));work=0;
 assert(switch(artifact_decode::decode(project_raw,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 let ids=artifact_model::source_identities(ref saved);var usize i=1;while(i<=artifact_ids::type_count(ids)){let t=clean::type_at(ir::cleanup_ir(p),i);let row=artifact_ids::type_at(ids,i);assert((row.node!=0)==(t.kind==20 || t.kind==21));i+=1;}
 if(controls){artifact_controls(p,raw,project_raw,ids,cost);}
 // A detached decoder cannot bypass the still-closed current aggregate owner.
 var usize a=0;var usize b=0;var usize c=0;assert(switch(core::restore_complete_sources(saved,sources,owners,imports,table,ref mut a,ref mut b,ref mut c)){case .ok(_):{false}case .err(e):{e.phase==7}});
}
fn bytes::Buffer artifact_copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);artifact_copy(raw)}};}
fn u8 artifact_octet(usize value){let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn void artifact_word(ref mut bytes::Buffer raw,usize at,usize value){assert(at<=bytes::len(raw) && bytes::len(raw)-at>=8);var usize n=value;var usize i=0;while(i<8){bytes::view_mut(raw)[at+7-i]=artifact_octet(n%256);n/=256;i+=1;}assert(n==0);}
fn void artifact_controls(ref ir::Program p,ref u8[] raw,ref u8[] project_raw,ref artifact_ids::Snapshot ids,usize project_cost){
 var usize work=0;assert(switch(artifact_pd::decode_aggregates(raw[..raw.len-1],ref mut work)){case .ok(_):{false}case .err(_):{true}});var bad=artifact_copy(raw);assert(bytes::append_byte(ref mut bad,0).is_ok);work=0;assert(switch(artifact_pd::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(_):{true}});
 bad=artifact_copy(raw);bytes::view_mut(ref mut bad)[23]=48;work=0;assert(switch(artifact_pd::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});bytes::view_mut(ref mut bad)[7]=48;work=0;assert(switch(artifact_pd::decode(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 work=artifact_wire::work_limit()-project_cost;assert(switch(artifact_decode::decode_aggregates(project_raw,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==artifact_wire::work_limit());work=artifact_wire::work_limit()-project_cost+1;assert(switch(artifact_decode::decode_aggregates(project_raw,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}});
 work=0;let payload=switch(artifact_ce::encode(ir::cleanup_ir(p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let cleanup_raw=artifact_component::content(ref payload);assert(bytes::equal(cleanup_raw[..8],b"FYCLEAN1"));work=0;assert(switch(artifact_cd::decode(cleanup_raw,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 var usize tuple=0;var usize instance=0;var usize i=1;while(i<=clean::type_count(ir::cleanup_ir(p))){let t=clean::type_at(ir::cleanup_ir(p),i);if(t.kind==22 && t.second!=0){tuple=i;}if(t.kind==20 || t.kind==21){instance=i;}i+=1;}assert(tuple!=0 && instance!=0);
 bad=artifact_copy(cleanup_raw);artifact_word(ref mut bad,16,4097);work=0;assert(switch(artifact_cd::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}});
 bad=artifact_copy(cleanup_raw);artifact_word(ref mut bad,24+clean::type_count(ir::cleanup_ir(p))*104+8+32,tuple);work=0;assert(switch(artifact_cd::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 bad=artifact_copy(cleanup_raw);artifact_word(ref mut bad,24+(tuple-1)*48+40,tuple);work=0;assert(switch(artifact_cd::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 bad=artifact_copy(cleanup_raw);artifact_word(ref mut bad,24+(instance-1)*48+32,0);work=0;assert(switch(artifact_cd::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 work=0;let identity=switch(artifact_ie::encode(ids,ir::cleanup_ir(p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};bad=artifact_copy(artifact_component::content(ref identity));artifact_word(ref mut bad,16+(instance-1)*24,0);work=0;assert(switch(artifact_id::decode(bytes::view(ref bad),ir::cleanup_ir(p),ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});
 work=0;assert(switch(artifact_decode::decode_aggregates(project_raw[..project_raw.len-1],ref mut work)){case .ok(_):{false}case .err(_):{true}});bad=artifact_copy(project_raw);assert(bytes::append_byte(ref mut bad,0).is_ok);work=0;assert(switch(artifact_decode::decode_aggregates(bytes::view(ref bad),ref mut work)){case .ok(_):{false}case .err(_):{true}});
}
