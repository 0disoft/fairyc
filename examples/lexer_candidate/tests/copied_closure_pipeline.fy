module tests::copied_closure_pipeline;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import module_unit_c_emit as c;
import module_project_complete_c as complete_c;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_project_artifact_encode as encode;
import module_project_artifact_decode as decode;
import candidate_artifact_wire as wire;
import static_operation_declarations as operations;
import c_text as w;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn coordinates::Coordinates table(){var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<2){let raw=if(i==0){b"fypc1-sha256-1111111111111111111111111111111111111111111111111111111111111111"}else{b"fypc1-sha256-2222222222222222222222222222222222222222222222222222222222222222"};let start=bytes::len(ref names);assert(bytes::append(ref mut names,raw).is_ok && v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=i,.start,.end=bytes::len(ref names)}).is_ok);i+=1;}return coordinates::assemble(names,rows);}
@effects(console) fn core::Complete compiled(ref u8[] raw,ref u8[] sdk){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(sdk)).is_ok);
    var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok && v::push<usize>(ref mut owners,1).is_ok);var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok);let table=table();var usize work=0;var usize linking=0;
    let result=switch(core::build_complete_sources(ref sources,ref owners,ref imports,ref table,ref mut work,ref mut linking)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);discard io::write_stderr("/");discard io::write_stderr_usize(e.start);discard io::write_stderr("/");discard io::write_stderr_usize(e.end);assert(false);return compiled(raw,sdk);}};
    work=0;let saved=switch(encode::encode(core::checked_program(ref result),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("encode ");discard io::write_stderr_u32(code);assert(false);return compiled(raw,sdk);}};work=0;
    let detached=switch(decode::decode(wire::content(ref saved),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr("decode ");discard io::write_stderr_u32(code);assert(false);return compiled(raw,sdk);}};
    var usize source_work=0;var usize artifact_work=0;var usize link_work=0;
    return switch(core::restore_complete_sources(detached,ref sources,ref owners,ref imports,ref table,ref mut source_work,ref mut artifact_work,ref mut link_work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("restore ");discard io::write_stderr_u32(e.phase);discard io::write_stderr("/");discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);compiled(raw,sdk)}};
}
@effects(console) fn void member(ref u8[] raw){assert(io::write_usize(raw.len).is_ok && io::write("\n").is_ok && io::write_bytes(raw).is_ok);}
// Provider calls use the complete partition, including private ordinary bodies.
@effects(console) pub fn void emit_closures(ref u8[] raw,ref u8[] sdk){
    let result=compiled(raw,sdk);let parsed=parsed(raw);let program=checked::ir_program(core::checked_program(ref result));var usize entry=0;var usize i=1;var bytes::Buffer bridges=bytes::empty();var w::State status=w::State{.code=0};assert(w::text(ref mut bridges,ref mut status,b"#include \"fy_project.h\"\n"));
    while(i<=ir::count(program)){let frame=clean::frame_at(ir::cleanup_ir(program),i);if(frame.module_index==0){let d=syntax::declaration_at(ref parsed,frame.declaration-1);let name=syntax::source_view(ref parsed)[d.name_start..d.name_end];if(bytes::equal(name,b"main")){entry=i;}}i+=1;}assert(entry!=0);
    let bundle=switch(complete_c::partition(result,entry,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return;}};
    assert(io::write("FYCPARTS1\n").is_ok);member(c::project_header(ref bundle));member(c::project_literals(ref bundle));assert(io::write_usize(c::project_unit_count(ref bundle)).is_ok && io::write("\n").is_ok);i=0;while(i<c::project_unit_count(ref bundle)){if(i+1==c::project_unit_count(ref bundle)){var bytes::Buffer tail=bytes::empty();assert(bytes::append(ref mut tail,c::project_source(ref bundle,i)).is_ok && bytes::append(ref mut tail,bytes::view(ref bridges)).is_ok);member(bytes::view(ref tail));}else{member(c::project_source(ref bundle,i));}i+=1;}
}
