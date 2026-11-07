module tests::composite_result_forward_call;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import module_unit_c_emit as c;
import module_project_complete_c as complete_c;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import static_operation_declarations as operations;
import c_text as w;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn coordinates::Coordinates table(){let raw=b"fypc1-sha256-1111111111111111111111111111111111111111111111111111111111111111";let names=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return table();}};var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=raw.len}).is_ok);return coordinates::assemble(names,rows);}
fn Result<core::Complete,core::Problem> compiled(ref u8[] raw){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);
    var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);let imports=v::empty<catalog::Binding>();let table=table();var usize work=0;var usize linking=0;
    return core::build_complete_sources(ref sources,ref owners,ref imports,ref table,ref mut work,ref mut linking);
}
@effects(console) fn void member(ref u8[] raw){assert(io::write_usize(raw.len).is_ok && io::write("\n").is_ok && io::write_bytes(raw).is_ok);}
// Provider calls use the complete partition, including private ordinary bodies.
@effects(console) pub fn i32 emit_forward(ref u8[] raw){
    let result=switch(compiled(raw)){case .ok(p):{p}case .err(e):{
        assert(io::write_stderr("phase=").is_ok && io::write_stderr_u32(e.phase).is_ok && io::write_stderr(" stage=").is_ok && io::write_stderr_u32(e.stage).is_ok && io::write_stderr(" code=").is_ok && io::write_stderr_u32(e.code).is_ok && io::write_stderr(" start=").is_ok && io::write_stderr_usize(e.start).is_ok && io::write_stderr("\n").is_ok);
        return 4;
    }};let parsed=parsed(raw);let program=checked::ir_program(core::checked_program(ref result));var usize entry=0;var usize i=1;var bytes::Buffer bridges=bytes::empty();var w::State status=w::State{.code=0};assert(w::text(ref mut bridges,ref mut status,b"#include \"fy_project.h\"\n"));
    while(i<=ir::count(program)){let frame=clean::frame_at(ir::cleanup_ir(program),i);if(frame.module_index==0){let d=syntax::declaration_at(ref parsed,frame.declaration-1);let name=syntax::source_view(ref parsed)[d.name_start..d.name_end];if(bytes::equal(name,b"main")){entry=i;}if(d.entry_count==1 && name.len>6 && bytes::equal(name[..6],b"probe_")){let type=operations::scalar(syntax::type_text(ref parsed,d.result_type));assert(type!=0);assert(w::type(ref mut bridges,ref mut status,type) && w::text(ref mut bridges,ref mut status,b" fairyc_") && w::text(ref mut bridges,ref mut status,name) && w::text(ref mut bridges,ref mut status,b"(") && w::type(ref mut bridges,ref mut status,type) && w::text(ref mut bridges,ref mut status,b" value){return fy_fn_") && w::number(ref mut bridges,ref mut status,i) && w::text(ref mut bridges,ref mut status,b"(value); }\n"));}}i+=1;}assert(entry!=0);
    let bundle=switch(complete_c::partition(result,entry,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);return 4;}};
    assert(io::write("FYCPARTS1\n").is_ok);member(c::project_header(ref bundle));member(c::project_literals(ref bundle));assert(io::write_usize(c::project_unit_count(ref bundle)).is_ok && io::write("\n").is_ok);i=0;while(i<c::project_unit_count(ref bundle)){if(i+1==c::project_unit_count(ref bundle)){var bytes::Buffer tail=bytes::empty();assert(bytes::append(ref mut tail,c::project_source(ref bundle,i)).is_ok && bytes::append(ref mut tail,bytes::view(ref bridges)).is_ok);member(bytes::view(ref tail));}else{member(c::project_source(ref bundle,i));}i+=1;}return 0;
}

@effects(console) pub fn i32 stdin_forward(){
    var bytes::Buffer raw=bytes::empty();var u8[4096] chunk=[0;4096];
    while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 4;}};if(n==0){break;}let ref u8[] content=ref chunk;assert(bytes::len(ref raw)<=32768-n && bytes::append(ref mut raw,content[..n]).is_ok);}
    return emit_forward(bytes::view(ref raw));
}
