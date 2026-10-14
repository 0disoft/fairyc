module tests::project_cache_disk_restore;
import project_cache_artifact as cache;
import project_cache_artifact_wire as transport;
import project_cache_disk as disk;
import project_cache_lookup as lookup;
import project_cache_environment as environment;
import project_cache_keys as keys;
import project_compiler_core as core;
import project_compiler_input as input;
import project_coordinate_graph as coordinates;
import project_module_visibility as visibility;
import project_source_set as sources;
import module_project_identified_ir as checked;
import module_project_complete_c as complete_c;
import module_project_interfaces as interfaces;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_emit as c;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;
import windows_rooted_file as rooted;
import file_source as files;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var p=bytes::empty();assert(bytes::append(ref mut p,root).is_ok && bytes::append_byte(ref mut p,47).is_ok && bytes::append(ref mut p,leaf).is_ok);return p;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] raw=[0;4096];let n=switch(process::arg(1,ref mut raw)){case .ok(p):{p}case .err(_):{assert(false);0}};let ref u8[] path=ref raw;return switch(files::manifest(path[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn text::CString cstr(ref u8[] raw){let s=switch(text::validate_utf8(raw)){case .ok(p):{p}case .err(_):{assert(false);return cstr(raw);}};return switch(text::cstring_from_text(s)){case .ok(p):{p}case .err(_):{assert(false);cstr(raw)}};}
@effects(filesystem) fn void mkdir(ref u8[] path){let s=cstr(path);assert(fs::create_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem,console) fn rooted::Directory directory(ref u8[] path){return switch(rooted::open(path)){case .ok(p):{p}case .err(_):{assert(false);directory(path)}};}
@effects(filesystem,console) fn void clear(ref u8[] path){{let root=directory(path);var usize work=0;let entries=switch(rooted::scan(ref root,4096,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;while(i<rooted::count(ref entries)){assert(switch(rooted::erase_if(ref root,rooted::name(ref entries,i),rooted::metadata(ref entries,i))){case .ok(p):{p}case .err(_):{false}});i+=1;}}let s=cstr(path);assert(fs::remove_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem,console) fn usize count(ref u8[] path){let root=directory(path);var usize work=0;let entries=switch(rooted::scan(ref root,4096,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return 0;}};return rooted::count(ref entries);}
@effects(filesystem,console) fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(visibility::load_phase(ref e));assert(false);return load(root,leaf);}};}
@effects(console) fn core::Complete cold(ref visibility::Resolved loaded){var usize projection=0;let table=switch(coordinates::collect(visibility::workspace_graph(loaded),ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return cold(loaded);}};let projected=switch(input::project(loaded,ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return cold(loaded);}};var usize source=0;var usize link=0;return switch(core::build_complete(ref projected,ref table,ref mut source,ref mut link)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return cold(loaded);}};}
@effects(filesystem,environment,console) fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{assert(false);current()}};}
@effects(filesystem,console) fn disk::Store store(ref u8[] path){return switch(disk::open(path,disk::Limits{.entries=8,.bytes=8388608,.record=8388608,.temporary_age=1})){case .ok(p):{p}case .err(_):{assert(false);store(path)}};}
@effects(console) fn lookup::Identity route(ref visibility::Resolved loaded,ref environment::Environment env){var usize work=0;return switch(lookup::collect(loaded,env,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);route(loaded,env)}};}
@effects(console) fn payload::Payload saved(ref visibility::Resolved loaded,ref core::Complete complete,ref environment::Environment env){var usize key=0;var usize artifact=0;return switch(cache::encode(loaded,complete,env,ref mut key,ref mut artifact)){case .ok(p):{p}case .err(_):{assert(false);saved(loaded,complete,env)}};}
@effects(filesystem,clock,console) fn void publish(ref disk::Store store,ref visibility::Resolved loaded,ref core::Complete complete,ref environment::Environment env){var work=cache::empty_disk_work();switch(cache::to_disk(store,loaded,complete,env,ref mut work)){case .ok(stats):{assert(stats.entries>0 && stats.bytes>0);}case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr_u32(e.code);assert(false);}};}
@effects(filesystem,console) fn core::Complete restored(ref disk::Store store,ref visibility::Resolved loaded,ref environment::Environment env,ref mut cache::DiskWork work){return switch(cache::from_disk(store,loaded,env,work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("disk restore ");discard io::write_stderr_u32(e.phase);discard io::write_stderr(" ");discard io::write_stderr_u32(e.code);assert(false);restored(store,loaded,env,work)}};}
@effects(filesystem,clock,console) fn void raw_publish(ref disk::Store store,ref lookup::Identity route,ref u8[] raw){var usize work=0;var usize scan=0;assert(disk::put(store,lookup::digest(route),raw,ref mut work,ref mut scan).is_ok);}
@effects(filesystem,console) fn bytes::Buffer raw_load(ref disk::Store store,ref lookup::Identity route){var usize work=0;var usize scan=0;return switch(disk::load(store,lookup::digest(route),ref mut work,ref mut scan)){case .ok(p):{p}case .err(_):{assert(false);raw_load(store,route)}};}

@test @effects(filesystem,environment,clock,console) fn void disk_admission_rejects_poisoned_preimage_and_retains_owner_on_exhaustion(){
    let base=root();let env=current();let path=join(bytes::view(ref base),b"cache-plain");mkdir(bytes::view(ref path));
    {let loaded=load(bytes::view(ref base),b"Plain");let store=store(bytes::view(ref path));var missing=cache::empty_disk_work();assert(switch(cache::from_disk(ref store,ref loaded,ref env,ref mut missing)){case .ok(_):{false}case .err(e):{e.phase==2 && e.code==2}});let complete=cold(ref loaded);let encoded=saved(ref loaded,ref complete,ref env);publish(ref store,ref loaded,ref complete,ref env);let route=route(ref loaded,ref env);let read=raw_load(ref store,ref route);assert(bytes::equal(bytes::view(ref read),payload::content(ref encoded)));
        var usize work=0;let parts=switch(transport::preflight(bytes::view(ref read),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let at=parts.keys.start+24;assert(at<bytes::len(ref read));var poisoned=switch(bytes::copy(bytes::view(ref read))){case .ok(p):{p}case .err(_):{assert(false);return;}};bytes::view_mut(ref mut poisoned)[at]=98;raw_publish(ref store,ref route,bytes::view(ref poisoned));assert(count(bytes::view(ref path))==2);
        var admitted=cache::empty_disk_work();let owner=restored(ref store,ref loaded,ref env,ref mut admitted);assert(interfaces::count(core::public_interfaces(ref owner))==3 && count(bytes::view(ref path))==1 && admitted.artifact>0 && admitted.source>0 && admitted.key>0);
        var limited=cache::empty_disk_work();limited.artifact=wire::work_limit()+1;assert(switch(cache::from_disk(ref store,ref loaded,ref env,ref mut limited)){case .ok(_):{false}case .err(e):{e.phase==3 && e.artifact.phase==1 && e.code==1002}});assert(count(bytes::view(ref path))==1);limited=cache::empty_disk_work();limited.io=disk::work_limit()+1;assert(switch(cache::from_disk(ref store,ref loaded,ref env,ref mut limited)){case .ok(_):{false}case .err(e):{e.phase==2 && e.code==9}});assert(count(bytes::view(ref path))==1);
    }clear(bytes::view(ref path));
}
@test @effects(filesystem,environment,clock,console) fn void current_routing_and_source_admission_reject_stale_body_and_wrong_source(){
    let base=root();let env=current();let path=join(bytes::view(ref base),b"cache-source");mkdir(bytes::view(ref path));
    {let original=load(bytes::view(ref base),b"Aliases");let complete=cold(ref original);let encoded=saved(ref original,ref complete,ref env);let store=store(bytes::view(ref path));publish(ref store,ref original,ref complete,ref env);let changed=load(bytes::view(ref base),b"PrivateEdit");var work=cache::empty_disk_work();assert(switch(cache::from_disk(ref store,ref changed,ref env,ref mut work)){case .ok(_):{false}case .err(e):{e.phase==2 && e.code==2}});let key=route(ref changed,ref env);raw_publish(ref store,ref key,payload::content(ref encoded));work=cache::empty_disk_work();assert(switch(cache::from_disk(ref store,ref changed,ref env,ref mut work)){case .ok(_):{false}case .err(e):{e.phase==3 && e.artifact.phase==4}});assert(count(bytes::view(ref path))==1);
        let state=load(bytes::view(ref base),b"State");let state_complete=cold(ref state);publish(ref store,ref state,ref state_complete,ref env);work=cache::empty_disk_work();let state_owner=restored(ref store,ref state,ref env,ref mut work);assert(interfaces::count(core::public_interfaces(ref state_owner))==3);
    }clear(bytes::view(ref path));
}
@effects(filesystem,clock,console) fn void cold_publish(ref u8[] root,ref u8[] leaf,ref u8[] path,ref environment::Environment env){let loaded=load(root,leaf);let complete=cold(ref loaded);let store=store(path);publish(ref store,ref loaded,ref complete,env);}
@effects(filesystem,console) fn core::Complete detached(ref u8[] root,ref u8[] leaf,ref u8[] path,ref environment::Environment env){let loaded=load(root,leaf);let store=store(path);var work=cache::empty_disk_work();return restored(ref store,ref loaded,env,ref mut work);}
@effects(console) fn c::Bundle emit(core::Complete complete){let program=checked::ir_program(core::checked_program(ref complete));assert(interfaces::count(core::public_interfaces(ref complete))==4);var usize i=1;var usize entry=0;while(i<=ir::count(program)){let f=clean::frame_at(ir::cleanup_ir(program),i);if(f.module_index==0 && f.kind==3){assert(entry==0);entry=i;}i+=1;}assert(entry>0);return switch(complete_c::partition_entry(complete,entry,128)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return emit(detached_for_error());}};}
@effects(console) fn core::Complete detached_for_error(){assert(false);return detached_for_error();}
@effects(filesystem,console) fn c::Bundle cold_bundle(ref u8[] root,ref environment::Environment env){let loaded=load(root,b"Vec");return emit(cold(ref loaded));}
@effects(console) fn void output(ref u8[] raw){assert(io::write_bytes(raw).is_ok);}
@effects(console) fn void member(ref u8[] raw){assert(io::write_usize(raw.len).is_ok);output(b"\n");output(raw);}
@test @effects(filesystem,environment,clock,console) fn void detached_disk_vec_owner_matches_cold_packet_and_executes_without_relowering(){
    let base=root();let env=current();let path=join(bytes::view(ref base),b"cache-vec");mkdir(bytes::view(ref path));cold_publish(bytes::view(ref base),b"Vec",bytes::view(ref path),ref env);let p=detached(bytes::view(ref base),b"Vec",bytes::view(ref path),ref env);assert(ir::work(checked::ir_program(core::checked_program(ref p)))==0);clear(bytes::view(ref path));let bundle=emit(p);let cold=cold_bundle(bytes::view(ref base),ref env);assert(complete_c::has_complete(ref bundle) && bytes::equal(c::project_header(ref bundle),c::project_header(ref cold)) && bytes::equal(c::project_literals(ref bundle),c::project_literals(ref cold)) && c::project_unit_count(ref bundle)==c::project_unit_count(ref cold));var usize i=0;while(i<c::project_unit_count(ref bundle)){assert(bytes::equal(c::project_source(ref bundle,i),c::project_source(ref cold,i)));i+=1;}
    output(b"FYCPARTS1\n");member(c::project_header(ref bundle));member(c::project_literals(ref bundle));assert(io::write_usize(c::project_unit_count(ref bundle)).is_ok);output(b"\n");i=0;while(i<c::project_unit_count(ref bundle)){member(c::project_source(ref bundle,i));i+=1;}
}
