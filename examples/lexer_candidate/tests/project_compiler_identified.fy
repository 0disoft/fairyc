module tests::project_compiler_identified;
import project_compiler_core as core;
import project_compiler_input as input;
import project_workspace_lock as loader;
import project_source_set as sources;
import project_module_visibility as visibility;
import project_coordinate_graph as coordinates;
import module_public_type_text as table;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import candidate_limits as limits;
import syntax as syntax;
import file_source as fixture;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;
import module_project_interfaces as interfaces;
fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,root).is_ok && bytes::append_byte(ref mut output,47).is_ok && bytes::append(ref mut output,leaf).is_ok);return output;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] path=[0;4096];let count=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(fixture::manifest(view[..count])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let count=number(data,at);assert(*at<=data.len && count<=data.len-*at);let start=*at;*at+=count;return data[start..*at];}
@effects(console)
fn void failed(ref u8[] name,core::Problem p){discard io::write_stderr_bytes(name);discard io::write_stderr(" phase ");discard io::write_stderr_u32(p.phase);discard io::write_stderr(" stage ");discard io::write_stderr_u32(p.stage);discard io::write_stderr(" code ");discard io::write_stderr_u32(p.code);discard io::write_stderr(" at ");discard io::write_stderr_usize(p.start);discard io::write_stderr("..");discard io::write_stderr_usize(p.end);}

pub resource Loaded{input::Input sources;table::Coordinates coordinates;cleanup{}}
@effects(filesystem)
fn Loaded owned(ref u8[] path){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let loaded=switch(loader::load(path,true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{assert(false);return owned(path);}};
 let coordinates=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(p):{p}case .err(_):{assert(false);return owned(path);}};
 let sources=switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(_):{assert(false);return owned(path);}};return Loaded{.sources,.coordinates};}
fn bool same(core::Problem a,core::Problem b){return a.phase==b.phase && a.stage==b.stage && a.code==b.code && a.module_index==b.module_index && a.start==b.start && a.end==b.end && a.expected==b.expected && a.actual==b.actual && a.declaration_start==b.declaration_start && a.declaration_end==b.declaration_end && a.move_start==b.move_start && a.move_end==b.move_end && a.header_start==b.header_start && a.header_end==b.header_end && a.field_module==b.field_module && a.field_start==b.field_start && a.field_end==b.field_end;}
@test @effects(filesystem,console)
fn void whole_project_outcomes_preserve_go_spans_and_body_failures(){let parent=root();let path=join(bytes::view(ref parent),b"cases.bin");let oracle=switch(fixture::manifest(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};let data=bytes::view(ref oracle);assert(bytes::equal(data[..8],b"FYCORE01"));var usize at=8;let count=number(data,ref mut at);assert(count==8);var usize index=0;
 while(index<count){let name=field(data,ref mut at);let file=field(data,ref mut at);let excerpt=field(data,ref mut at);let instance=field(data,ref mut at);let exit=number(data,ref mut at);let start=number(data,ref mut at);let end=number(data,ref mut at);let phase=number(data,ref mut at);let stage=number(data,ref mut at);let frames=number(data,ref mut at);let root_path=join(bytes::view(ref parent),name);let loaded=owned(bytes::view(ref root_path));var usize source_work=0;var usize link_work=0;
  switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){
   case .ok(p):{assert(exit==0 && ir::count(identified::ir_program(ref p))==frames && proof::check(identified::ir_program(ref p)).code==0);assert(source_work>0 && link_work>0);switch(core::build(ref loaded.sources)){case .ok(old):{assert(ir::count(ref old)==frames);}case .err(e):{failed(name,e);assert(false);}};}
   case .err(p):{if(exit==0 || p.phase as usize!=phase || p.stage as usize!=stage || p.start!=start || p.end!=end){failed(name,p);assert(false);}assert(core::has_location(ref loaded.sources,p));assert(bytes::equal(input::file(ref loaded.sources,p.module_index),file) && bytes::equal(input::instance_id(ref loaded.sources,p.module_index),instance));assert(bytes::equal(syntax::source_view(input::source(ref loaded.sources,p.module_index))[p.start..p.end],excerpt));switch(core::build(ref loaded.sources)){case .ok(_):{assert(false);}case .err(old):{assert(same(p,old));}};assert(link_work==0);}
  };index+=1;
 }assert(at==data.len);
}
@effects(filesystem,console)
fn identified::Program detached(ref u8[] path){let loaded=owned(path);var usize source_work=0;var usize link_work=0;return switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{failed(b"detached",e);assert(false);return detached(path);}};}
@test @effects(filesystem,console)
fn void generic_vec_owner_survives_project_and_source_plan_closing(){let parent=root();let path=join(bytes::view(ref parent),b"Generic");let p=detached(bytes::view(ref path));let program=identified::ir_program(ref p);assert(proof::check(program).code==0);var usize named=0;var usize vector_types=0;var usize calls=0;var usize i=1;
 while(i<=identified::frame_count(ref p)){let row=identified::frame_at(ref p,i);let id=identified::identity(ref p,row);if(row.node!=0){assert(id.len==67 && bytes::equal(id[..3],b"g0_"));named+=1;}let body=ir::at(program,i);var usize j=0;while(j<ir::call_count(body)){let call=ir::call_at(body,j);if(call.kind==2){var usize work=0;let target=switch(identified::call_frame(ref p,i,j,ref mut work)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(target>0 && identified::frame_at(ref p,target).node!=0);calls+=1;}j+=1;}i+=1;}
 i=1;while(i<=clean::type_count(ir::cleanup_ir(program))){if(clean::type_at(ir::cleanup_ir(program),i).kind==13){let row=identified::type_at(ref p,i);assert(row.node!=0 && identified::identity(ref p,row).len==67);vector_types+=1;}i+=1;}assert(named>1 && calls>0 && vector_types>0);

}
@test @effects(filesystem,console)
fn void independent_work_limits_and_invalid_coordinates_publish_no_owner(){let parent=root();let path=join(bytes::view(ref parent),b"Current");let loaded=owned(bytes::view(ref path));var usize required_source=0;var usize required_link=0;
 switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut required_source,ref mut required_link)){case .ok(_):{}case .err(e):{failed(b"measure",e);assert(false);}};assert(required_source>0 && required_link>0);
 var usize source_work=16777216-required_source;var usize link_work=limits::project_work()-required_link;switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){case .ok(_):{assert(source_work==16777216 && link_work==limits::project_work());}case .err(e):{failed(b"exact",e);assert(false);}};
 source_work=16777216-required_source+1;link_work=0;switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==5 || e.phase==3 && e.stage==8);assert(link_work==0);}};
 source_work=0;link_work=limits::project_work()-required_link+1;switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==4 && e.code==1002);}};
 source_work=16777217;link_work=0;switch(core::build_identified(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==5 && e.stage==0 && e.code==10 && link_work==0);}};
 let missing=table::assemble(bytes::empty(),v::empty<table::Coordinate>());source_work=0;link_work=0;switch(core::build_identified(ref loaded.sources,ref missing,ref mut source_work,ref mut link_work)){case .ok(_):{assert(false);}case .err(e):{assert(e.phase==5 && e.code==9 && link_work==0);}};
}
@test @effects(filesystem,console)
fn void real_workspace_complete_result_owns_all_reachable_package_interfaces(){let parent=root();var usize case_index=0;while(case_index<3){let path=join(bytes::view(ref parent),if(case_index==0){b"Current"}else if(case_index==1){b"SDK"}else{b"Generic"});let loaded=owned(bytes::view(ref path));var usize source_work=0;var usize link_work=0;
 let result=switch(core::build_complete(ref loaded.sources,ref loaded.coordinates,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{failed(b"complete workspace",e);assert(false);return;}};let program=core::checked_program(ref result);assert(proof::check(identified::ir_program(program)).code==0);let packages=core::public_interfaces(ref result);var bool[512] owners=[false;512];var usize expected=0;var usize i=0;
 while(i<input::count(ref loaded.sources)){let owner=*v::at<usize>(input::owners(ref loaded.sources),i);if(!owners[owner]){owners[owner]=true;expected+=1;}i+=1;}assert(interfaces::count(packages)==expected);i=0;while(i<expected){let record=interfaces::at(packages,i);assert(owners[interfaces::owner(record)] && interfaces::ordinary_hash(record).len==77 && interfaces::source_hash(record).len==78);if(i>0){assert(interfaces::owner(interfaces::at(packages,i-1))<interfaces::owner(record));}i+=1;}
 if(case_index==0){assert(ir::count(identified::ir_program(program))==13);}if(case_index==1){assert(ir::count(identified::ir_program(program))==166);}case_index+=1;}}
