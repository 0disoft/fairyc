module tests::module_unit_ir_budgets;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as interfaces;
import module_types as types;
import module_project_compile as compile;
import module_signatures as signatures;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import file_source as files;
import std::process as process;
import std::io as io;
import std::bytes as bytes;
import std::vec as v;

fn Result<ir::Program,clean::Problem> empty(usize producer_work){
    var v::Vec<types::Type> graph=v::empty<types::Type>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    let cleaned=switch(clean::assemble(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return ir::assemble(cleaned,v::empty<ir::Body>(),v::empty<ir::Layout>(),v::empty<interfaces::Field>(),v::empty<interfaces::Callable>(),v::empty<usize>(),bytes::empty(),producer_work);
}
@test fn void owned_ir_revalidation_has_an_independent_bounded_budget(){
    let p=switch(empty(1048576)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::work(ref p)==1048576 && proof::check(ref p).code==0);
    switch(empty(1048577)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002);}};
}
fn Result<ir::Program,clean::Problem> empty_project(usize producer_work){
    var v::Vec<types::Type> graph=v::empty<types::Type>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    let cleaned=switch(clean::assemble_project(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),producer_work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return ir::assemble(cleaned,v::empty<ir::Body>(),v::empty<ir::Layout>(),v::empty<interfaces::Field>(),v::empty<interfaces::Callable>(),v::empty<usize>(),bytes::empty(),producer_work);
}
@test fn void project_ir_work_capacity_keeps_an_exact_ceiling_and_unit_budget(){
    assert(limits::project_frames()==16384 && limits::project_work()==16777216);var usize work=limits::project_work()-1;assert(limits::project_charge(ref mut work,1) && work==limits::project_work());assert(!limits::project_charge(ref mut work,1) && work==limits::project_work());
    let p=switch(empty_project(limits::project_work())){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::work(ref p)==limits::project_work() && proof::check(ref p).code==0);
    switch(empty_project(limits::project_work()+1)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002);}};
    switch(empty(1048577)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002);}};
}
@test @effects(filesystem,console) fn void original_bodies_unit_revalidates_with_all_main_sdk_sources(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<count){var u8[4096] path=[0;4096];let n=switch(process::arg(i+1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let b=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};let s=switch(scanner::collect(b,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)==0);assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);i+=1;}
    assert(bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,5))[7..13],b"bodies"));
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(compile::range(ref sources,ref metadata,5,6)){case .ok(p):{p}case .err(e):{discard io::write("stage ");discard io::write_u32(e.stage);discard io::write(" code ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};
    assert(ir::count(ref p)>0 && ir::work(ref p)<=1048576 && proof::check(ref p).code==0);discard io::write("bodies frames ");discard io::write_usize(ir::count(ref p));discard io::write(" producer work ");discard io::write_usize(ir::work(ref p));discard io::write("\n");
}
@test @effects(filesystem,console) fn void original_generic_expand_unit_compiles_without_repeated_type_scans(){
    let count=process::arg_count()-1;assert(count>200 && count<=limits::source_files());var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<count){var u8[4096] path=[0;4096];let n=switch(process::arg(i+1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);return;}};let b=switch(files::load(ref path[..n])){case .ok(b):{b}case .err(_):{assert(false);return;}};let s=switch(scanner::collect(b,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)==0);assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);i+=1;}
    assert(bytes::equal(syntax::source_view(v::at<syntax::Parsed>(ref sources,35))[7..21],b"generic_expand"));
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(compile::range(ref sources,ref metadata,35,36)){case .ok(p):{p}case .err(e):{discard io::write("stage ");discard io::write_u32(e.stage);discard io::write(" code ");discard io::write_u32(e.code);discard io::write("\n");assert(false);return;}};
    assert(ir::count(ref p)>0 && ir::work(ref p)<=1048576 && proof::check(ref p).code==0);discard io::write("generic_expand frames ");discard io::write_usize(ir::count(ref p));discard io::write(" producer work ");discard io::write_usize(ir::work(ref p));discard io::write("\n");
}
