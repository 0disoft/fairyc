module tests::module_project_identified_vec_link;
import module_project_identified_compile as project;
import module_project_compile as compile;
import module_identified_ir as identified;
import module_project_identified_ir as linked;
import module_project_link as linking;
import module_project_inputs as inputs;
import module_unit_body_ir_verify as proof;
import module_unit_c_emit as c;
import candidate_limits as limits;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import generic_package_bindings as bindings;
import generic_source_catalog as templates;
import generic_source_requests as requests;
import generic_source_plan_input as prepared;
import generic_source_plan as plan;
import generic_package_context as context;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import std::fs as fs;
import std::text as utf8;
import std::process as process;

fn syntax::Parsed parsed(bytes::Buffer raw){let scanned=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};return switch(syntax::parse(scanned,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(bytes::empty())}};}
fn syntax::Parsed text(ref u8[] raw){return parsed(switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}});}
fn syntax::Parsed provider(ref u8[] name,bool invalid){var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module ").is_ok && bytes::append(ref mut raw,name).is_ok && bytes::append(ref mut raw,b";import helper as shared;import std::vec as vec;").is_ok);var u8 i=0;
    while(i<8){let u8[1] letter=[97+i];assert(bytes::append(ref mut raw,b"fn T ").is_ok && bytes::append(ref mut raw,ref letter[..]).is_ok && bytes::append(ref mut raw,b"<T>(T n){return n;}").is_ok);i+=1;}
    assert(bytes::append(ref mut raw,b"pub fn i32 run(){let vec::Vec<i32> items=vec::empty<i32>();discard items;").is_ok);i=0;
    while(i<8){let u8[1] letter=[97+i];var usize type=0;while(type<5){assert(bytes::append(ref mut raw,b"discard ").is_ok && bytes::append(ref mut raw,ref letter[..]).is_ok);let call=if(type==0){b"<bool>(false);"}else if(type==1){b"<u8>(1);"}else if(type==2){b"<u32>(1);"}else if(type==3){b"<i32>(1);"}else{b"<usize>(1);"};assert(bytes::append(ref mut raw,call).is_ok);type+=1;}i+=1;}
    assert(bytes::append(ref mut raw,if(invalid){b"discard shared::id<i32>(1);return true;}"}else{b"return shared::id<i32>(21);}"}).is_ok);return parsed(raw);
}
@effects(console)
fn plan::Plan planned(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"fypc1-sha256-").is_ok);var usize i=0;while(i<64){assert(bytes::append_byte(ref mut text,49).is_ok);i+=1;}var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref text)}).is_ok);let coordinates=coordinates::assemble(text,rows);
    var v::Vec<usize> owners=v::empty<usize>();var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();i=0;while(i<v::len<syntax::Parsed>(sources)){assert(v::push<usize>(ref mut owners,0).is_ok);let source=v::at<syntax::Parsed>(sources,i);var usize j=1;while(j<syntax::declaration_count(source)){let d=syntax::declaration_at(source,j);if(d.kind==2){var usize measured=0;let target=signatures::import_target_measured(metadata,i,syntax::source_view(source)[d.name_start..d.name_end],ref mut measured);assert(target.code==0 && target.item>0);assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=i,.declaration=j+1,.target=target.item-1}).is_ok);}j+=1;}i+=1;}
    var context::Budget work=context::empty();let bound=switch(bindings::build(sources,ref owners,ref imports,ref coordinates,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,metadata);}};
    let catalog=switch(templates::build(bound,sources,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,metadata);}};
    let roots=switch(requests::build(catalog,sources,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,metadata);}};
    let input=switch(prepared::build(roots,sources,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.source.code);assert(false);return planned(sources,metadata);}};
    return switch(plan::build(input,plan::defaults(),ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);planned(sources,metadata)}};
}

@effects(filesystem,console)
fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;let name=switch(utf8::validate_utf8(view[..size])){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};var fs::File file=switch(fs::open_read_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};var bytes::Buffer data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,1048577).is_ok);var usize used=0;while(true){let n=switch(fs::read(ref mut file,bytes::view_mut(ref mut data)[used..])){case .ok(n):{n}case .err(_):{assert(false);return data;}};used+=n;assert(used<=1048576);if(n==0){break;}}assert(fs::close(file).is_ok && bytes::truncate(ref mut data,used).is_ok);return data;}
@effects(filesystem,console)
fn v::Vec<syntax::Parsed> source_set(){let sdk=fixture();var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut out,text(b"module root;import left as a;import right as b;fn i32 main(){return a::run()+b::run();}")).is_ok && v::push<syntax::Parsed>(ref mut out,provider(b"left",false)).is_ok && v::push<syntax::Parsed>(ref mut out,provider(b"right",false)).is_ok && v::push<syntax::Parsed>(ref mut out,text(b"module helper;pub fn T id<T>(T n){return n;}")).is_ok && v::push<syntax::Parsed>(ref mut out,text(bytes::view(ref sdk))).is_ok);return out;}
@effects(filesystem,console)
fn linked::Program compiled(){let sources=source_set();let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return compiled();}};let original=planned(ref sources,ref metadata);var usize work=0;let units=switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(b"compile=");discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);return compiled();}};assert(project::count(ref units)>1);var usize repeated=0;var usize unit=0;while(unit<project::count(ref units)){let owner=project::at(ref units,unit);let ir=identified::ir_program(owner);var usize i=1;while(i<=ir::count(ir)){let f=clean::frame_at(ir::cleanup_ir(ir),i);if(f.kind==2){let t=clean::type_at(ir::cleanup_ir(ir),f.receiver_type);if(t.kind==13 && clean::type_at(ir::cleanup_ir(ir),t.first).scalar==1){repeated+=1;}}i+=1;}unit+=1;}assert(repeated>1);work=0;return switch(linked::build(ref metadata,ref units,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(b"link=");discard io::write_stderr_u32(e.code);assert(false);compiled()}};}
@test @effects(filesystem,console)
fn void shared_vec_type_cleanup_and_call_ids_survive_project_link(){let owned=compiled();let program=linked::ir_program(ref owned);assert(proof::check(program).code==0);var usize vectors=0;var usize i32_vectors=0;var usize i32_cleanup=0;var usize cleanup_count=0;var usize i=1;
    while(i<=clean::type_count(ir::cleanup_ir(program))){let t=clean::type_at(ir::cleanup_ir(program),i);let row=linked::type_at(ref owned,i);let id=linked::identity(ref owned,row);assert((row.node!=0)==(t.kind==13));if(t.kind==13){assert(id.len==67 && bytes::equal(id[..3],b"g0_"));vectors+=1;if(clean::type_at(ir::cleanup_ir(program),t.first).scalar==1){i32_vectors+=1;}let glue=clean::glue_at(ir::cleanup_ir(program),i);assert(glue.frame>0 && bytes::equal(id,linked::identity(ref owned,linked::frame_at(ref owned,glue.frame))));}i+=1;}
    i=1;while(i<=ir::count(program)){let f=clean::frame_at(ir::cleanup_ir(program),i);if(f.kind==2 && clean::type_at(ir::cleanup_ir(program),f.receiver_type).kind==13){cleanup_count+=1;let t=clean::type_at(ir::cleanup_ir(program),f.receiver_type);if(clean::type_at(ir::cleanup_ir(program),t.first).scalar==1){i32_cleanup+=1;}}let body=ir::at(program,i);var usize j=0;while(j<ir::call_count(body)){let call=ir::call_at(body,j);if(call.kind==2){var usize work=0;let target=switch(linked::call_frame(ref owned,i,j,ref mut work)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(target>0 && call.instance==target && linked::frame_at(ref owned,target).node!=0);}j+=1;}i+=1;}
    assert(vectors>=1 && cleanup_count==vectors && i32_vectors==1 && i32_cleanup==1);
}
@test @effects(filesystem,console)
fn void vec_link_native_transport(){let owned=compiled();let program=linked::ir_program(ref owned);var usize entry=0;var usize i=1;while(i<=ir::count(program)){let f=clean::frame_at(ir::cleanup_ir(program),i);if(f.module_index==0 && f.kind==3){assert(entry==0);entry=i;}i+=1;}assert(entry>0);var bytes::Buffer out=bytes::empty();let rendered=c::render(program,entry,ref mut out);if(rendered.code!=0){discard io::write_stderr_u32(rendered.code);}assert(rendered.code==0 && bytes::len(ref out)>1000);assert(io::write_bytes(bytes::view(ref out)).is_ok);}
