module tests::module_unit_artifact_admission;
import module_unit_artifact_admission as admission;
import module_project_compile as compile;
import module_identified_ir as identified;
import module_program_artifact_encode as encoder;
import module_program_artifact_wire as budget;
import candidate_artifact_wire as wire;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_source_plan as planning;
import generic_source_plan as plan;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn bytes::Buffer copy(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn syntax::Parsed parsed(ref u8[] raw){let scanned=switch(scanner::collect(copy(raw),false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};return switch(syntax::parse(scanned,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(b"module empty;")}};}
pub resource Fixture{v::Vec<syntax::Parsed> sources;catalog::Catalog scope;signatures::Program metadata;plan::Plan plan;cleanup{}}
pub fn Fixture fixture(u32 mode,bool generic){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();let root=if(generic){b"module root;import helper as h;fn i32 main(){return h::id<i32>(42);}"}else if(mode==2){b"module root;import helper as h;fn i32 main(){return h::get()+1;}"}else{b"module root;import helper as h;fn i32 main(){return h::get();}"};
    let dependency=if(generic){if(mode==1){b"module helper;pub fn T id<T>(T value){return value+0;}"}else{b"module helper;pub fn T id<T>(T value){return value;}"}}else if(mode==1){b"module helper;pub fn i32 get(){return 43;}"}else if(mode==3){b"module helper;pub fn u32 get(){return 42;}"}else{b"module helper;pub fn i32 get(){return 42;}"};
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(dependency)).is_ok);
    var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok && v::push<usize>(ref mut owners,1).is_ok);var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok);
    let text=copy(b"fypc1-sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaafypc1-sha256-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=77}).is_ok && v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=1,.start=77,.end=154}).is_ok);let table=coordinates::assemble(text,rows);
    let scope=switch(catalog::build_scoped(ref sources,ref owners,ref imports)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0,false);}};let scope2=switch(catalog::build_scoped(ref sources,ref owners,ref imports)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0,false);}};
    let metadata=switch(signatures::build_catalog(ref sources,scope2)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0,false);}};var usize work=0;let original=switch(planning::build_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return fixture(0,false);}};return Fixture{.sources,.scope,.metadata,.plan=original};
}
pub fn wire::Payload saved(ref Fixture p,usize first,usize stop){var usize work=0;let unit=switch(compile::range_identified(ref p.sources,ref p.metadata,ref p.plan,first,stop,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return saved(p,first,stop);}};var usize artifact=0;return switch(encoder::encode(identified::ir_program(ref unit),ref mut artifact)){case .ok(p):{p}case .err(_):{assert(false);saved(p,first,stop)}};}
pub fn bool accepts(ref u8[] raw,ref Fixture p,usize first,usize stop){var usize source=0;var usize artifact=0;return admission::restore(raw,ref p.sources,ref p.scope,ref p.metadata,ref p.plan,first,stop,ref mut source,ref mut artifact).is_ok;}
@test @effects(console) fn void private_dependency_edit_reuses_caller_but_rejects_changed_owner_and_interface(){let old=fixture(0,false);let caller=saved(ref old,0,1);let dependency=saved(ref old,1,2);let changed=fixture(1,false);var usize source=0;var usize artifact=0;switch(admission::restore(wire::content(ref caller),ref changed.sources,ref changed.scope,ref changed.metadata,ref changed.plan,0,1,ref mut source,ref mut artifact)){case .ok(_):{}case .err(code):{discard io::write_stderr("caller ");discard io::write_stderr_u32(code);assert(false);}};assert(!accepts(wire::content(ref dependency),ref changed,1,2));let own=fixture(2,false);assert(!accepts(wire::content(ref caller),ref own,0,1));let interface=fixture(3,false);assert(!accepts(wire::content(ref caller),ref interface,0,1));}
@test fn void generic_body_changes_and_extra_frames_cannot_acquire_reuse_authority(){let old=fixture(0,true);let caller=saved(ref old,0,1);assert(accepts(wire::content(ref caller),ref old,0,1));let changed=fixture(1,true);assert(!accepts(wire::content(ref caller),ref changed,0,1));let plain=fixture(0,false);let extra=saved(ref plain,0,2);assert(!accepts(wire::content(ref extra),ref plain,0,1));}
@test fn void malformed_and_exhausted_units_publish_no_owner(){let p=fixture(0,false);let data=saved(ref p,0,1);let raw=wire::content(ref data);assert(!accepts(raw[..raw.len-1],ref p,0,1));var usize source=0;var usize artifact=budget::work_limit()+1;assert(switch(admission::restore(raw,ref p.sources,ref p.scope,ref p.metadata,ref p.plan,0,1,ref mut source,ref mut artifact)){case .ok(_):{false}case .err(code):{code==1002}});assert(artifact==budget::work_limit()+1);}
