module tests::project_compiler_core;
import project_compiler_core as core;
import project_compiler_input as input;
import project_workspace_lock as loader;
import project_source_set as sources;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import syntax as syntax;
import file_source as fixture;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,root).is_ok && bytes::append_byte(ref mut output,47).is_ok && bytes::append(ref mut output,leaf).is_ok);return output;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] path=[0;4096];let count=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(fixture::manifest(view[..count])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let count=number(data,at);assert(*at<=data.len && count<=data.len-*at);let start=*at;*at+=count;return data[start..*at];}
@effects(filesystem)
fn input::Input owned(ref u8[] path){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let loaded=switch(loader::load(path,true,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return owned(path);}};return switch(input::project(loader::workspace(ref loaded),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return owned(path);}};}
@effects(console)
fn void failed(ref u8[] name,core::Problem p){discard io::write_stderr_bytes(name);discard io::write_stderr(" phase ");discard io::write_stderr_u32(p.phase);discard io::write_stderr(" stage ");discard io::write_stderr_u32(p.stage);discard io::write_stderr(" code ");discard io::write_stderr_u32(p.code);discard io::write_stderr(" at ");discard io::write_stderr_usize(p.start);discard io::write_stderr("..");discard io::write_stderr_usize(p.end);}
@test
@effects(filesystem,console)
fn void go_whole_closure_outcomes_spans_instances_and_sdk_match(){let parent=root();let path=join(bytes::view(ref parent),b"cases.bin");let oracle=switch(fixture::manifest(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let data=bytes::view(ref oracle);assert(data.len>=16 && bytes::equal(data[..8],b"FYCORE01"));var usize at=8;let count=number(data,ref mut at);assert(count==8);var usize index=0;
    while(index<count){let name=field(data,ref mut at);let file=field(data,ref mut at);let excerpt=field(data,ref mut at);let instance=field(data,ref mut at);let exit=number(data,ref mut at);let start=number(data,ref mut at);let end=number(data,ref mut at);let phase=number(data,ref mut at);let stage=number(data,ref mut at);let frames=number(data,ref mut at);let root_path=join(bytes::view(ref parent),name);let projected=owned(bytes::view(ref root_path));
        switch(core::build(ref projected)){
            case .ok(p):{assert(exit==0);if(ir::count(ref p)!=frames){discard io::write_stderr_bytes(name);discard io::write_stderr_usize(ir::count(ref p));assert(false);}assert(proof::check(ref p).code==0);assert(input::count(ref projected)==if(bytes::equal(name,b"SDK")){13}else{6});}
            case .err(p):{if(exit==0 || p.phase as usize!=phase || p.stage as usize!=stage || p.start!=start || p.end!=end){failed(name,p);assert(false);}assert(p.code!=0 && core::has_location(ref projected,p));assert(bytes::equal(input::file(ref projected,p.module_index),file) && bytes::equal(input::instance_id(ref projected,p.module_index),instance));let source=syntax::source_view(input::source(ref projected,p.module_index));assert(bytes::equal(source[p.start..p.end],excerpt));if(bytes::equal(name,b"RootBody") || bytes::equal(name,b"DependencyBody") || bytes::equal(name,b"Nominal")){assert(p.expected!=0 && p.actual!=0 && p.expected!=p.actual);}if(bytes::equal(name,b"Moved")){assert(p.code==1101);}}
        };index+=1;
    }assert(at==data.len);
}
@effects(filesystem,console)
fn ir::Program detached(ref u8[] path){let projected=owned(path);return switch(core::build(ref projected)){case .ok(value):{value}case .err(p):{failed(b"detached",p);assert(false);return detached(path);}};}
@test
@effects(filesystem,console)
fn void linked_ir_survives_source_and_workspace_owners_closing(){let parent=root();let path=join(bytes::view(ref parent),b"Current");let p=detached(bytes::view(ref path));assert(ir::count(ref p)==13 && proof::check(ref p).code==0);assert(ir::layout_count(ref p)==3 && ir::field_count(ref p)==3);var bool root=false;var bool left=false;var bool right=false;var usize i=0;
    while(i<ir::field_count(ref p)){let name=ir::field_name(ref p,i);root=root || bytes::equal(name,b"root");left=left || bytes::equal(name,b"left");right=right || bytes::equal(name,b"right");i+=1;}assert(root && left && right);var usize owned=0;i=1;while(i<=ir::count(ref p)){let frame=clean::frame_at(ir::cleanup_ir(ref p),i);assert(frame.module_index<6 && frame.declaration>=2);assert(ir::source_bytes(ir::at(ref p,i))>0);owned+=1;i+=1;}assert(owned==13);
}
