module tests::project_compiler_entry;
import project_compiler_entry as entry;
import project_compiler_input as input;
import project_compiler_core as core;
import project_workspace_lock as loader;
import project_source_set as sources;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import syntax as syntax;
import file_source as fixture;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,leaf).is_ok);return out;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] path=[0;4096];let count=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(fixture::manifest(view[..count])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem) fn input::Input projected(ref u8[] path){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;let loaded=switch(loader::load(path,true,ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return projected(path);}};return switch(input::project(loader::workspace(ref loaded),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return projected(path);}};}
@effects(console) fn ir::Program compiled(ref input::Input sources){return switch(core::build(sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return compiled(sources);}};}
@test @effects(filesystem,console) fn void root_main_entry_ignores_helper_and_dependency_main_and_matches_go(){
    let parent=root();let path=join(bytes::view(ref parent),b"Root");let sources=projected(bytes::view(ref path));let p=compiled(ref sources);var usize work=0;let id=switch(entry::select(ref sources,ref p,ref mut work)){case .ok(n):{n}case .err(e):{discard io::write_stderr_u32(e.kind);assert(false);return;}};
    let frame=clean::frame_at(ir::cleanup_ir(ref p),id);let original=input::origin(ref sources,frame.module_index);let parsed=input::source(ref sources,frame.module_index);let module_decl=syntax::declaration_at(parsed,0);let declaration=syntax::declaration_at(parsed,frame.declaration-1);
    assert(original.owner==0 && frame.module_index!=0 && bytes::equal(syntax::source_view(parsed)[module_decl.name_start..module_decl.name_end],b"main") && bytes::equal(syntax::source_view(parsed)[declaration.name_start..declaration.name_end],b"main"));
    assert(work>0);work=1048576;switch(entry::select(ref sources,ref p,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==5);}};
}
@test @effects(filesystem,console) fn void missing_or_invalid_root_entry_rejects_without_selecting_provider(){
    let parent=root();var usize mode=0;while(mode<5){let name=if(mode==0){b"Missing"}else if(mode==1){b"WrongModule"}else if(mode==2){b"WrongResult"}else if(mode==3){b"Parameter"}else{b"Generic"};let path=join(bytes::view(ref parent),name);let sources=projected(bytes::view(ref path));let p=compiled(ref sources);var usize work=0;
        switch(entry::select(ref sources,ref p,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==if(mode==1){1}else if(mode==2 || mode==3){3}else{2});if(e.kind==3){let parsed=input::source(ref sources,e.module_index);assert(e.start<e.end && e.end<=syntax::source_view(parsed).len && input::origin(ref sources,e.module_index).owner==0);}}};
        mode+=1;
    }
}
