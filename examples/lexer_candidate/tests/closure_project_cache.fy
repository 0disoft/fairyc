module tests::closure_project_cache;
import project_compiler_input as input;
import project_module_visibility as visibility;
import project_coordinate_graph as coordinates;
import project_source_set as source_set;
import project_cache_environment as environment;
import project_cache_disk as disk;
import project_cache_lookup as lookup;
import project_cache_keys as keys;
import project_cache_artifact as cache;
import project_cache_artifact_wire as transport;
import project_compiler_core as core;
import project_compiler_cached as pipeline;
import generic_signature_diagnostics as diagnostics;
import module_project_identified_ir as identified;
import module_project_complete_c as complete_c;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_emit as c;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;
import syntax as syntax;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
import std::vec as v;

@effects(console) fn void require(bool ok,u32 point){if(!ok){discard io::write_stderr("closure-project-cache checkpoint=");discard io::write_stderr_u32(point);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{require(false,1);copied(raw)}};}
@effects(console) fn bytes::Buffer argument(usize index){var u8[4096] raw=[0;4096];let n=switch(process::arg(index,ref mut raw)){case .ok(p):{p}case .err(_):{require(false,2);0}};let ref u8[] view=ref raw;return copied(view[..n]);}
@effects(filesystem,console) fn visibility::Resolved loaded(ref u8[] root){var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(root,ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{require(false,3);loaded(root)}};}
@effects(filesystem,environment,console) fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{require(false,4);current()}};}
@effects(filesystem,console) fn disk::Store opened(ref u8[] path){return switch(disk::open(path,disk::Limits{.entries=16,.bytes=16777216,.record=8388608,.temporary_age=1})){case .ok(p):{p}case .err(_):{require(false,5);opened(path)}};}
@effects(console) fn Result<core::Complete,core::Problem> cold(ref visibility::Resolved loaded,bool aggregate){var usize projection=0;let table=switch(coordinates::collect(visibility::workspace_graph(loaded),ref mut projection)){case .ok(p):{p}case .err(_):{require(false,6);return cold(loaded,aggregate);}};let sources=switch(input::project(loaded,ref mut projection)){case .ok(p):{p}case .err(_):{require(false,7);return cold(loaded,aggregate);}};var usize source=0;var usize link=0;return if(aggregate){core::build_complete_aggregate_sources(input::sources(ref sources),input::owners(ref sources),input::bindings(ref sources),ref table,ref mut source,ref mut link)}else{core::build_complete(ref sources,ref table,ref mut source,ref mut link)};}
@effects(console) fn payload::Payload encoded(ref visibility::Resolved loaded,ref core::Complete complete,ref environment::Environment env,bool aggregate){var usize key=0;var usize artifact=0;return switch(cache::encode_closures(loaded,complete,env,aggregate,ref mut key,ref mut artifact)){case .ok(p):{p}case .err(_):{require(false,8);encoded(loaded,complete,env,aggregate)}};}
fn Result<core::Complete,cache::Problem> restore(ref u8[] raw,ref visibility::Resolved loaded,ref environment::Environment env,bool aggregate){var usize projection=0;var usize source=0;var usize artifact=0;var usize link=0;var usize key=0;return cache::restore_closures(raw,loaded,env,aggregate,ref mut projection,ref mut source,ref mut artifact,ref mut link,ref mut key);}

// Every control uses loader-owned current source and the production Complete.
@effects(filesystem,console) fn void controls(ref visibility::Resolved resolved,ref core::Complete complete,ref environment::Environment env,bool aggregate,ref u8[] raw){
    var usize work=0;let scalar=switch(lookup::collect(resolved,env,ref mut work)){case .ok(p):{p}case .err(_):{require(false,9);return;}};work=0;let aggregates=switch(lookup::collect_aggregates(resolved,env,ref mut work)){case .ok(p):{p}case .err(_):{require(false,10);return;}};work=0;let route=switch(lookup::collect_closures(resolved,env,aggregate,ref mut work)){case .ok(p):{p}case .err(_):{require(false,11);return;}};work=0;let opposite=switch(lookup::collect_closures(resolved,env,!aggregate,ref mut work)){case .ok(p):{p}case .err(_):{require(false,12);return;}};
    require(!bytes::equal(lookup::digest(ref route),lookup::digest(ref scalar)) && !bytes::equal(lookup::digest(ref route),lookup::digest(ref aggregates)) && !bytes::equal(lookup::digest(ref route),lookup::digest(ref opposite)) && !bytes::equal(lookup::digest(ref scalar),lookup::digest(ref aggregates)),13);
    work=0;require(!keys::collect(resolved,complete,env,ref mut work).is_ok,14);work=0;require(!keys::collect_aggregates(resolved,complete,env,ref mut work).is_ok,15);work=0;require(!keys::collect_closures(resolved,complete,env,!aggregate,ref mut work).is_ok,16);
    var usize artifact=0;work=0;require(!cache::encode(resolved,complete,env,ref mut work,ref mut artifact).is_ok,17);artifact=0;work=0;require(!cache::encode_aggregates(resolved,complete,env,ref mut work,ref mut artifact).is_ok,18);
    work=0;let parts=switch(transport::preflight_closures(raw,aggregate,ref mut work)){case .ok(p):{p}case .err(_):{require(false,19);return;}};require(bytes::equal(raw[..8],b"FYCACHE2") && bytes::equal(raw[parts.keys.start..parts.keys.start+8],b"FYCKEYS2") && bytes::equal(raw[parts.program.start..parts.program.start+8],b"FYPRJ002"),20);
    work=0;require(!transport::preflight(raw,ref mut work).is_ok,21);work=0;require(!transport::preflight_aggregates(raw,ref mut work).is_ok,22);require(!restore(raw,resolved,env,!aggregate).is_ok,23);
    var bad=copied(raw);bytes::view_mut(ref mut bad)[7]=48;require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,24);bytes::view_mut(ref mut bad)[7]=49;require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,25);
    bad=copied(raw);bytes::view_mut(ref mut bad)[parts.keys.start+7]=49;require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,26);
    // Same raw FYCACHE2 transport as UNIT, but never the UNIT identity domain.
    bad=copied(raw);var usize i=0;while(i<8){bytes::view_mut(ref mut bad)[parts.keys.start+i]=b"FYPUNIT2"[i];i+=1;}require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,27);
    bad=copied(raw);bytes::view_mut(ref mut bad)[parts.keys.start+15]=if(aggregate){0}else{1};require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,28);bytes::view_mut(ref mut bad)[parts.keys.start+15]=2;require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,29);
    bad=copied(raw);bytes::view_mut(ref mut bad)[parts.program.start+7]=49;require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,30);bad=copied(raw);bytes::view_mut(ref mut bad)[parts.program.start+7]=48;require(!restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,31);
    bad=copied(raw);bytes::view_mut(ref mut bad)[parts.keys.start+32]=if(raw[parts.keys.start+32]==97){98}else{97};require(switch(restore(bytes::view(ref bad),resolved,env,aggregate)){case .ok(_):{false}case .err(e):{e.phase==6 && e.code==1001}},32);
    require(!restore(raw[..raw.len-1],resolved,env,aggregate).is_ok,33);bad=copied(raw);require(bytes::append_byte(ref mut bad,0).is_ok && !restore(bytes::view(ref bad),resolved,env,aggregate).is_ok,34);
    var usize projection=0;var usize source=0;artifact=wire::work_limit()+1;var usize link=0;var usize key=0;require(switch(cache::restore_closures(raw,resolved,env,aggregate,ref mut projection,ref mut source,ref mut artifact,ref mut link,ref mut key)){case .ok(_):{false}case .err(e):{e.code==1002}} && artifact==wire::work_limit()+1 && source==0 && projection==0,35);
    let other_path=argument(5);let other=loaded(bytes::view(ref other_path));require(switch(restore(raw,ref other,env,aggregate)){case .ok(_):{false}case .err(e):{e.phase==4}},36);
}
@effects(console) fn void member(ref u8[] raw){require(io::write_usize(raw.len).is_ok && io::write("\n").is_ok && io::write_bytes(raw).is_ok,37);}
@effects(console) fn void emit(core::Complete complete,ref visibility::Resolved loaded){var usize work=0;let sources=switch(input::project(loaded,ref mut work)){case .ok(p):{p}case .err(_):{require(false,38);return;}};var usize entry=0;{let p=identified::ir_program(core::checked_program(ref complete));var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.kind==3 && bytes::equal(syntax::declaration_name(input::source(ref sources,f.module_index),f.declaration),b"main")){require(entry==0,39);entry=i;}i+=1;}}require(entry!=0,40);let bundle=switch(complete_c::partition_entry(complete,entry,128)){case .ok(p):{p}case .err(_):{require(false,41);return;}};require(complete_c::has_complete(ref bundle),42);require(io::write("FYCPARTS1\n").is_ok,43);member(c::project_header(ref bundle));member(c::project_literals(ref bundle));require(io::write_usize(c::project_unit_count(ref bundle)).is_ok && io::write("\n").is_ok,44);var usize i=0;while(i<c::project_unit_count(ref bundle)){member(c::project_source(ref bundle,i));i+=1;}}

@effects(filesystem,environment,clock,console) pub fn i32 run(){
    let root=argument(1);let path=argument(2);let mode_bytes=argument(3);let mode=bytes::view(ref mode_bytes);let feature=argument(4);let aggregate=bytes::equal(bytes::view(ref feature),b"aggregate");let resolved=loaded(bytes::view(ref root));let env=current();let store=opened(bytes::view(ref path));var history=cache::empty_disk_work();if(bytes::equal(mode,b"exhausted")){history.artifact=wire::work_limit()+1;}if(bytes::equal(mode,b"io")){history.io=disk::work_limit()+1;}
    if(bytes::equal(mode,b"pipeline-cold") || bytes::equal(mode,b"pipeline-warm")){
        var usize projection=0;let table=switch(coordinates::collect(visibility::workspace_graph(ref resolved),ref mut projection)){case .ok(p):{p}case .err(_):{require(false,57);return 1;}};let sources=switch(input::project(ref resolved,ref mut projection)){case .ok(p):{p}case .err(_):{require(false,58);return 1;}};
        var reports=v::empty<diagnostics::Reports>();var usize source=0;var usize link=0;var observed=pipeline::empty();let complete=switch(pipeline::build(ref store,ref env,ref resolved,ref sources,ref table,ref mut reports,ref mut history,ref mut source,ref mut link,ref mut observed)){case .ok(p):{p}case .err(_):{require(false,59);return 1;}};
        if(bytes::equal(mode,b"pipeline-warm")){require(observed.path==2 && observed.lookup_phase==0 && observed.lookup_code==0 && !observed.stored && observed.store_phase==0 && observed.store_code==0 && history.source>0 && history.artifact>0 && ir::work(identified::ir_program(core::checked_program(ref complete)))==0,60);}else{require(observed.path==1 && observed.stored && observed.lookup_phase==2 && observed.lookup_code==2 && observed.store_phase==0 && observed.store_code==0,61);}emit(complete,ref resolved);return 0;
    }
    let restored=cache::from_disk_closures(ref store,ref resolved,ref env,aggregate,ref mut history);
    if(bytes::equal(mode,b"invalid")){require(!restored.is_ok && !cold(ref resolved,aggregate).is_ok,45);return 0;}
    let expected_hit=bytes::equal(mode,b"warm");let complete=switch(restored){case .ok(p):{require(expected_hit && history.source>0 && history.artifact>0 && history.link>0 && history.key>0,46);require(ir::work(identified::ir_program(core::checked_program(ref p)))==0,47);p}case .err(e):{require(!expected_hit,48);if(bytes::equal(mode,b"exhausted")){require(e.code==1002 && history.artifact==wire::work_limit()+1,49);}else if(bytes::equal(mode,b"io")){require(e.phase==2 && e.code==9,50);}switch(cold(ref resolved,aggregate)){case .ok(p):{p}case .err(_):{require(false,51);return 1;}}}};
    let fresh=switch(cold(ref resolved,aggregate)){case .ok(p):{p}case .err(_):{require(false,52);return 1;}};let actual=encoded(ref resolved,ref complete,ref env,aggregate);let expected=encoded(ref resolved,ref fresh,ref env,aggregate);require(bytes::equal(payload::content(ref actual),payload::content(ref expected)),53);require(clean::closures(ir::cleanup_ir(identified::ir_program(core::checked_program(ref complete)))) && clean::project(ir::cleanup_ir(identified::ir_program(core::checked_program(ref complete)))),54);
    if(bytes::equal(mode,b"cold")){controls(ref resolved,ref complete,ref env,aggregate,payload::content(ref actual));}if(bytes::equal(mode,b"exhausted") || bytes::equal(mode,b"io")){require(!cache::to_disk_closures(ref store,ref resolved,ref complete,ref env,aggregate,ref mut history).is_ok,55);}else{require(cache::to_disk_closures(ref store,ref resolved,ref complete,ref env,aggregate,ref mut history).is_ok,56);}emit(complete,ref resolved);return 0;
}
