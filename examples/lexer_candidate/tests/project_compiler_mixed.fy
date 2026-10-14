module tests::project_compiler_mixed;
import project_compiler_cached as compiler;
import project_compiler_core as core;
import project_compiler_input as input;
import project_coordinate_graph as coordinates;
import project_module_visibility as visibility;
import project_source_set as sources;
import project_cache_artifact as artifacts;
import project_cache_disk as disk;
import project_cache_environment as environment;
import generic_signature_diagnostics as diagnostics;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import windows_rooted_file as rooted;
import file_source as files;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;
import std::process as process;
import std::vec as v;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var p=bytes::empty();assert(bytes::append(ref mut p,root).is_ok && bytes::append_byte(ref mut p,47).is_ok && bytes::append(ref mut p,leaf).is_ok);return p;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(p):{p}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn text::CString cstr(ref u8[] raw){let s=switch(text::validate_utf8(raw)){case .ok(p):{p}case .err(_):{assert(false);return cstr(raw);}};return switch(text::cstring_from_text(s)){case .ok(p):{p}case .err(_):{assert(false);cstr(raw)}};}
@effects(filesystem) fn void mkdir(ref u8[] path){let s=cstr(path);assert(fs::create_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem) fn disk::Store store(ref u8[] path){return switch(disk::open(path,disk::Limits{.entries=64,.bytes=67108864,.record=16777216,.temporary_age=1})){case .ok(p):{p}case .err(_):{assert(false);store(path)}};}
@effects(filesystem) fn void clear(ref u8[] path){{let directory=switch(rooted::open(path)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;let entries=switch(rooted::scan(ref directory,4096,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;while(i<rooted::count(ref entries)){assert(switch(rooted::erase_if(ref directory,rooted::name(ref entries,i),rooted::metadata(ref entries,i))){case .ok(p):{p}case .err(_):{false}});i+=1;}}let s=cstr(path);assert(fs::remove_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem) fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);load(root,leaf)}};}
@effects(filesystem,environment) fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{assert(false);current()}};}


@effects(filesystem,environment,clock,console)
fn compiler::Observation compiled(ref u8[] root,ref u8[] leaf,ref disk::Store store,ref environment::Environment env){
    let loaded=load(root,leaf);var usize projection=0;let table=switch(coordinates::collect(visibility::workspace_graph(ref loaded),ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return compiler::empty();}};let sources=switch(input::project(ref loaded,ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return compiler::empty();}};
    var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();var cache_work=artifacts::empty_disk_work();var usize source_work=0;var usize link_work=0;var observed=compiler::empty();
    let complete=switch(compiler::build(store,env,ref loaded,ref sources,ref table,ref mut reports,ref mut cache_work,ref mut source_work,ref mut link_work,ref mut observed)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);assert(false);return observed;}};
    assert(ir::count(identified::ir_program(core::checked_program(ref complete)))>0 && observed.stored || observed.path==2);return observed;
}
@test @effects(filesystem,environment,clock,console)
fn void private_body_edit_rebuilds_one_package_and_reuses_two(){let base=root();let env=current();let path=join(bytes::view(ref base),b"private-cache");mkdir(bytes::view(ref path));{
    let store=store(bytes::view(ref path));let cold=compiled(bytes::view(ref base),b"Base",ref store,ref env);assert(cold.path==1 && cold.rebuilt_units==3 && cold.stored_units==3);let hit=compiled(bytes::view(ref base),b"Base",ref store,ref env);assert(hit.path==2);
    let changed=compiled(bytes::view(ref base),b"Private",ref store,ref env);assert(changed.path==3 && changed.reused_units==2 && changed.rebuilt_units==1 && changed.stored_units==1);let again=compiled(bytes::view(ref base),b"Private",ref store,ref env);assert(again.path==2);
    let root_edit=compiled(bytes::view(ref base),b"Root",ref store,ref env);assert(root_edit.path==3 && root_edit.reused_units==2 && root_edit.rebuilt_units==1);
}clear(bytes::view(ref path));}
@test @effects(filesystem,environment,clock,console)
fn void public_contract_and_generic_body_changes_invalidate_consumers(){let base=root();let env=current();let path=join(bytes::view(ref base),b"contract-cache");mkdir(bytes::view(ref path));{
    let store=store(bytes::view(ref path));discard compiled(bytes::view(ref base),b"Base",ref store,ref env);let changed=compiled(bytes::view(ref base),b"Public",ref store,ref env);assert(changed.path==3 && changed.reused_units==1 && changed.rebuilt_units==2);
    discard compiled(bytes::view(ref base),b"Generic",ref store,ref env);let generic=compiled(bytes::view(ref base),b"GenericEdit",ref store,ref env);assert(generic.path==3 && generic.reused_units>=1 && generic.rebuilt_units>=1);
}clear(bytes::view(ref path));}
@test @effects(filesystem,environment,clock,console)
fn void invalid_source_keeps_exact_diagnostics_and_publishes_no_unit_prefix(){let base=root();let env=current();let path=join(bytes::view(ref base),b"error-cache");mkdir(bytes::view(ref path));{
    let store=store(bytes::view(ref path));discard compiled(bytes::view(ref base),b"Base",ref store,ref env);let loaded=load(bytes::view(ref base),b"Invalid");var usize projection=0;let table=switch(coordinates::collect(visibility::workspace_graph(ref loaded),ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return;}};let sources=switch(input::project(ref loaded,ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();var usize source_work=0;var usize link_work=0;let expected=switch(core::build_complete_reporting(ref sources,ref table,ref mut reports,ref mut source_work,ref mut link_work)){case .ok(_):{assert(false);core::empty()}case .err(e):{e}};source_work=0;link_work=0;var cache_work=artifacts::empty_disk_work();var observed=compiler::empty();
    let actual=switch(compiler::build(ref store,ref env,ref loaded,ref sources,ref table,ref mut reports,ref mut cache_work,ref mut source_work,ref mut link_work,ref mut observed)){case .ok(_):{assert(false);core::empty()}case .err(e):{e}};assert(expected.phase==actual.phase && expected.stage==actual.stage && expected.code==actual.code && expected.module_index==actual.module_index && expected.start==actual.start && expected.end==actual.end && expected.expected==actual.expected && expected.actual==actual.actual && !observed.stored && observed.stored_units==0);let hit=compiled(bytes::view(ref base),b"Base",ref store,ref env);assert(hit.path==2);
}clear(bytes::view(ref path));}
