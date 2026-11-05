module tests::generic_aggregate_disk_cache;
import project_cache_artifact as whole;
import project_cache_keys as whole_keys;
import project_cache_artifact_wire as whole_wire;
import project_cache_lookup as lookup;
import module_program_artifact_wire as artifact_wire;
import project_source_file as files;
import project_compiler_core as core;
import project_compiler_input as input;
import project_coordinate_graph as coordinates;
import module_public_type_text as module_public_type_text;
import project_module_visibility as visibility;
import project_source_set as source_set;
import project_cache_environment as environment;
import project_cache_disk as disk;
import project_cache_unit_store as cache;
import project_cache_unit_key as keys;
import module_project_identified_compile as units;
import module_project_identified_ir as checked;
import module_identified_ir as unit;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_project_complete_c as complete_c;
import module_unit_c_emit as c;
import module_project_interfaces as interfaces;
import module_public_interface as codec;
import module_state_layout as state;
import project_source_plan as planning;
import generic_aggregate_types as aggregates;
import module_signatures as signatures;
import candidate_artifact_wire as payload;
import generic_signature_diagnostics as diagnostics;
import std::process as process;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn void failure(core::Problem e){discard io::write_stderr("unit cache ");discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);assert(false);}
fn bytes::Buffer argument(usize index){var u8[4096] raw=[0;4096];let n=switch(process::arg(index,ref mut raw)){case .ok(p):{p}case .err(_):{assert(false);0}};let ref u8[] view=ref raw;return switch(bytes::copy(view[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,console) fn visibility::Resolved loaded(ref u8[] root){var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(root,ref mut budget,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("unit cache load ");discard io::write_stderr_u32(visibility::load_phase(ref e));assert(false);loaded(root)}};}
@effects(filesystem,environment) fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{assert(false);current()}};}
@effects(filesystem) fn disk::Store store(ref u8[] path){return switch(disk::open(path,disk::Limits{.entries=16,.bytes=16777216,.record=8388608,.temporary_age=1})){case .ok(p):{p}case .err(_):{assert(false);store(path)}};}
@effects(console) fn core::Complete cold(ref input::Input source,ref module_public_type_text::Coordinates table,bool ordinary){var usize a=0;var usize b=0;return switch(if(ordinary){core::build_complete_sources(input::sources(source),input::owners(source),input::bindings(source),table,ref mut a,ref mut b)}else{core::build_complete_aggregate_sources(input::sources(source),input::owners(source),input::bindings(source),table,ref mut a,ref mut b)}){case .ok(p):{p}case .err(e):{failure(e);cold(source,table,ordinary)}};}
fn usize entry(ref core::Complete complete,ref input::Input source){let p=checked::ir_program(core::checked_program(complete));var usize i=1;var usize result=0;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.kind==3 && input::origin(source,f.module_index).owner==0){assert(result==0);result=i;}i+=1;}assert(result!=0);return result;}
fn c::Bundle unreachable(){assert(false);return unreachable();}
fn c::Bundle parts(core::Complete p,usize entry){return switch(complete_c::partition(p,entry,2)){case .ok(p):{p}case .err(_):{unreachable()}};}
fn void same_interfaces(ref core::Complete a,ref core::Complete b){let x=core::public_interfaces(a);let y=core::public_interfaces(b);assert(interfaces::count(x)==interfaces::count(y));let empty=state::empty();var usize i=0;while(i<interfaces::count(x)){let p=interfaces::at(x,i);let q=interfaces::at(y,i);assert(interfaces::owner(p)==interfaces::owner(q) && bytes::equal(interfaces::ordinary_hash(p),interfaces::ordinary_hash(q)) && bytes::equal(interfaces::source_hash(p),interfaces::source_hash(q)));var usize work=0;let px=switch(codec::encode(ref empty,interfaces::owner(p),interfaces::descriptor(p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let qx=switch(codec::encode(ref empty,interfaces::owner(q),interfaces::descriptor(q),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref px),bytes::view(ref qx)));i+=1;}}

// The independent full-source cold factory uses the same project grouping as
// whole-project restoration. Compare all interfaces and every emitted C byte.
@effects(console) fn void emit(core::Complete p,ref input::Input source,ref module_public_type_text::Coordinates table,bool ordinary){
 let fresh=cold(source,table,ordinary);same_interfaces(ref p,ref fresh);let at=entry(ref p,source);let expected_at=entry(ref fresh,source);let output=parts(p,at);let expected=parts(fresh,expected_at);
 assert(bytes::equal(c::project_header(ref output),c::project_header(ref expected)) && bytes::equal(c::project_literals(ref output),c::project_literals(ref expected)) && c::project_unit_count(ref output)==c::project_unit_count(ref expected));var usize i=0;while(i<c::project_unit_count(ref output)){assert(bytes::equal(c::project_source(ref output,i),c::project_source(ref expected,i)));i+=1;}
 assert(io::write("FYCPARTS1\n").is_ok);member(c::project_header(ref output));member(c::project_literals(ref output));assert(io::write_usize(c::project_unit_count(ref output)).is_ok && io::write("\n").is_ok);i=0;while(i<c::project_unit_count(ref output)){member(c::project_source(ref output,i));i+=1;}
}
@effects(console) fn void member(ref u8[] raw){assert(io::write_usize(raw.len).is_ok && io::write("\n").is_ok && io::write_bytes(raw).is_ok);}
fn Result<core::Complete,whole::Problem> restore(ref u8[] raw,ref visibility::Resolved loaded,ref environment::Environment env,bool ordinary){var usize a=0;var usize b=0;var usize c=0;var usize d=0;var usize e=0;return if(ordinary){whole::restore(raw,loaded,env,ref mut a,ref mut b,ref mut c,ref mut d,ref mut e)}else{whole::restore_aggregates(raw,loaded,env,ref mut a,ref mut b,ref mut c,ref mut d,ref mut e)};}
@effects(console) fn void controls(ref visibility::Resolved loaded,ref core::Complete p,ref environment::Environment env,bool ordinary){
 var usize key=0;var usize art=0;let data=switch(if(ordinary){whole::encode(loaded,p,env,ref mut key,ref mut art)}else{whole::encode_aggregates(loaded,p,env,ref mut key,ref mut art)}){case .ok(p):{p}case .err(e):{discard io::write_stderr("encode ");discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);assert(false);return;}};let raw=payload::content(ref data);assert(bytes::equal(raw[..8],if(ordinary){b"FYCACHE0"}else{b"FYCACHE1"}));
 art=0;let framing=switch(if(ordinary){whole_wire::preflight(raw,ref mut art)}else{whole_wire::preflight_aggregates(raw,ref mut art)}){case .ok(p):{p}case .err(_):{assert(false);return;}};let measured=art;assert(measured>0);art=artifact_wire::work_limit()-measured;assert(if(ordinary){whole_wire::preflight(raw,ref mut art).is_ok}else{whole_wire::preflight_aggregates(raw,ref mut art).is_ok} && art==artifact_wire::work_limit());art=artifact_wire::work_limit()-measured+1;assert(!if(ordinary){whole_wire::preflight(raw,ref mut art).is_ok}else{whole_wire::preflight_aggregates(raw,ref mut art).is_ok});
 assert(!restore(raw,loaded,env,!ordinary).is_ok);key=0;assert(!if(ordinary){whole_keys::collect_aggregates(loaded,p,env,ref mut key).is_ok}else{whole_keys::collect(loaded,p,env,ref mut key).is_ok});key=0;let a=switch(lookup::collect(loaded,env,ref mut key)){case .ok(p):{p}case .err(_):{assert(false);return;}};key=0;let b=switch(lookup::collect_aggregates(loaded,env,ref mut key)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(!bytes::equal(lookup::canonical(ref a),lookup::canonical(ref b)) && !bytes::equal(lookup::digest(ref a),lookup::digest(ref b)));
 var altered=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return;}};{let view=bytes::view_mut(ref mut altered);view[framing.keys.end-1]=view[framing.keys.end-1]^1;}let refusal=switch(restore(bytes::view(ref altered),loaded,env,ordinary)){case .ok(_):{assert(false);return;}case .err(e):{e}};assert(refusal.phase==6 && refusal.code==1001);
}
@effects(filesystem,environment,clock,console) pub fn i32 run(){
 let root=argument(1);let path=argument(2);let mode_bytes=argument(3);let mode=bytes::view(ref mode_bytes);let loaded=loaded(bytes::view(ref root));var usize projection=0;let table=switch(coordinates::collect(visibility::workspace_graph(ref loaded),ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};let source=switch(input::project(ref loaded,ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};assert(input::count(ref source)==3);let store=store(bytes::view(ref path));let env=current();let ordinary=bytes::equal(mode,b"ordinarycold") || bytes::equal(mode,b"ordinarywarm");let warm=bytes::equal(mode,b"warm") || bytes::equal(mode,b"ordinarywarm");var work=whole::empty_disk_work();if(bytes::equal(mode,b"exhausted")){work.io=disk::work_limit();}
 if(bytes::equal(mode,b"private") || bytes::equal(mode,b"generic")){var usize file_work=0;let old=switch(files::read(bytes::view(ref root),b"old.fyc",8388608,ref mut file_work)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};let refused=switch(restore(bytes::view(ref old),ref loaded,ref env,false)){case .ok(_):{assert(false);return 1;}case .err(e):{e}};assert(refused.phase==4);}
 if(bytes::equal(mode,b"invalid")){assert(!whole::from_disk_aggregates(ref store,ref loaded,ref env,ref mut work).is_ok);var usize a=0;var usize b=0;assert(!core::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut a,ref mut b).is_ok);return 0;}
 let selected=if(ordinary){whole::from_disk(ref store,ref loaded,ref env,ref mut work)}else{whole::from_disk_aggregates(ref store,ref loaded,ref env,ref mut work)};let p=switch(selected){case .ok(p):{assert(warm && work.source>0 && work.artifact>0 && work.link>0 && work.key>0 && ir::work(checked::ir_program(core::checked_program(ref p)))==0);p}case .err(_):{assert(!warm);let fresh=cold(ref source,ref table,ordinary);let saved=if(ordinary){whole::to_disk(ref store,ref loaded,ref fresh,ref env,ref mut work)}else{whole::to_disk_aggregates(ref store,ref loaded,ref fresh,ref env,ref mut work)};assert(saved.is_ok!=bytes::equal(mode,b"exhausted"));if(bytes::equal(mode,b"exhausted")){assert(work.io==disk::work_limit());}fresh}};
 if(bytes::equal(mode,b"cold") || bytes::equal(mode,b"ordinarycold")){controls(ref loaded,ref p,ref env,ordinary);}emit(p,ref source,ref table,ordinary);return 0;
}
