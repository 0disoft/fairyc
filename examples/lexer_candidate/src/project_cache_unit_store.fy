module project_cache_unit_store;
import project_cache_unit_key as keys;
import project_cache_disk as disk;
import module_unit_artifact_admission as admission;
import module_identified_ir as identified;
import module_program_artifact_encode as encoder;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;
import module_signatures as signatures;
import module_catalog as catalog;
import generic_source_plan as plan;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Work{usize key;usize scan;usize io;usize source;usize artifact;usize hits;usize misses;usize stored;usize cold0;usize cold1;usize cold2;usize cold3;usize cold4;usize cold5;usize cold6;usize cold7;}
pub fn Work empty(){return Work{.key=0,.scan=0,.io=0,.source=0,.artifact=0,.hits=0,.misses=0,.stored=0,.cold0=0,.cold1=0,.cold2=0,.cold3=0,.cold4=0,.cold5=0,.cold6=0,.cold7=0};}
pub fn void mark_cold(ref mut Work p,usize index){assert(index<512);let usize mask=1<<(index%64);if(index/64==0){p.cold0=p.cold0|mask;}else if(index/64==1){p.cold1=p.cold1|mask;}else if(index/64==2){p.cold2=p.cold2|mask;}else if(index/64==3){p.cold3=p.cold3|mask;}else if(index/64==4){p.cold4=p.cold4|mask;}else if(index/64==5){p.cold5=p.cold5|mask;}else if(index/64==6){p.cold6=p.cold6|mask;}else if(index/64==7){p.cold7=p.cold7|mask;}}
pub fn bool is_cold(ref Work p,usize index){assert(index<512);let usize mask=1<<(index%64);var usize selected=0;if(index/64==0){selected=p.cold0;}else if(index/64==1){selected=p.cold1;}else if(index/64==2){selected=p.cold2;}else if(index/64==3){selected=p.cold3;}else if(index/64==4){selected=p.cold4;}else if(index/64==5){selected=p.cold5;}else if(index/64==6){selected=p.cold6;}else if(index/64==7){selected=p.cold7;}return (selected&mask)!=0;}
// Share the bounded store and retention policy with whole-project generations.
// Distinct route preimages and envelope members keep their admission isolated.
pub fn Result<payload::Payload,u32> encode(ref keys::Identity key,ref identified::Program unit,ref mut usize work){
    let program=switch(encoder::encode(identified::ir_program(unit),work)){case .ok(p):{p}case .err(code):{return .err(code);}};var wire::Status status=wire::Status{.code=0};var out=wire::writer_marker(b"FYCACHE0",ref mut status,work);wire::member(ref mut out,keys::canonical(key),ref mut status,work);wire::member(ref mut out,payload::content(ref program),ref mut status,work);return wire::finish(out,status);
}
pub fn Result<identified::Program,u32> restore(ref u8[] raw,ref keys::Identity key,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref plan::Plan original,usize first,usize stop,ref mut usize source_work,ref mut usize artifact_work){
    var cursor=switch(wire::cursor_marker(raw,b"FYCACHE0",artifact_work)){case .ok(p):{p}case .err(code):{return .err(code);}};let identity=wire::member_from(raw,ref mut cursor,16777216,b"FYPUNIT0",artifact_work);let program=wire::member_from(raw,ref mut cursor,wire::byte_limit(),b"FYPROG00",artifact_work);let code=wire::end(raw,ref mut cursor,artifact_work);if(code!=0){return .err(code);}let expected=keys::canonical(key);if(!wire::charge(artifact_work,expected.len+1)){return .err(1002);}if(!bytes::equal(raw[identity.start..identity.end],expected)){return .err(1001);}
    return admission::restore(raw[program.start..program.end],sources,scope,metadata,original,first,stop,source_work,artifact_work);
}
@effects(filesystem)
pub fn Result<identified::Program,u32> load(ref disk::Store store,ref keys::Identity key,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog scope,ref signatures::Program metadata,ref plan::Plan original,usize first,usize stop,ref mut Work work){
    var usize scan=work.scan;let result=disk::candidates(store,keys::route(key),ref mut scan);work.scan=scan;let cursor=switch(result){case .ok(p):{p}case .err(_):{return .err(1001);}};var usize index=disk::candidate_count(ref cursor);
    while(index>0){var usize io=work.io;let candidate_result=disk::next(store,ref cursor,ref mut index,ref mut io);work.io=io;let candidate=switch(candidate_result){case .ok(p):{p}case .err(_):{return .err(1001);}};
        var usize source=work.source;var usize artifact=work.artifact;let result=restore(disk::candidate_bytes(ref candidate),key,sources,scope,metadata,original,first,stop,ref mut source,ref mut artifact);work.source=source;work.artifact=artifact;
        switch(result){case .ok(p):{return .ok(p);}case .err(code):{if(code==10 || code==11 || code==1002 || code==1003){return .err(code);}}};
        // A valid old dependency contract may simply be incompatible today.
        // Do not delete that generation as if semantic mismatch meant corruption.
    }return .err(1001);
}
@effects(filesystem,clock)
pub fn bool save(ref disk::Store store,ref keys::Identity key,ref identified::Program unit,ref mut Work work){
    var usize artifact=work.artifact;let encoded=encode(key,unit,ref mut artifact);work.artifact=artifact;let data=switch(encoded){case .ok(p):{p}case .err(_):{return false;}};var usize io=work.io;var usize scan=work.scan;let result=disk::put(store,keys::route(key),payload::content(ref data),ref mut io,ref mut scan);work.io=io;work.scan=scan;return result.is_ok;
}
