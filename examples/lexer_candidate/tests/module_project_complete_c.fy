module tests::module_project_complete_c;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import module_project_interfaces as interfaces;
import module_generic_interface as templates;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import module_unit_c_emit as c;
import module_unit_c_scalar as scalar;
import module_project_complete_c as complete_c;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
// Drop all parser, coordinate and planning inputs before the bundle is built.
fn core::Complete detached_source(ref u8[] raw){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok && v::push<usize>(ref mut owners,0).is_ok);var bytes::Buffer names=bytes::empty();assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize i=0;while(i<64){assert(bytes::append_byte(ref mut names,49).is_ok);i+=1;}var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref names)}).is_ok);let table=coordinates::assemble(names,rows);var usize work=0;var usize linking=0;return switch(core::build_complete_sources(ref sources,ref owners,ref imports,ref table,ref mut work,ref mut linking)){case .ok(p):{p}case .err(_):{assert(false);detached_source(raw)}};}
fn core::Complete detached(){return detached_source(b"module main;pub fn T id<T>(T value){return value;}fn i32 main(){return id<i32>(42);}fn i32 unused(){return 99;}");}
fn usize entry(ref core::Complete p){let program=checked::ir_program(core::checked_program(p));var usize i=1;while(i<=ir::count(program)){if(clean::frame_at(ir::cleanup_ir(program),i).declaration==3){return i;}i+=1;}assert(false);return 0;}
fn bool contains(ref u8[] raw,ref u8[] part){var usize i=0;while(i<=raw.len){if(part.len<=raw.len-i && bytes::equal(raw[i..i+part.len],part)){return true;}i+=1;}return false;}
@test fn void complete_partition_retains_detached_interfaces_and_named_entry_closure(){let result=detached();let selected=entry(ref result);let records=core::public_interfaces(ref result);assert(interfaces::count(records)==1);let record=interfaces::at(records,0);assert(templates::count(interfaces::templates(record))==1);let expected=switch(bytes::copy(interfaces::source_hash(record))){case .ok(p):{p}case .err(_):{assert(false);return;}};let packed=switch(complete_c::partition_entry(result,selected,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(complete_c::has_complete(ref packed) && c::project_selected_frames(ref packed)==2 && c::project_frames(ref packed)==3 && c::project_unit_count(ref packed)==3);let retained=core::public_interfaces(complete_c::complete(ref packed));assert(interfaces::count(retained)==1 && interfaces::owner(interfaces::at(retained,0))==0 && bytes::equal(interfaces::source_hash(interfaces::at(retained,0)),bytes::view(ref expected)));assert(contains(c::project_header(ref packed),b"fy_fn_g0_"));var usize definitions=0;var usize i=0;while(i<c::project_unit_count(ref packed)){if(contains(c::project_source(ref packed,i),b"fy_fn_g0_")){definitions+=1;}i+=1;}assert(definitions>=1);}
@test fn void complete_partition_rejects_invalid_entry_and_frame_limit(){assert(switch(complete_c::partition(detached(),0,1)){case .ok(_):{false}case .err(e):{e.code==1001}});assert(switch(complete_c::partition_entry(detached(),1,0)){case .ok(_):{false}case .err(e):{e.code==1001}});}

fn usize void_entry(ref core::Complete p,usize ordinal){let program=checked::ir_program(core::checked_program(p));var usize i=1;var usize found=0;while(i<=ir::count(program)){let frame=clean::frame_at(ir::cleanup_ir(program),i);if(scalar::scalar(program,frame.result)==7){if(found==ordinal){return i;}found+=1;}i+=1;}assert(false);return 0;}
fn void same_emission(ref c::Emission a,ref c::Emission b){assert(bytes::equal(c::emission_header(a),c::emission_header(b)) && bytes::equal(c::emission_literals(a),c::emission_literals(b)));assert(c::emission_frames(a)==c::emission_frames(b) && c::emission_selected_frames(a)==c::emission_selected_frames(b) && c::emission_unit_count(a)==c::emission_unit_count(b));assert(c::emission_work(a)==c::emission_work(b) && c::emission_retries(a)==c::emission_retries(b));var usize i=0;while(i<c::emission_unit_count(a)){assert(c::emission_first(a,i)==c::emission_first(b,i) && c::emission_stop(a,i)==c::emission_stop(b,i) && bytes::equal(c::emission_source(a,i),c::emission_source(b,i)));i+=1;}}
@test fn void native_test_emission_reuses_checked_ir_and_owns_output(){
    let result=detached_source(b"module main;pub fn T id<T>(T value){return value;}fn i32 main(){return 0;}fn void alpha(){assert(id<i32>(42)==42);}fn void beta(){assert(true);}");let first=void_entry(ref result,0);let second=void_entry(ref result,1);assert(first!=second);
    let a=switch(complete_c::emit_test(ref result,first,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let b=switch(complete_c::emit_test(ref result,second,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(c::emission_selected_frames(ref a)>c::emission_selected_frames(ref b));
    assert(switch(complete_c::emit_test(ref result,0,1)){case .ok(_):{false}case .err(e):{e.code==1001}});
    let again=switch(complete_c::emit_test(ref result,first,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};same_emission(ref a,ref again);
    // Moving the original Complete proves these outputs carry no borrow of it.
    let packed=switch(complete_c::partition_test(result,first,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(complete_c::has_complete(ref packed));same_emission(ref a,c::project_emission(ref packed));
}
