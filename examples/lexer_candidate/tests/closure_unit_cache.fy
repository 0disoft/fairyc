module tests::closure_unit_cache;
import project_compiler_input as input;
import project_module_visibility as visibility;
import project_coordinate_graph as coordinates;
import project_source_set as source_set;
import project_source_plan as planning;
import project_cache_environment as environment;
import project_cache_disk as disk;
import project_cache_unit_key as keys;
import project_cache_unit_store as cache;
import generic_source_plan as plan;
import generic_aggregate_types as aggregates;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as table;
import module_project_compile as compile;
import module_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_program_artifact_wire as wire;
import candidate_artifact_wire as payload;
import module_unit_c_emit as c;
import syntax as syntax;
import std::bytes as bytes;
import std::process as process;
import std::vec as v;
import std::io as io;

@effects(console) fn void require(bool ok,u32 point){if(!ok){discard io::write_stderr("closure-unit-cache checkpoint=");discard io::write_stderr_u32(point);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{require(false,1);copied(raw)}};}
@effects(console) fn bytes::Buffer argument(usize index){var u8[4096] raw=[0;4096];let n=switch(process::arg(index,ref mut raw)){case .ok(p):{p}case .err(_):{require(false,2);0}};let ref u8[] view=ref raw;return copied(view[..n]);}
@effects(filesystem,console) fn visibility::Resolved loaded(ref u8[] root){var source_set::Budget budget=source_set::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(root,ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{require(false,3);loaded(root)}};}
@effects(filesystem,environment,console) fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{require(false,4);current()}};}
@effects(filesystem,console) fn disk::Store opened(ref u8[] path){return switch(disk::open(path,disk::Limits{.entries=16,.bytes=16777216,.record=8388608,.temporary_age=1})){case .ok(p):{p}case .err(_):{require(false,5);opened(path)}};}
resource Prepared{signatures::Program metadata;v::Vec<plan::Plan> ordinary;cleanup{}}
@effects(console) fn Prepared prepared(ref input::Input sources,ref table::Coordinates table,bool aggregate){
    var usize work=0;let original=switch(if(aggregate){planning::build_aggregate_sources(input::sources(sources),input::owners(sources),input::bindings(sources),table,ref mut work)}else{planning::build(sources,table,ref mut work)}){case .ok(p):{p}case .err(_):{require(false,6);return prepared(sources,table,aggregate);}};
    var ordinary=v::empty<plan::Plan>();
    let metadata=if(aggregate){let retained=switch(aggregates::build(original,input::sources(sources),ref mut work)){case .ok(p):{p}case .err(_):{require(false,7);return prepared(sources,table,aggregate);}};switch(signatures::build_aggregates(input::sources(sources),retained,ref mut work)){case .ok(p):{p}case .err(_):{require(false,8);return prepared(sources,table,aggregate);}}}else{require(v::push<plan::Plan>(ref mut ordinary,original).is_ok,9);let scope=switch(input::declarations(sources)){case .ok(p):{p}case .err(_):{require(false,10);return prepared(sources,table,aggregate);}};switch(signatures::build_catalog(input::sources(sources),scope)){case .ok(p):{p}case .err(_):{require(false,11);return prepared(sources,table,aggregate);}}};
    return Prepared{.metadata,.ordinary};
}
@effects(console) fn ref plan::Plan original(ref Prepared p){return if(signatures::aggregate_types(ref p.metadata)){aggregates::source_plan(signatures::aggregate_source(ref p.metadata))}else{v::at<plan::Plan>(ref p.ordinary,0)};}
@effects(console) fn identified::Program cold(ref input::Input sources,ref Prepared p){var usize work=0;return switch(compile::range_identified(input::sources(sources),ref p.metadata,original(p),0,1,ref mut work)){case .ok(p):{p}case .err(_):{require(false,12);cold(sources,p)}};}
@effects(console) fn u32 rejects(ref u8[] raw,ref keys::Identity key,ref input::Input sources,ref catalog::Catalog scope,ref Prepared p){var usize source=0;var usize artifact=0;return switch(cache::restore_closures(raw,key,input::sources(sources),scope,ref p.metadata,original(p),0,1,ref mut source,ref mut artifact)){case .ok(_):{0}case .err(code):{code}};}

// Assertions execute only in Native. Source checks retain every authored branch.
@effects(console) fn void controls(ref input::Input sources,ref catalog::Catalog scope,ref Prepared p,ref environment::Environment env,ref keys::Identity key,ref identified::Program unit){
    var usize work=0;let scalar=switch(keys::collect(sources,original(p),env,0,1,ref mut work)){case .ok(p):{p}case .err(_):{require(false,13);return;}};work=0;let aggregate=switch(keys::collect_aggregates(sources,original(p),env,0,1,ref mut work)){case .ok(p):{p}case .err(_):{require(false,14);return;}};work=0;let opposite=switch(keys::collect_closures(sources,original(p),env,0,1,!keys::aggregates(key),ref mut work)){case .ok(p):{p}case .err(_):{require(false,15);return;}};
    require(keys::closures(key) && !keys::closures(ref scalar) && !keys::closures(ref aggregate),16);require(!bytes::equal(keys::route(key),keys::route(ref scalar)) && !bytes::equal(keys::route(key),keys::route(ref aggregate)) && !bytes::equal(keys::route(key),keys::route(ref opposite)),17);require(!bytes::equal(keys::route(ref scalar),keys::route(ref aggregate)),18);
    work=0;require(switch(cache::encode(key,unit,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}},19);work=0;require(switch(cache::encode_closures(ref scalar,unit,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}},20);work=0;require(switch(cache::encode_closures(ref aggregate,unit,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}},21);
    work=0;let saved=switch(cache::encode_closures(key,unit,ref mut work)){case .ok(p):{p}case .err(_):{require(false,22);return;}};let raw=payload::content(ref saved);require(bytes::equal(raw[..8],b"FYCACHE2"),23);require(rejects(raw,key,sources,scope,p)==0 && rejects(raw,ref opposite,sources,scope,p)==1001 && rejects(raw,ref scalar,sources,scope,p)==1001 && rejects(raw,ref aggregate,sources,scope,p)==1001,24);
    var usize source=0;work=0;require(switch(cache::restore(raw,key,input::sources(sources),scope,ref p.metadata,original(p),0,1,ref mut source,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}},25);
    work=0;var cursor=switch(wire::cursor_marker(raw,b"FYCACHE2",ref mut work)){case .ok(p):{p}case .err(_):{require(false,26);return;}};let identity=wire::member_from(raw,ref mut cursor,16777216,b"FYPUNIT2",ref mut work);let program=wire::member_from(raw,ref mut cursor,wire::byte_limit(),b"FYPROG02",ref mut work);require(wire::end(raw,ref mut cursor,ref mut work)==0,27);
    var bad=copied(raw);bytes::view_mut(ref mut bad)[7]=48;require(rejects(bytes::view(ref bad),key,sources,scope,p)==1001,28);bytes::view_mut(ref mut bad)[7]=49;require(rejects(bytes::view(ref bad),key,sources,scope,p)==1001,29);
    bad=copied(raw);bytes::view_mut(ref mut bad)[identity.start+7]=49;require(rejects(bytes::view(ref bad),key,sources,scope,p)==1001,30);bad=copied(raw);bytes::view_mut(ref mut bad)[identity.start+15]=if(keys::aggregates(key)){0}else{1};require(rejects(bytes::view(ref bad),key,sources,scope,p)==1001,31);
    bad=copied(raw);bytes::view_mut(ref mut bad)[program.start+7]=49;require(rejects(bytes::view(ref bad),key,sources,scope,p)==1001,32);require(rejects(raw[..raw.len-1],key,sources,scope,p)==1001,33);bad=copied(raw);require(bytes::append_byte(ref mut bad,0).is_ok,34);require(rejects(bytes::view(ref bad),key,sources,scope,p)==1001,35);
    source=0;work=wire::work_limit()+1;require(switch(cache::restore_closures(raw,key,input::sources(sources),scope,ref p.metadata,original(p),0,1,ref mut source,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==wire::work_limit()+1 && source==0,36);
    work=0;require(!keys::collect_closures(sources,original(p),env,1,1,false,ref mut work).is_ok,37);work=wire::work_limit()+1;require(!keys::collect_closures(sources,original(p),env,0,1,false,ref mut work).is_ok,38);
}
@effects(console) fn void member(ref u8[] raw){require(io::write_usize(raw.len).is_ok && io::write("\n").is_ok && io::write_bytes(raw).is_ok,39);}
@effects(console) fn void emit(ref identified::Program unit,ref input::Input sources){let raw=identified::ir_program(unit);var usize entry=0;var usize at=1;while(at<=ir::count(raw)){let frame=clean::frame_at(ir::cleanup_ir(raw),at);if(frame.kind==3 && bytes::equal(syntax::declaration_name(input::source(sources,frame.module_index),frame.declaration),b"main")){require(entry==0,40);entry=at;}at+=1;}require(entry!=0,41);let output=switch(c::emit_closures(raw,entry,2,true)){case .ok(p):{p}case .err(_):{require(false,42);return;}};require(io::write("FYCPARTS1\n").is_ok,43);member(c::emission_header(ref output));member(c::emission_literals(ref output));require(io::write_usize(c::emission_unit_count(ref output)).is_ok && io::write("\n").is_ok,44);at=0;while(at<c::emission_unit_count(ref output)){member(c::emission_source(ref output,at));at+=1;}}

@effects(filesystem,environment,clock,console) pub fn i32 run(){
    let root=argument(1);let path=argument(2);let mode_bytes=argument(3);let mode=bytes::view(ref mode_bytes);let feature=argument(4);let aggregate=bytes::equal(bytes::view(ref feature),b"aggregate");let resolved=loaded(bytes::view(ref root));var usize work=0;let table=switch(coordinates::collect(visibility::workspace_graph(ref resolved),ref mut work)){case .ok(p):{p}case .err(_):{require(false,45);return 1;}};let sources=switch(input::project(ref resolved,ref mut work)){case .ok(p):{p}case .err(_):{require(false,46);return 1;}};require(input::count(ref sources)==1,47);let p=prepared(ref sources,ref table,aggregate);let scope=switch(input::declarations(ref sources)){case .ok(p):{p}case .err(_):{require(false,48);return 1;}};let env=current();let store=opened(bytes::view(ref path));work=0;let key=switch(keys::collect_closures(ref sources,original(ref p),ref env,0,1,aggregate,ref mut work)){case .ok(p):{p}case .err(_):{require(false,49);return 1;}};
    var history=cache::empty();if(bytes::equal(mode,b"exhausted")){history.artifact=wire::work_limit()+1;}if(bytes::equal(mode,b"io")){history.io=disk::work_limit();}
    let restored=cache::load_closures(ref store,ref key,input::sources(ref sources),ref scope,ref p.metadata,original(ref p),0,1,ref mut history);
    if(bytes::equal(mode,b"invalid")){require(!restored.is_ok,50);var usize source=0;require(!compile::range_identified(input::sources(ref sources),ref p.metadata,original(ref p),0,1,ref mut source).is_ok,51);return 0;}
    let expected_hit=bytes::equal(mode,b"warm");let unit=switch(restored){case .ok(p):{require(expected_hit && history.source>0 && history.artifact>0,52);p}case .err(code):{require(!expected_hit,53);if(bytes::equal(mode,b"exhausted")){require(code==1002 && history.artifact==wire::work_limit()+1,54);}else{require(code==1001,55);}cold(ref sources,ref p)}};
    let fresh=cold(ref sources,ref p);var usize a=0;var usize b=0;let actual=switch(cache::encode_closures(ref key,ref unit,ref mut a)){case .ok(p):{p}case .err(_):{require(false,56);return 1;}};let expected=switch(cache::encode_closures(ref key,ref fresh,ref mut b)){case .ok(p):{p}case .err(_):{require(false,57);return 1;}};require(bytes::equal(payload::content(ref actual),payload::content(ref expected)),58);require(clean::closures(ir::cleanup_ir(identified::ir_program(ref unit))) && !clean::project(ir::cleanup_ir(identified::ir_program(ref unit))),59);
    if(bytes::equal(mode,b"cold")){controls(ref sources,ref scope,ref p,ref env,ref key,ref unit);}if(bytes::equal(mode,b"exhausted") || bytes::equal(mode,b"io")){require(!cache::save_closures(ref store,ref key,ref unit,ref mut history),60);}else{require(cache::save_closures(ref store,ref key,ref unit,ref mut history),61);}
    if(aggregate){require(io::write("closure aggregate unit cache checked\n").is_ok,62);}else{emit(ref unit,ref sources);}return 0;
}
