module tests::project_compiler_cached;
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

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var p=bytes::empty();assert(bytes::append(ref mut p,root).is_ok && bytes::append_byte(ref mut p,47).is_ok && bytes::append(ref mut p,leaf).is_ok);return p;}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(p):{p}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn text::CString cstr(ref u8[] raw){let s=switch(text::validate_utf8(raw)){case .ok(p):{p}case .err(_):{assert(false);return cstr(raw);}};return switch(text::cstring_from_text(s)){case .ok(p):{p}case .err(_):{assert(false);cstr(raw)}};}
@effects(filesystem) fn void mkdir(ref u8[] path){let s=cstr(path);assert(fs::create_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem) fn disk::Store store(ref u8[] path){return switch(disk::open(path,disk::Limits{.entries=8,.bytes=8388608,.record=8388608,.temporary_age=1})){case .ok(p):{p}case .err(_):{assert(false);store(path)}};}
@effects(filesystem) fn void clear(ref u8[] path){{let directory=switch(rooted::open(path)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;let entries=switch(rooted::scan(ref directory,4096,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;while(i<rooted::count(ref entries)){assert(switch(rooted::erase_if(ref directory,rooted::name(ref entries,i),rooted::metadata(ref entries,i))){case .ok(p):{p}case .err(_):{false}});i+=1;}}let s=cstr(path);assert(fs::remove_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem) fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;return switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);load(root,leaf)}};}
@effects(filesystem,environment) fn environment::Environment current(){var usize work=0;return switch(environment::current(ref mut work)){case .ok(p):{p}case .err(_):{assert(false);current()}};}

@test @effects(filesystem,environment,clock)
fn void cold_then_disk_hit_skips_lowering_and_budget_failure_falls_back(){
    let base=root();let env=current();let path=join(bytes::view(ref base),b"compiler-cache");mkdir(bytes::view(ref path));
    {let loaded=load(bytes::view(ref base),b"Plain");let store=store(bytes::view(ref path));var usize projection=0;
        let table=switch(coordinates::collect(visibility::workspace_graph(ref loaded),ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        let sources=switch(input::project(ref loaded,ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();var cache_work=artifacts::empty_disk_work();var usize source_work=0;var usize link_work=0;var observed=compiler::empty();
        {let cold=switch(compiler::build(ref store,ref env,ref loaded,ref sources,ref table,ref mut reports,ref mut cache_work,ref mut source_work,ref mut link_work,ref mut observed)){case .ok(p):{p}case .err(_):{assert(false);return;}};
            assert(observed.path==1 && observed.stored && observed.lookup_phase==2 && observed.lookup_code==2 && source_work>0 && ir::count(identified::ir_program(core::checked_program(ref cold)))>0);}
        cache_work=artifacts::empty_disk_work();source_work=0;link_work=0;
        {let hit=switch(compiler::build(ref store,ref env,ref loaded,ref sources,ref table,ref mut reports,ref mut cache_work,ref mut source_work,ref mut link_work,ref mut observed)){case .ok(p):{p}case .err(_):{assert(false);return;}};
            assert(observed.path==2 && !observed.stored && source_work==0 && link_work==0 && cache_work.source>0 && ir::work(identified::ir_program(core::checked_program(ref hit)))==0);}
        cache_work=artifacts::empty_disk_work();cache_work.io=disk::work_limit()+1;
        let fallback=switch(compiler::build(ref store,ref env,ref loaded,ref sources,ref table,ref mut reports,ref mut cache_work,ref mut source_work,ref mut link_work,ref mut observed)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(observed.path==1 && !observed.stored && observed.lookup_code==9 && observed.store_code==9 && source_work>0 && ir::count(identified::ir_program(core::checked_program(ref fallback)))>0 && v::len<diagnostics::Reports>(ref reports)==0);
    }clear(bytes::view(ref path));
}

@test @effects(filesystem,environment,clock)
fn void cache_miss_preserves_exact_cold_compiler_error(){
    let base=root();let env=current();let path=join(bytes::view(ref base),b"compiler-errors");mkdir(bytes::view(ref path));
    {let loaded=load(bytes::view(ref base),b"InvalidBody");let store=store(bytes::view(ref path));var usize projection=0;
        let table=switch(coordinates::collect(visibility::workspace_graph(ref loaded),ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        let sources=switch(input::project(ref loaded,ref mut projection)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        var v::Vec<diagnostics::Reports> reports=v::empty<diagnostics::Reports>();var usize source_work=0;var usize link_work=0;
        let expected=switch(core::build_complete_reporting(ref sources,ref table,ref mut reports,ref mut source_work,ref mut link_work)){case .ok(_):{assert(false);core::empty()}case .err(e):{e}};
        source_work=0;link_work=0;var cache_work=artifacts::empty_disk_work();var observed=compiler::empty();
        let actual=switch(compiler::build(ref store,ref env,ref loaded,ref sources,ref table,ref mut reports,ref mut cache_work,ref mut source_work,ref mut link_work,ref mut observed)){case .ok(_):{assert(false);core::empty()}case .err(e):{e}};
        assert(actual.phase==expected.phase && actual.stage==expected.stage && actual.code==expected.code && actual.module_index==expected.module_index && actual.start==expected.start && actual.end==expected.end && actual.expected==expected.expected && actual.actual==expected.actual && observed.path==1 && !observed.stored && observed.store_phase==0);
    }clear(bytes::view(ref path));
}
