module tests::project_emit_cached;
import project_emit as emit;
import project_emit_text as diagnostics;
import project_compiler_cached as cached;
import project_cache_disk as disk;
import module_unit_c_emit as c;
import project_lock_policy as locks;
import windows_rooted_file as rooted;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::fs as fs;
import std::text as text;

fn disk::Limits limits(){return disk::Limits{.entries=8,.bytes=8388608,.record=8388608,.temporary_age=1};}
@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn bytes::Buffer join(ref u8[] base,ref u8[] leaf){var p=bytes::empty();assert(bytes::append(ref mut p,base).is_ok && bytes::append_byte(ref mut p,47).is_ok && bytes::append(ref mut p,leaf).is_ok);return p;}
fn text::CString cstr(ref u8[] raw){let s=switch(text::validate_utf8(raw)){case .ok(p):{p}case .err(_):{assert(false);return cstr(raw);}};return switch(text::cstring_from_text(s)){case .ok(p):{p}case .err(_):{assert(false);cstr(raw)}};}
@effects(filesystem) fn void mkdir(ref u8[] path){let s=cstr(path);assert(fs::create_dir(text::cstring_as_cstr(ref s)).is_ok);}
@effects(filesystem) fn void clear(ref u8[] path){{let directory=switch(rooted::open(path)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;let entries=switch(rooted::scan(ref directory,4096,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;while(i<rooted::count(ref entries)){assert(switch(rooted::erase_if(ref directory,rooted::name(ref entries,i),rooted::metadata(ref entries,i))){case .ok(p):{p}case .err(_):{false}});i+=1;}}let s=cstr(path);assert(fs::remove_dir(text::cstring_as_cstr(ref s)).is_ok);}
fn void same(ref emit::Plan a,ref emit::Plan b){assert(bytes::equal(emit::package_name(a),emit::package_name(b)) && locks::state_code(emit::lock_state(a))==locks::state_code(emit::lock_state(b)));let x=emit::output_bundle(a);let y=emit::output_bundle(b);assert(bytes::equal(c::project_header(x),c::project_header(y)) && bytes::equal(c::project_literals(x),c::project_literals(y)) && c::project_unit_count(x)==c::project_unit_count(y));var usize i=0;while(i<c::project_unit_count(x)){assert(bytes::equal(c::project_source(x,i),c::project_source(y,i)));i+=1;}}
@effects(filesystem,environment,clock) fn void retained(ref u8[] base,ref u8[] leaf,ref u8[] cache_leaf){let project=join(base,leaf);let path=join(base,cache_leaf);mkdir(bytes::view(ref path));{
    {let initialized=switch(emit::evaluate(bytes::view(ref project),false)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(c::project_unit_count(emit::output_bundle(ref initialized))>0);}
    let expected=switch(emit::evaluate(bytes::view(ref project),true)){case .ok(p):{p}case .err(_):{assert(false);return;}};var observed=cached::empty();
    {let cold=switch(emit::evaluate_cached(bytes::view(ref project),true,bytes::view(ref path),limits(),ref mut observed)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(observed.path==1 && observed.stored);same(ref expected,ref cold);}
    {let hit=switch(emit::evaluate_cached(bytes::view(ref project),true,bytes::view(ref path),limits(),ref mut observed)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(observed.path==2 && !observed.stored);same(ref expected,ref hit);}
    let missing=join(bytes::view(ref path),b"missing");let fallback=switch(emit::evaluate_cached(bytes::view(ref project),true,bytes::view(ref missing),limits(),ref mut observed)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(observed.path==1 && !observed.stored && observed.lookup_phase==7);same(ref expected,ref fallback);
}clear(bytes::view(ref path));}
@test @effects(filesystem,environment,clock)
fn void cached_emission_retains_exact_cold_packet_and_identity(){let base=root();retained(bytes::view(ref base),b"Plain",b"plain-cache");retained(bytes::view(ref base),b"GenericVec",b"vec-cache");}
@effects(filesystem,environment,clock) fn void rejected(ref u8[] base,ref u8[] leaf,ref u8[] cache_leaf,u32 phase){let project=join(base,leaf);let path=join(base,cache_leaf);mkdir(bytes::view(ref path));{
    let expected=switch(emit::evaluate(bytes::view(ref project),false)){case .ok(_):{assert(false);return;}case .err(e):{e}};var observed=cached::empty();
    let actual=switch(emit::evaluate_cached(bytes::view(ref project),false,bytes::view(ref path),limits(),ref mut observed)){case .ok(_):{assert(false);return;}case .err(e):{e}};
    assert(emit::phase(ref actual)==phase && emit::phase(ref actual)==emit::phase(ref expected) && diagnostics::exit_code(ref actual)==diagnostics::exit_code(ref expected));var a=bytes::empty();var b=bytes::empty();assert(diagnostics::error(ref expected,ref mut a) && diagnostics::error(ref actual,ref mut b) && bytes::equal(bytes::view(ref a),bytes::view(ref b)));assert(observed.path==1 && observed.stored==(phase==2));
}clear(bytes::view(ref path));}
@test @effects(filesystem,environment,clock)
fn void cached_emission_preserves_frontend_and_entry_failures(){let base=root();rejected(bytes::view(ref base),b"InvalidBody",b"error-cache",1);rejected(bytes::view(ref base),b"InvalidEntry",b"entry-cache",2);}
